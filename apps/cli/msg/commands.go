package msg

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

var (
	msgFrom string
	msgType string
)

func init() {
	sendCmd.Flags().StringVar(&msgFrom, "from", "", "Sender agent name (required)")
	sendCmd.Flags().StringVar(&msgType, "type", "task", "Message type: task, reply, question, announcement")
	sendCmd.MarkFlagRequired("from")

	Cmd.AddCommand(sendCmd)
	Cmd.AddCommand(inboxCmd)
	Cmd.AddCommand(watchCmd)
}

// newStepper creates a Stepper that respects the --json flag.
func newStepper(cmd *cobra.Command) *ui.Stepper {
	j, _ := cmd.Flags().GetBool("json")
	return ui.New(j)
}

// --- send ---

var sendCmd = &cobra.Command{
	Use:   "send <agent-name> [message]",
	Short: "Send a message to an agent's inbox",
	Long:  `Send an async message to a running agent. The agent must be in an active world.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		toAgent := args[0]
		content := ""
		if len(args) > 1 {
			content = args[1]
		}
		s := newStepper(cmd)

		containerID, _, err := findAgentContainer(toAgent)
		if err != nil {
			return fmt.Errorf("agent %q is not in any active world", toAgent)
		}

		now := time.Now()
		from := msgFrom
		id := fmt.Sprintf("msg-%s-%s-%03d", from, now.Format("20060102-150405"), now.Nanosecond()/1000000)

		msg := map[string]interface{}{
			"id":        id,
			"from":      from,
			"to":        toAgent,
			"timestamp": now,
			"type":      msgType,
			"content":   content,
			"status":    "unread",
		}

		data, err := json.MarshalIndent(msg, "", "  ")
		if err != nil {
			return err
		}

		// Create inbox dir + write message
		mkdirArgs := []string{"exec", containerID, "mkdir", "-p", "/world/inbox/" + toAgent}
		if _, err := exec.Command("docker", mkdirArgs...).CombinedOutput(); err != nil {
			return fmt.Errorf("cannot create inbox directory: %w", err)
		}

		path := fmt.Sprintf("/world/inbox/%s/%s.json", toAgent, id)
		writeCmd := fmt.Sprintf("cat > %s << 'MSGEOF'\n%s\nMSGEOF", path, string(data))
		writeArgs := []string{"exec", containerID, "sh", "-c", writeCmd}
		if _, err := exec.Command("docker", writeArgs...).CombinedOutput(); err != nil {
			return fmt.Errorf("cannot write message: %w", err)
		}

		s.Blank()
		s.Done("Sent message", fmt.Sprintf("%s → %s", from, toAgent))
		s.Blank()
		return nil
	},
}

// --- inbox ---

var inboxCmd = &cobra.Command{
	Use:   "inbox <agent-name>",
	Short: "Show messages in an agent's inbox",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		s := newStepper(cmd)
		j, _ := cmd.Flags().GetBool("json")

		containerID, _, err := findAgentContainer(agentName)
		if err != nil {
			return fmt.Errorf("agent %q is not in any active world", agentName)
		}

		msgs, err := readContainerInbox(containerID, agentName)
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

		t := ui.NewTable("FROM", "TYPE", "STATUS", "TIME", "CONTENT")
		for _, m := range msgs {
			content := m.Content
			if len(content) > 50 {
				content = content[:47] + "..."
			}
			t.AddRow(m.From, m.Type, m.Status, messageTimeAgo(m.Timestamp), content)
		}
		t.Render()
		return nil
	},
}

// --- watch ---

var watchCmd = &cobra.Command{
	Use:   "watch <agent-name>",
	Short: "Watch for new messages to an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		agentName := args[0]
		s := newStepper(cmd)

		containerID, worldID, err := findAgentContainer(agentName)
		if err != nil {
			return fmt.Errorf("agent %q is not in any active world", agentName)
		}

		s.Blank()
		s.Info("Watching:", fmt.Sprintf("%s in %s", agentName, worldID))
		s.Info("Polling:", "every 5 seconds")
		s.Blank()

		exec.Command("docker", "exec", containerID, "touch", "/tmp/.last-check").Run()

		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			findCmd := "find /world/inbox -name '*.json' -newer /tmp/.last-check 2>/dev/null || true"
			out, err := exec.Command("docker", "exec", containerID, "sh", "-c", findCmd).Output()
			if err == nil {
				files := strings.Split(strings.TrimSpace(string(out)), "\n")
				for _, f := range files {
					f = strings.TrimSpace(f)
					if f == "" {
						continue
					}

					msgOut, err := exec.Command("docker", "exec", containerID, "cat", f).Output()
					if err != nil {
						continue
					}

					var msg messenger.Message
					if err := json.Unmarshal(msgOut, &msg); err != nil {
						continue
					}

					if msg.Status != "unread" || msg.To != agentName {
						continue
					}

					fmt.Fprintf(cmd.ErrOrStderr(), "  [%s] %s → %s: %s\n",
						time.Now().Format("15:04:05"),
						msg.From, msg.To, truncate(msg.Content, 60))

					msg.Status = "delivered"
					updated, err := json.MarshalIndent(msg, "", "  ")
					if err == nil {
						writeCmd := fmt.Sprintf("cat > %s << 'MSGEOF'\n%s\nMSGEOF", f, string(updated))
						exec.Command("docker", "exec", containerID, "sh", "-c", writeCmd).Run()
					}
				}
			}

			exec.Command("docker", "exec", containerID, "touch", "/tmp/.last-check").Run()
			time.Sleep(5 * time.Second)
		}
	},
}

// --- helpers ---

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

	for _, u := range worlds {
		if u.ContainerID == "" {
			continue
		}

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

	return "", "", fmt.Errorf("agent %q not found in any running world", agentName)
}

func isContainerRunning(containerID string) bool {
	out, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", containerID).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func readContainerInbox(containerID, agentName string) ([]messenger.Message, error) {
	listCmd := fmt.Sprintf("find /world/inbox/%s -name '*.json' 2>/dev/null || true", agentName)
	out, err := exec.Command("docker", "exec", containerID, "sh", "-c", listCmd).Output()
	if err != nil {
		return nil, nil
	}

	files := strings.Split(strings.TrimSpace(string(out)), "\n")
	var msgs []messenger.Message
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		msgOut, err := exec.Command("docker", "exec", containerID, "cat", f).Output()
		if err != nil {
			continue
		}
		var msg messenger.Message
		if err := json.Unmarshal(msgOut, &msg); err != nil {
			continue
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
