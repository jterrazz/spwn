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
	Long: `Open a conversation with a named agent running inside a universe.

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
			return fmt.Errorf("error: agent %q not found.\nRun 'spwn agent list' to see available agents.", name)
		}

		// Find which universe this agent is in
		containerID, universeID, err := findAgentContainer(name)
		if err != nil {
			return err
		}

		// Read the OAuth token
		token := readAuthToken()

		s.Blank()
		s.Info("Agent:", name)
		s.Info("Universe:", universeID)
		s.Blank()

		// Base claude command: continue latest session in /workspace
		claudeArgs := []string{
			"claude",
			"--dangerously-skip-permissions",
			"--continue",
		}

		if message != "" {
			// One-shot mode: run claude with --print
			claudeArgs = append(claudeArgs, "-p", message, "--print")

			dockerArgs := []string{"exec", "-w", "/workspace"}
			if token != "" {
				dockerArgs = append(dockerArgs, "-e", "CLAUDE_CODE_OAUTH_TOKEN="+token)
			}
			dockerArgs = append(dockerArgs, containerID)
			dockerArgs = append(dockerArgs, claudeArgs...)

			execCmd := exec.Command("docker", dockerArgs...)
			output, err := execCmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("error: claude exec failed.\n%s\n%w", string(output), err)
			}

			fmt.Fprint(os.Stdout, string(output))
		} else {
			// Interactive mode: attach stdin/stdout/stderr
			dockerArgs := []string{"exec", "-it", "-w", "/workspace"}
			if token != "" {
				dockerArgs = append(dockerArgs, "-e", "CLAUDE_CODE_OAUTH_TOKEN="+token)
			}
			dockerArgs = append(dockerArgs, containerID)
			dockerArgs = append(dockerArgs, claudeArgs...)

			execCmd := exec.Command("docker", dockerArgs...)
			execCmd.Stdin = os.Stdin
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr

			if err := execCmd.Run(); err != nil {
				return fmt.Errorf("error: interactive session failed.\n%w", err)
			}
		}

		return nil
	},
}

// findAgentContainer looks up state.json to find a running universe
// that contains the given agent. Returns (containerID, universeID, error).
func findAgentContainer(agentName string) (string, string, error) {
	ctx := context.Background()

	arc, err := universe.NewArchitectFromEnv()
	if err != nil {
		return "", "", fmt.Errorf("error: cannot connect to backend.\n%w", err)
	}

	universes, err := arc.List(ctx)
	if err != nil {
		return "", "", fmt.Errorf("error: cannot list universes.\n%w", err)
	}

	// Check the primary agent field, then the agents array
	for _, u := range universes {
		if u.Status != universe.StatusRunning && u.Status != universe.StatusIdle {
			continue
		}

		// Check primary agent
		if u.Agent == agentName {
			if u.ContainerID == "" {
				return "", "", fmt.Errorf("error: universe %s has no container ID", u.ID)
			}
			return u.ContainerID, u.ID, nil
		}

		// Check multi-agent records
		for _, a := range u.Agents {
			if a.Name == agentName && (a.Status == universe.StatusRunning || a.Status == universe.StatusIdle) {
				if u.ContainerID == "" {
					return "", "", fmt.Errorf("error: universe %s has no container ID", u.ID)
				}
				return u.ContainerID, u.ID, nil
			}
		}
	}

	return "", "", fmt.Errorf("error: agent %q is not in any active universe.\nSpawn it first with: spwn universe --agent %s", agentName, agentName)
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
