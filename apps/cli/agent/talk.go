package agent

import (
	"context"
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

		// Validate the agent exists
		if err := agentDomain.ValidateMind(name); err != nil {
			return fmt.Errorf("agent %q not found\n\n  Create one with: spwn agent new %s", name, name)
		}

		// Find which world this agent is in
		containerID, worldID, world, err := findAgentContainer(name, talkWorldID)
		if err != nil {
			return err
		}

		s.Blank()
		s.Info("Agent:", name)
		s.Info("World:", worldID)
		s.Blank()

		// Get the runtime from the world state and build command via adapter
		runtimeName := world.Runtime
		if runtimeName == "" {
			runtimeName = "claude-code"
		}
		runtimeCmd, rtErr := universe.BuildRuntimeCommand(runtimeName, universe.RuntimeSpawnConfig{
			MindPath:  world.MindPath,
			AgentName: name,
			WorldID:   worldID,
			Prompt:    message,
		})
		if rtErr != nil {
			return fmt.Errorf("unknown runtime %q for world %s", runtimeName, worldID)
		}

		// For claude-code one-shot: add --print flag
		if runtimeName == "claude-code" && message != "" {
			if talkOutputFormat == "stream-json" {
				runtimeCmd = append(runtimeCmd, "--output-format", "stream-json", "--verbose")
			} else {
				runtimeCmd = append(runtimeCmd, "--print")
			}
		}

		// Build docker exec args with auth env vars
		buildDockerArgs := func(interactive bool) []string {
			args := []string{"exec"}
			if interactive {
				args = append(args, "-it")
			}
			args = append(args, "-w", "/workspace")
			args = append(args, authEnvArgs()...)
			args = append(args, containerID)
			return args
		}

		if message != "" {
			// One-shot mode
			dockerArgs := append(buildDockerArgs(false), runtimeCmd...)
			execCmd := exec.Command("docker", dockerArgs...)

			if talkOutputFormat == "stream-json" {
				execCmd.Stdout = os.Stdout
				execCmd.Stderr = os.Stderr
				if err := execCmd.Run(); err != nil {
					return formatExecError(err, nil)
				}
			} else {
				output, err := execCmd.CombinedOutput()
				if err != nil {
					return formatExecError(err, output)
				}
				fmt.Fprint(os.Stdout, string(output))
			}
		} else {
			// Interactive mode — same command but with TTY
			dockerArgs := append(buildDockerArgs(true), runtimeCmd...)
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

// isContainerRunning checks if a Docker container is actually alive.
func isContainerRunning(containerID string) bool {
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", containerID).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// findAgentContainer looks up state.json to find a running world that contains
// the given agent. Returns the container ID, world ID, and full world record.
func findAgentContainer(agentName, worldID string) (string, string, *universe.World, error) {
	ctx := context.Background()

	arc, err := universe.NewArchitectFromEnv()
	if err != nil {
		if strings.Contains(err.Error(), "cannot connect to Docker") {
			return "", "", nil, fmt.Errorf("Docker is not running")
		}
		return "", "", nil, err
	}

	worlds, err := arc.List(ctx)
	if err != nil {
		return "", "", nil, fmt.Errorf("cannot list worlds: %w", err)
	}

	cID, foundWorldID, routeErr := routeAgentToWorld(worlds, agentName, worldID, isContainerRunning)
	if routeErr != nil {
		return "", "", nil, routeErr
	}

	// Return the full world record for runtime/mind info
	for i := range worlds {
		if worlds[i].ID == foundWorldID {
			return cID, foundWorldID, &worlds[i], nil
		}
	}

	return cID, foundWorldID, nil, nil
}

// worldHasAgent returns true when the given agent name matches the world's
// primary agent or any entry in its agents slice.
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

// routeAgentToWorld contains the pure routing logic.
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

// authEnvArgs returns Docker -e flags for all available auth credentials.
func authEnvArgs() []string {
	return auth.DockerEnvArgs()
}

// formatExecError parses docker exec output for common auth errors.
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
