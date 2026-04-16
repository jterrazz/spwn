package world

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/architect"
	"github.com/spf13/cobra"
)

func init() {
	knowledgeCmd.AddCommand(knowledgeLsCmd)
	knowledgeCmd.AddCommand(knowledgeShowCmd)
	Cmd.AddCommand(knowledgeCmd)
}

var knowledgeCmd = &cobra.Command{
	Use:   "knowledge",
	Short: "Read a world's shared knowledge",
	Long: `Each world carries its own knowledge base at /world/knowledge/ inside the
container - shared notes, decisions, and context for the agents working there.
Knowledge is per-world; destroy the world and its knowledge goes with it.`,
}

var knowledgeLsCmd = &cobra.Command{
	Use:   "ls <world-id>",
	Short: "List all knowledge files in a world",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID, err := resolveWorldContainer(args[0])
		if err != nil {
			return err
		}

		out, err := dockerExec(containerID,
			"find", "/world/knowledge/", "-type", "f", "-not", "-name", ".*", "-printf", "%P\n")
		if err != nil || strings.TrimSpace(out) == "" {
			s := ui.New()
			s.Blank()
			s.Info("Knowledge:", "empty")
			s.Blank()
			return nil
		}

		s := ui.New()
		s.Blank()
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", ui.Strong("Knowledge files:"))
		s.Blank()
		for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
			if f == "" {
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", ui.ColorizeHelpName(f))
		}
		s.Blank()
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", ui.Faint("Use \"spwn world knowledge show <world-id> <path>\" to read a file."))
		s.Blank()
		return nil
	},
}

var knowledgeShowCmd = &cobra.Command{
	Use:   "show <world-id> <path>",
	Short: "Show the contents of a knowledge file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldID, relPath := args[0], args[1]
		if strings.Contains(relPath, "..") {
			return fmt.Errorf("invalid path")
		}
		containerID, err := resolveWorldContainer(worldID)
		if err != nil {
			return err
		}

		out, err := dockerExec(containerID, "cat", "/world/knowledge/"+relPath)
		if err != nil {
			return fmt.Errorf("file not found: %s", relPath)
		}

		s := ui.New()
		s.Blank()
		fmt.Fprint(cmd.OutOrStdout(), out)
		s.Blank()
		return nil
	},
}

func resolveWorldContainer(worldID string) (string, error) {
	arc, err := architect.NewFromEnv()
	if err != nil {
		return "", dockerHint(err)
	}
	u, err := arc.Inspect(context.Background(), worldID)
	if err != nil {
		return "", fmt.Errorf("world %s not found\n\n  List worlds with: spwn world list", worldID)
	}
	if u.ContainerID == "" {
		return "", fmt.Errorf("world %s has no running container", worldID)
	}
	return u.ContainerID, nil
}

func dockerExec(containerID string, args ...string) (string, error) {
	cmdArgs := append([]string{"exec", containerID}, args...)
	out, err := exec.Command("docker", cmdArgs...).Output()
	return string(out), err
}
