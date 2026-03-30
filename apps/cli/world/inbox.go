package world

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/messenger"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(inboxCmd)
}

var inboxCmd = &cobra.Command{
	Use:   "inbox <world-id> [agent-name]",
	Short: "Show messages in a world's inbox",
	Long: `Show messages from the inbox inside a running world.

If an agent name is provided, shows only that agent's messages.
If not, shows all messages across all inboxes.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		worldID := args[0]
		agentName := ""
		if len(args) > 1 {
			agentName = args[1]
		}
		s := newStepper(cmd)

		j, _ := cmd.Flags().GetBool("json")

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return err
		}

		u, err := arc.Inspect(ctx, worldID)
		if err != nil {
			return fmt.Errorf("error: world %s not found.\nRun 'spwn world list' to see available worlds.", worldID)
		}

		var msgs []messenger.Message

		if agentName != "" {
			// Read messages for a specific agent
			msgs, err = readContainerInbox(u.ContainerID, agentName)
		} else {
			// Read all messages from all inboxes
			msgs, err = readAllContainerInboxes(u.ContainerID)
		}
		if err != nil {
			return err
		}

		if j {
			data, _ := json.MarshalIndent(msgs, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(msgs) == 0 {
			s.Blank()
			s.Success("No messages.")
			s.Blank()
			return nil
		}

		t := ui.NewTable(ui.ModeNormal, "FROM", "TO", "TYPE", "STATUS", "TIME", "CONTENT")
		for _, m := range msgs {
			content := m.Content
			if len(content) > 50 {
				content = content[:47] + "..."
			}
			t.AddRow(m.From, m.To, m.Type, m.Status, messageTimeAgo(m.Timestamp), content)
		}
		t.Render()

		return nil
	},
}

// readContainerInbox reads messages for a specific agent from inside a container.
func readContainerInbox(containerID, agentName string) ([]messenger.Message, error) {
	// List JSON files in the agent's inbox
	listCmd := fmt.Sprintf("find /world/inbox/%s -name '*.json' 2>/dev/null || true", agentName)
	out, err := exec.Command("docker", "exec", containerID, "sh", "-c", listCmd).Output()
	if err != nil {
		return nil, nil // Inbox doesn't exist yet
	}

	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	return readContainerMessages(containerID, files)
}

// readAllContainerInboxes reads all messages from all agent inboxes inside a container.
func readAllContainerInboxes(containerID string) ([]messenger.Message, error) {
	// List all JSON files under /world/inbox/
	listCmd := "find /world/inbox -name '*.json' 2>/dev/null || true"
	out, err := exec.Command("docker", "exec", containerID, "sh", "-c", listCmd).Output()
	if err != nil {
		return nil, nil
	}

	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	return readContainerMessages(containerID, files)
}

// readContainerMessages reads and parses message JSON files from inside a container.
func readContainerMessages(containerID string, files []string) ([]messenger.Message, error) {
	var msgs []messenger.Message
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}

		out, err := exec.Command("docker", "exec", containerID, "cat", f).Output()
		if err != nil {
			continue
		}

		var msg messenger.Message
		if err := json.Unmarshal(out, &msg); err != nil {
			continue
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

// messageTimeAgo formats a message timestamp as a relative time string.
func messageTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "\u2014"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
