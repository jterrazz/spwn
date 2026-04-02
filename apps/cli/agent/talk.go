package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
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

		// Read the OAuth token
		token := readAuthToken()

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
			for _, kv := range authEnvVars(token) {
				args = append(args, "-e", kv)
			}
			args = append(args, containerID)
			return args
		}

		if message != "" {
			// One-shot mode: run claude with --print
			claudeArgs = append(claudeArgs, "-p", message, "--print")

			dockerArgs := append(buildDockerArgs(false), claudeArgs...)
			execCmd := exec.Command("docker", dockerArgs...)
			output, err := execCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("exec failed: %w", err)
			}

			fmt.Fprint(os.Stdout, string(output))
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

// readAuthToken reads the cached OAuth token from ~/.spwn/.auth-token.
func readAuthToken() string {
	cachePath := foundation.BaseDir() + "/.auth-token"
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// authEnvVars returns Docker -e flags for all available auth credentials.
func authEnvVars(cachedToken string) []string {
	var envs []string
	for _, key := range []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_AUTH_TOKEN"} {
		if val := os.Getenv(key); val != "" {
			envs = append(envs, key+"="+val)
		}
	}
	// Use cached token if CLAUDE_CODE_OAUTH_TOKEN not already set
	if cachedToken != "" && os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") == "" {
		envs = append(envs, "CLAUDE_CODE_OAUTH_TOKEN="+cachedToken)
	}
	return envs
}
