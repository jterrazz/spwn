package agent

import (
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

		runtimeCmd, rtErr := universe.BuildRuntimeCommand(runtimeName, universe.RuntimeSpawnConfig{
			MindPath:  world.MindPath,
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
			args := []string{"exec"}
			if interactive {
				args = append(args, "-it")
			}
			args = append(args, "-w", "/workspace")
			// No -e flags needed — credentials are in /credentials/.env (bind mount)
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

			if talkOutputFormat == "stream-json" {
				execCmd.Stdout = os.Stdout
				// Suppress stderr in streaming mode (codex emits noisy MCP errors)
				if err := execCmd.Run(); err != nil {
					return formatExecError(err, nil)
				}
			} else {
				output, err := execCmd.CombinedOutput()
				if err != nil {
					return formatExecError(err, output)
				}

				// Parse response based on runtime to extract session ID and text
				switch runtimeName {
				case "claude-code":
					var resp struct {
						Result    string `json:"result"`
						SessionID string `json:"session_id"`
					}
					if jsonErr := json.Unmarshal(output, &resp); jsonErr == nil {
						if resp.SessionID != "" && arc != nil {
							_ = arc.SetSessionID(worldID, name, resp.SessionID)
						}
						fmt.Fprintln(os.Stdout, resp.Result)
					} else {
						fmt.Fprint(os.Stdout, string(output))
					}

				case "codex":
					// Codex JSONL: parse line by line for thread_id and agent messages
					var threadID string
					var texts []string
					for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
						if line == "" || line[0] != '{' {
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
						if jsonErr := json.Unmarshal([]byte(line), &event); jsonErr != nil {
							continue
						}
						if event.Type == "thread.started" && event.ThreadID != "" {
							threadID = event.ThreadID
						}
						if event.Type == "item.completed" && event.Item.Text != "" {
							texts = append(texts, event.Item.Text)
						}
					}
					if threadID != "" && arc != nil {
						_ = arc.SetSessionID(worldID, name, threadID)
					}
					if len(texts) > 0 {
						fmt.Fprintln(os.Stdout, strings.Join(texts, "\n"))
					} else {
						fmt.Fprint(os.Stdout, string(output))
					}

				default:
					fmt.Fprint(os.Stdout, string(output))
				}
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
