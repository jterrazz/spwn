package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/foundation/auth"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var (
	talkOutputFormat string
	talkWorldID      string
)

func init() {
	talkCmd.Flags().StringVar(&talkOutputFormat, "output-format", "", "Output format: text (default) or stream-json")
	talkCmd.Flags().StringVar(&talkWorldID, "world", "", "World ID to target (disambiguates when the same agent exists in multiple worlds)")
	Cmd.AddCommand(talkCmd)
}

var talkCmd = &cobra.Command{
	Use:   "talk <agent-name> [message]",
	Short: "Talk to a running agent — interactive or one-shot",
	Long: `Open a conversation with a named agent running inside a world.

If a message is provided, runs a one-shot query and prints the response.
If no message is provided, opens an interactive session inside the container.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		message := ""
		if len(args) > 1 {
			message = args[1]
		}
		s := newStepper(cmd)

		if err := agentDomain.ValidateMind(name); err != nil {
			return fmt.Errorf("agent %q not found\n\n  Create one with: spwn agent new %s", name, name)
		}

		containerID, worldID, world, arc, err := findAgentContainer(name, talkWorldID)
		if err != nil {
			return err
		}

		// Suppress stepper output in stream-json mode (observatory reads raw JSON)
		if talkOutputFormat != "stream-json" {
			s.Blank()
			s.Info("Agent:", name)
			s.Info("World:", worldID)
			s.Blank()
		}

		runtimeName := world.Runtime
		if runtimeName == "" {
			runtimeName = "claude-code"
		}

		// Look up existing session ID for this agent (enables conversation continuity)
		sessionID := ""
		if arc != nil {
			sessionID = arc.GetSessionID(worldID, name)
		}

		_ = world // world record kept for future use; runtime config no longer needs MindPath
		runtimeCmd, rtErr := universe.BuildRuntimeCommand(runtimeName, universe.RuntimeSpawnConfig{
			AgentName: name,
			WorldID:   worldID,
			Prompt:    message,
			SessionID: sessionID,
		})
		if rtErr != nil {
			return fmt.Errorf("unknown runtime %q for world %s", runtimeName, worldID)
		}

		// Configure output format for session ID capture
		// Claude: --print --output-format json → single JSON with session_id
		// Codex: --json is already in BuildCommand → JSONL with thread_id
		if runtimeName == "claude-code" && message != "" {
			if talkOutputFormat == "stream-json" {
				runtimeCmd = append(runtimeCmd, "--output-format", "stream-json", "--verbose")
			} else {
				runtimeCmd = append(runtimeCmd, "--print", "--output-format", "json")
			}
		}

		// Sync credentials before talking (updates bind-mounted /credentials/.env)
		_ = auth.SyncCredentials()

		buildDockerArgs := func(interactive bool) []string {
			agentHome := "/agents/" + name
			args := []string{"exec"}
			if interactive {
				args = append(args, "-it")
			}
			// Per-agent operational isolation: cwd + HOME + identity env
			// vars. Tools that respect $HOME (claude, git, ssh, shell
			// history, …) automatically land in this agent's persistent
			// home dir on the host.
			args = append(args,
				"-w", agentHome,
				"-e", "HOME="+agentHome,
				"-e", "SPWN_AGENT_NAME="+name,
				"-e", "SPWN_WORLD_ID="+worldID,
			)
			// Credentials still come from /credentials/.env via the
			// bind mount; no -e flags needed for them.
			args = append(args, containerID)
			return args
		}

		// Wrap runtime command to source credentials from bind-mounted directory
		wrapWithCredentials := func(cmd []string) []string {
			escaped := make([]string, len(cmd))
			for i, arg := range cmd {
				escaped[i] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
			}
			// Source env vars + set up runtime-specific credential files (symlinks)
			setup := "source /credentials/.env 2>/dev/null"
			// Codex: symlink auth.json from credentials dir to ~/.codex/
			setup += "; [ -f /credentials/openai/auth.json ] && mkdir -p $HOME/.codex && ln -sf /credentials/openai/auth.json $HOME/.codex/auth.json 2>/dev/null"
			shellCmd := setup + "; exec " + strings.Join(escaped, " ")
			return []string{"bash", "-c", shellCmd}
		}

		if message != "" {
			dockerArgs := append(buildDockerArgs(false), wrapWithCredentials(runtimeCmd)...)
			execCmd := exec.Command("docker", dockerArgs...)

			// persistSession is called once we discover the runtime's session
			// identifier on stdout. It is intentionally idempotent — for streamed
			// runs we'll be called per-line and will only update on first capture.
			var captured string
			persistSession := func(id string) {
				if id == "" || id == captured || arc == nil {
					return
				}
				captured = id
				_ = arc.SetSessionID(worldID, name, id)
			}

			if talkOutputFormat == "stream-json" {
				// Streaming mode (used by the observatory): tee stdout so we
				// can both forward each line to the caller AND scan it for the
				// runtime's session/thread id. Without this scan, every
				// observatory message starts a fresh conversation — the #1
				// reported "agent forgets" bug.
				stdoutPipe, pipeErr := execCmd.StdoutPipe()
				if pipeErr != nil {
					return fmt.Errorf("stdout pipe: %w", pipeErr)
				}
				// Suppress stderr in streaming mode (codex emits noisy MCP errors)
				execCmd.Stderr = nil
				if err := execCmd.Start(); err != nil {
					return formatExecError(err, nil)
				}
				scanner := bufio.NewScanner(stdoutPipe)
				scanner.Buffer(make([]byte, 1024*1024), 4*1024*1024)
				for scanner.Scan() {
					line := scanner.Bytes()
					// Forward verbatim so the observatory's SSE relay still
					// sees the original event stream byte-for-byte.
					_, _ = os.Stdout.Write(line)
					_, _ = os.Stdout.Write([]byte{'\n'})
					if id := extractSessionID(runtimeName, line); id != "" {
						persistSession(id)
					}
				}
				if err := execCmd.Wait(); err != nil {
					return formatExecError(err, nil)
				}
				return nil
			}

			// Non-streaming mode: capture stdout and stderr separately so
			// stray stderr lines (warnings, MCP boot noise) cannot corrupt
			// the JSON parse on stdout.
			var stdoutBuf, stderrBuf bytes.Buffer
			execCmd.Stdout = &stdoutBuf
			execCmd.Stderr = &stderrBuf
			err := execCmd.Run()
			output := stdoutBuf.Bytes()
			if err != nil {
				combined := append([]byte{}, output...)
				combined = append(combined, stderrBuf.Bytes()...)
				return formatExecError(err, combined)
			}

			// Parse response based on runtime to extract session ID and text
			switch runtimeName {
			case "claude-code":
				var resp struct {
					Result    string `json:"result"`
					SessionID string `json:"session_id"`
				}
				if jsonErr := json.Unmarshal(output, &resp); jsonErr == nil {
					persistSession(resp.SessionID)
					fmt.Fprintln(os.Stdout, resp.Result)
				} else {
					// Fallback: even if the wrapper JSON failed to parse,
					// scan the raw output for an embedded session_id so we
					// don't silently lose continuity.
					if id := extractSessionID(runtimeName, output); id != "" {
						persistSession(id)
					}
					fmt.Fprint(os.Stdout, string(output))
				}

			case "codex":
				// Codex JSONL: parse line by line for thread_id and agent messages
				var texts []string
				for _, line := range bytes.Split(bytes.TrimSpace(output), []byte("\n")) {
					if len(line) == 0 || line[0] != '{' {
						continue
					}
					var event struct {
						Type     string `json:"type"`
						ThreadID string `json:"thread_id"`
						Item     struct {
							Type string `json:"type"`
							Text string `json:"text"`
						} `json:"item"`
					}
					if jsonErr := json.Unmarshal(line, &event); jsonErr != nil {
						continue
					}
					if event.Type == "thread.started" && event.ThreadID != "" {
						persistSession(event.ThreadID)
					}
					if event.Type == "item.completed" && event.Item.Text != "" {
						texts = append(texts, event.Item.Text)
					}
				}
				if len(texts) > 0 {
					fmt.Fprintln(os.Stdout, strings.Join(texts, "\n"))
				} else {
					fmt.Fprint(os.Stdout, string(output))
				}

			default:
				fmt.Fprint(os.Stdout, string(output))
			}
		} else {
			// Interactive mode
			dockerArgs := append(buildDockerArgs(true), wrapWithCredentials(runtimeCmd)...)
			execCmd := exec.Command("docker", dockerArgs...)
			execCmd.Stdin = os.Stdin
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr

			if err := execCmd.Run(); err != nil {
				return fmt.Errorf("interactive session: %w", err)
			}
		}

		return nil
	},
}

func isContainerRunning(containerID string) bool {
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", containerID).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// findAgentContainer returns containerID, worldID, world record, and architect.
func findAgentContainer(agentName, worldID string) (string, string, *universe.World, *universe.Architect, error) {
	ctx := context.Background()

	arc, err := universe.NewArchitectFromEnv()
	if err != nil {
		if strings.Contains(err.Error(), "cannot connect to Docker") {
			return "", "", nil, nil, fmt.Errorf("Docker is not running")
		}
		return "", "", nil, nil, err
	}

	worlds, err := arc.List(ctx)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("cannot list worlds: %w", err)
	}

	cID, foundWorldID, routeErr := routeAgentToWorld(worlds, agentName, worldID, isContainerRunning)
	if routeErr != nil {
		return "", "", nil, nil, routeErr
	}

	for i := range worlds {
		if worlds[i].ID == foundWorldID {
			return cID, foundWorldID, &worlds[i], arc, nil
		}
	}

	return cID, foundWorldID, nil, arc, nil
}

func worldHasAgent(u universe.World, agentName string) bool {
	if u.Agent == agentName {
		return true
	}
	for _, a := range u.Agents {
		if a.Name == agentName {
			return true
		}
	}
	return false
}

func routeAgentToWorld(
	worlds []universe.World,
	agentName, worldID string,
	isRunning func(containerID string) bool,
) (string, string, error) {
	if worldID != "" {
		for _, u := range worlds {
			if u.ID != worldID {
				continue
			}
			if u.ContainerID == "" {
				return "", "", fmt.Errorf("world %s has no container", u.ID)
			}
			if !isRunning(u.ContainerID) {
				return "", "", fmt.Errorf("world %s is not running", u.ID)
			}
			if !worldHasAgent(u, agentName) {
				return "", "", fmt.Errorf("world %s does not contain agent %q", u.ID, agentName)
			}
			return u.ContainerID, u.ID, nil
		}
		return "", "", fmt.Errorf("world %q not found", worldID)
	}

	for _, u := range worlds {
		if u.ContainerID == "" || !isRunning(u.ContainerID) {
			continue
		}
		if worldHasAgent(u, agentName) {
			return u.ContainerID, u.ID, nil
		}
	}

	return "", "", fmt.Errorf("agent %q is not in any active world\n\n  Spawn one with: spwn up --agent %s -w <workspace>", agentName, agentName)
}


// extractSessionID looks at one line (or one document) of runtime output
// and returns the runtime's session/thread identifier if present.
//
// Both Claude (--output-format stream-json) and Codex (--json) emit their
// session id on the very first event of the conversation:
//
//   - Claude: {"type":"system","subtype":"init","session_id":"..."}
//     (and the "session_id" field is repeated on subsequent events too)
//   - Codex:  {"type":"thread.started","thread_id":"..."}
//
// We accept both shapes and return the first non-empty value found. The
// function is tolerant — non-JSON lines, partial lines, and unknown event
// types all return "".
func extractSessionID(runtimeName string, line []byte) string {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return ""
	}
	var event struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
		ThreadID  string `json:"thread_id"`
	}
	if err := json.Unmarshal(trimmed, &event); err != nil {
		return ""
	}
	if runtimeName == "codex" {
		return event.ThreadID
	}
	return event.SessionID
}

func formatExecError(err error, output []byte) error {
	out := string(output)

	switch {
	case strings.Contains(out, "authentication_error"):
		return fmt.Errorf("authentication failed\n\n  Run: spwn auth check\n  Run: spwn auth login")
	case strings.Contains(out, "OAuth token has expired"):
		return fmt.Errorf("OAuth token has expired — refresh credentials\n\n  Run: spwn auth login")
	case strings.Contains(out, "Invalid API key") || strings.Contains(out, "invalid x-api-key"):
		return fmt.Errorf("invalid API key\n\n  Run: spwn auth login")
	case strings.Contains(out, "Could not resolve host") || strings.Contains(out, "connection refused"):
		return fmt.Errorf("network error — cannot reach API\n\n  Output: %s", strings.TrimSpace(out))
	case strings.Contains(out, "rate_limit") || strings.Contains(out, "overloaded"):
		return fmt.Errorf("rate limited — try again in a moment")
	}

	if out != "" {
		if len(out) > 500 {
			out = out[:500] + "..."
		}
		return fmt.Errorf("agent exec failed: %w\n\n  Output:\n  %s\n\n  Hint: Run 'spwn auth check' to verify credentials",
			err, strings.TrimSpace(out))
	}

	return fmt.Errorf("agent exec failed: %w\n\n  Hint: Run 'spwn auth check' to verify credentials", err)
}
