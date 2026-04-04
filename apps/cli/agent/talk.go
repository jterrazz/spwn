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

var talkOutputFormat string

func init() {
	talkCmd.Flags().StringVar(&talkOutputFormat, "output-format", "", "Output format: text (default) or stream-json")
	Cmd.AddCommand(talkCmd)
}

var talkCmd = &cobra.Command{
	Use:   "talk <agent-name> [message]",
	Short: "Talk to a running agent — interactive or one-shot",
	Long: `Open a conversation with a named agent running inside a world.

If a message is provided, runs a one-shot query and prints the response.
If no message is provided, opens an interactive Claude session inside the container.`,
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
		containerID, worldID, err := findAgentContainer(name)
		if err != nil {
			return err
		}

		s.Blank()
		s.Info("Agent:", name)
		s.Info("World:", worldID)
		s.Blank()

		// Base claude command — continue latest session in this workspace
		claudeArgs := []string{
			"claude",
			"--dangerously-skip-permissions",
			"--continue",
		}

		// Build docker exec args with auth env vars
		buildDockerArgs := func(interactive bool) []string {
			args := []string{"exec"}
			if interactive {
				args = append(args, "-it")
			}
			args = append(args, "-w", "/workspace")
			// Pass all available auth credentials
			args = append(args, authEnvArgs()...)
			args = append(args, containerID)
			return args
		}

		if message != "" {
			// One-shot mode
			claudeArgs = append(claudeArgs, "-p", message)

			if talkOutputFormat == "stream-json" {
				claudeArgs = append(claudeArgs, "--output-format", "stream-json", "--verbose")
			} else {
				claudeArgs = append(claudeArgs, "--print")
			}

			dockerArgs := append(buildDockerArgs(false), claudeArgs...)
			execCmd := exec.Command("docker", dockerArgs...)

			if talkOutputFormat == "stream-json" {
				// Stream stdout directly for real-time output
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
			// Interactive mode: attach stdin/stdout/stderr
			dockerArgs := append(buildDockerArgs(true), claudeArgs...)
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

// findAgentContainer looks up state.json to find a running world
// that contains the given agent. Verifies the container is actually alive.
func findAgentContainer(agentName string) (string, string, error) {
	ctx := context.Background()

	arc, err := universe.NewArchitectFromEnv()
	if err != nil {
		if strings.Contains(err.Error(), "cannot connect to Docker") {
			return "", "", fmt.Errorf("Docker is not running")
		}
		return "", "", err
	}

	worlds, err := arc.List(ctx)
	if err != nil {
		return "", "", fmt.Errorf("cannot list worlds: %w", err)
	}

	// Check the primary agent field, then the agents array
	for _, u := range worlds {
		if u.ContainerID == "" {
			continue
		}

		// Verify the container is actually running (not just in state.json)
		if !isContainerRunning(u.ContainerID) {
			continue
		}

		// Check primary agent
		if u.Agent == agentName {
			return u.ContainerID, u.ID, nil
		}

		// Check multi-agent records
		for _, a := range u.Agents {
			if a.Name == agentName {
				if u.ContainerID == "" {
					return "", "", fmt.Errorf("world %s has no container", u.ID)
				}
				return u.ContainerID, u.ID, nil
			}
		}
	}

	return "", "", fmt.Errorf("agent %q is not in any active world\n\n  Spawn one with: spwn up --agent %s -w <workspace>", agentName, agentName)
}

// authEnvArgs returns Docker -e flags for all available auth credentials.
func authEnvArgs() []string {
	return auth.DockerEnvArgs()
}

// formatExecError parses docker exec output for common auth errors
// and returns an actionable error message.
func formatExecError(err error, output []byte) error {
	out := string(output)

	// Check for common auth-related errors
	switch {
	case strings.Contains(out, "authentication_error"):
		return fmt.Errorf("authentication failed — your API key or OAuth token was rejected\n\n  %s\n  %s",
			"Run: spwn auth check    (validate credentials)",
			"Run: spwn auth login    (refresh credentials)")
	case strings.Contains(out, "OAuth token has expired"):
		return fmt.Errorf("OAuth token has expired\n\n  %s\n  %s",
			"Run: spwn auth login    (refresh from Keychain)",
			"Or re-authenticate in Claude Code CLI first")
	case strings.Contains(out, "Invalid API key") || strings.Contains(out, "invalid x-api-key"):
		return fmt.Errorf("invalid API key\n\n  %s\n  %s",
			"Run: spwn auth login    (set up fresh credentials)",
			"Run: spwn auth token <key>  (set key directly)")
	case strings.Contains(out, "Could not resolve host") || strings.Contains(out, "connection refused"):
		return fmt.Errorf("network error — cannot reach API\n\n  Output: %s", strings.TrimSpace(out))
	case strings.Contains(out, "rate_limit") || strings.Contains(out, "overloaded"):
		return fmt.Errorf("rate limited or API overloaded — try again in a moment\n\n  Output: %s", strings.TrimSpace(out))
	}

	// Generic fallback: include the output so users can see what happened
	if out != "" {
		// Truncate very long output
		if len(out) > 500 {
			out = out[:500] + "..."
		}
		return fmt.Errorf("agent exec failed: %w\n\n  Output:\n  %s\n\n  Hint: Run 'spwn auth check' to verify credentials",
			err, strings.TrimSpace(out))
	}

	return fmt.Errorf("agent exec failed: %w\n\n  Hint: Run 'spwn auth check' to verify credentials", err)
}
