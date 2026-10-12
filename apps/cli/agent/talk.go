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
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/authentication"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/runtimes"
	"spwn.sh/packages/world"
)

var (
	talkOutputFormat   string
	talkWorldID        string
	talkSuppressHeader bool
)

// SuppressTalkHeader is set by runInteractiveSession just before it
// Delegates to talkCmd.RunE. The `spwn agent <name>` shortcut already
// Printed a Hero + Spawn phase with world/agent IDs, so repeating
// "Agent: X / World: Y" inside talk would be redundant. One-shot
// Resets itself at the end of the talk call.
func SuppressTalkHeader() {
	talkSuppressHeader = true
}

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
		s := ui.New()

		if err := agent.ValidateMind(name); err != nil {
			return fmt.Errorf("agent %q not found\n\n  Create one with: spwn agent create %s", name, name)
		}

		containerID, worldID, w, arc, err := findAgentContainer(name, talkWorldID)
		if err != nil {
			return err
		}

		suppressed := talkOutputFormat == "stream-json" || talkSuppressHeader
		talkSuppressHeader = false
		if !suppressed {
			s.Blank()
			s.Info("Agent:", name)
			s.Info("World:", worldID)
			s.Blank()
		}

		sessionID := ""
		if arc != nil {
			sessionID = arc.GetSessionID(worldID, name)
		}

		runtimeName := "claude-code"
		if w != nil && w.Runtime != "" {
			runtimeName = w.Runtime
		}
		rtSpawner, rtErr := runtimes.GetSpawner(runtimeName)
		if rtErr != nil {
			return fmt.Errorf("cannot resolve runtime %q for world %s: %w", runtimeName, worldID, rtErr)
		}
		runtimeCmd := rtSpawner.BuildCommand(runtimes.SpawnConfig{
			AgentName: name,
			WorldID:   worldID,
			Prompt:    message,
			SessionID: sessionID,
		})

		if message != "" {
			runtimeCmd = rtSpawner.OneShotFlags(runtimeCmd, talkOutputFormat)
		}

		_ = authentication.SyncCredentials()
		if err := rtSpawner.SyncHostCredentials(platform.CredentialsDir()); err != nil {
			return fmt.Errorf("sync credentials: %w", err)
		}

		buildDockerArgs := func(interactive bool) []string {
			agentHome := "/agents/" + name
			args := []string{"exec"}
			if interactive {
				args = append(args, "-it")
			}
			args = append(args,
				"-w", agentHome,
				"-e", "HOME="+agentHome,
				"-e", "SPWN_AGENT_NAME="+name,
				"-e", "SPWN_WORLD_ID="+worldID,
			)
			args = append(args, containerID)
			return args
		}

		wrapWithCredentials := func(cmd []string) []string {
			escaped := make([]string, len(cmd))
			for i, arg := range cmd {
				escaped[i] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
			}
			setup := "source /credentials/.env 2>/dev/null; " + rtSpawner.PrelaunchShell()
			shellCmd := setup + "; exec " + strings.Join(escaped, " ")
			return []string{"bash", "-c", shellCmd}
		}

		if message != "" {
			dockerArgs := append(buildDockerArgs(false), wrapWithCredentials(runtimeCmd)...)
			execCmd := exec.Command("docker", dockerArgs...)

			var captured string
			persistSession := func(id string) {
				if id == "" || id == captured || arc == nil {
					return
				}
				captured = id
				_ = arc.SetSessionID(worldID, name, id)
			}

			if talkOutputFormat == "stream-json" {
				stdoutPipe, pipeErr := execCmd.StdoutPipe()
				if pipeErr != nil {
					return fmt.Errorf("stdout pipe: %w", pipeErr)
				}
				execCmd.Stderr = nil
				if err := execCmd.Start(); err != nil {
					return formatExecError(err, nil)
				}
				scanner := bufio.NewScanner(stdoutPipe)
				scanner.Buffer(make([]byte, 1024*1024), 4*1024*1024)
				for scanner.Scan() {
					line := scanner.Bytes()
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

			text, sessionID, parseErr := rtSpawner.ParseOneShotResult(output)
			if parseErr == nil {
				persistSession(sessionID)
				fmt.Fprintln(os.Stdout, text)
			} else {
				if id := extractSessionID(runtimeName, output); id != "" {
					persistSession(id)
				}
				fmt.Fprint(os.Stdout, string(output))
			}
		} else {
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

func findAgentContainer(agentName, worldID string) (string, string, *world.World, *architect.Architect, error) {
	ctx := context.Background()

	arc, err := architect.NewFromEnv()
	if err != nil {
		if strings.Contains(err.Error(), "cannot connect to Docker") {
			return "", "", nil, nil, fmt.Errorf("cannot reach Docker — is Docker Desktop or OrbStack running?")
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
