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

	"github.com/spf13/cobra"
	"spwn.sh/packages/auth"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/paths"
	"spwn.sh/packages/world"
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
	Short: "Talk to a running agent - interactive or one-shot",
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

		if err := agent.ValidateMind(name); err != nil {
			return fmt.Errorf("agent %q not found\n\n  Create one with: spwn agent create %s", name, name)
		}

		containerID, worldID, w, arc, err := findAgentContainer(name, talkWorldID)
		if err != nil {
			return err
		}

		// Suppress stepper output in stream-json mode (web UI reads raw JSON)
		if talkOutputFormat != "stream-json" {
			s.Blank()
			s.Info("Agent:", name)
			s.Info("World:", worldID)
			s.Blank()
		}

		// Look up existing session ID for this agent (enables conversation continuity)
		sessionID := ""
		if arc != nil {
			sessionID = arc.GetSessionID(worldID, name)
		}

		_ = w // world record kept for future use; runtime config no longer needs MindPath
		runtimeCmd, rtErr := world.BuildRuntimeCommand("claude-code", world.RuntimeSpawnConfig{
			AgentName: name,
			WorldID:   worldID,
			Prompt:    message,
			SessionID: sessionID,
		})
		if rtErr != nil {
			return fmt.Errorf("cannot build claude-code runtime command for world %s", worldID)
		}

		// Configure output format for session ID capture.
		// Claude: --print --output-format json → single JSON with session_id
		if message != "" {
			if talkOutputFormat == "stream-json" {
				runtimeCmd = append(runtimeCmd, "--output-format", "stream-json", "--verbose")
			} else {
				runtimeCmd = append(runtimeCmd, "--print", "--output-format", "json")
			}
		}

		// Sync credentials before talking. Two layers:
		//   1. packages/auth writes env vars + the codex auth.json into
		//      ~/.spwn/credentials/ (bind-mounted at /credentials/).
		//   2. The runtime provider syncs its own host-side files
		//      (e.g. Claude's ~/.claude/.credentials.json or the
		//      macOS Keychain) into the same directory.
		_ = auth.SyncCredentials()
		rt, rtErr2 := world.GetRuntime("claude-code")
		if rtErr2 != nil {
			return fmt.Errorf("lookup runtime: %w", rtErr2)
		}
		if err := rt.SyncHostCredentials(paths.CredentialsDir()); err != nil {
			return fmt.Errorf("sync credentials: %w", err)
		}

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

		// Wrap the runtime command with the provider's prelaunch
		// shell snippet. The snippet sources env vars from the bind
		// mount and symlinks/copies runtime-specific credential files
		// into the agent's HOME before exec'ing the real command.
		wrapWithCredentials := func(cmd []string) []string {
			escaped := make([]string, len(cmd))
			for i, arg := range cmd {
				escaped[i] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
			}
			setup := rt.PrelaunchShell()
			shellCmd := setup + "; exec " + strings.Join(escaped, " ")
			return []string{"bash", "-c", shellCmd}
		}

		if message != "" {
			dockerArgs := append(buildDockerArgs(false), wrapWithCredentials(runtimeCmd)...)
			execCmd := exec.Command("docker", dockerArgs...)

			// persistSession is called once we discover the runtime's session
			// identifier on stdout. It is intentionally idempotent - for streamed
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
				// Streaming mode (used by the web UI): tee stdout so we
				// can both forward each line to the caller AND scan it for the
				// runtime's session/thread id. Without this scan, every
				// web UI message starts a fresh conversation - the #1
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
					// Forward verbatim so the web UI's SSE relay still
					// sees the original event stream byte-for-byte.
					_, _ = os.Stdout.Write(line)
					_, _ = os.Stdout.Write([]byte{'\n'})
					if id := extractSessionID("claude-code", line); id != "" {
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

			// Parse the claude-code response to extract session ID and text.
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
				if id := extractSessionID("claude-code", output); id != "" {
					persistSession(id)
				}
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
func findAgentContainer(agentName, worldID string) (string, string, *world.World, *world.Architect, error) {
	ctx := context.Background()

	arc, err := world.NewArchitectFromEnv()
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

func worldHasAgent(u world.World, agentName string) bool {
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
	worlds []world.World,
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
// function is tolerant - non-JSON lines, partial lines, and unknown event
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
		return fmt.Errorf("OAuth token has expired - refresh credentials\n\n  Run: spwn auth login")
	case strings.Contains(out, "Invalid API key") || strings.Contains(out, "invalid x-api-key"):
		return fmt.Errorf("invalid API key\n\n  Run: spwn auth login")
	case strings.Contains(out, "Could not resolve host") || strings.Contains(out, "connection refused"):
		return fmt.Errorf("network error - cannot reach API\n\n  Output: %s", strings.TrimSpace(out))
	case strings.Contains(out, "rate_limit") || strings.Contains(out, "overloaded"):
		return fmt.Errorf("rate limited - try again in a moment")
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
