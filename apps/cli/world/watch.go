package world

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"spwn.sh/core/messenger"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(watchCmd)
}

var watchCmd = &cobra.Command{
	Use:   "watch <world-id>",
	Short: "Watch for new messages in a world",
	Long: `Run in the foreground, polling inbox directories every 5 seconds.
When new unread messages are found, prints a notification and wakes
the recipient agent via 'spwn agent talk'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		worldID := args[0]
		s := newStepper(cmd)

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		u, err := arc.Inspect(ctx, worldID)
		if err != nil {
			return fmt.Errorf("error: world %s not found.\nRun 'spwn world list' to see available worlds.", worldID)
		}

		s.Blank()
		s.Info("Watching:", worldID)
		s.Info("Polling:", "every 5 seconds")
		s.Blank()

		// Initialize last-check marker inside the container
		initCmd := []string{"exec", u.ContainerID, "touch", "/tmp/.last-check"}
		exec.Command("docker", initCmd...).Run()

		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			// Find new JSON files since last check
			findCmd := fmt.Sprintf("find /world/inbox -name '*.json' -newer /tmp/.last-check 2>/dev/null || true")
			out, err := exec.Command("docker", "exec", u.ContainerID, "sh", "-c", findCmd).Output()
			if err == nil {
				files := strings.Split(strings.TrimSpace(string(out)), "\n")
				for _, f := range files {
					f = strings.TrimSpace(f)
					if f == "" {
						continue
					}

					// Read the message
					msgOut, err := exec.Command("docker", "exec", u.ContainerID, "cat", f).Output()
					if err != nil {
						continue
					}

					var msg messenger.Message
					if err := json.Unmarshal(msgOut, &msg); err != nil {
						continue
					}

					if msg.Status != "unread" {
						continue
					}

					// Print notification
					fmt.Fprintf(cmd.ErrOrStderr(), "  [%s] %s → %s: %s\n",
						time.Now().Format("15:04:05"),
						msg.From, msg.To, truncate(msg.Content, 60))

					// Wake up the recipient agent
					talkArgs := []string{
						"agent", "talk", msg.To,
						fmt.Sprintf("You have a new message from %s. Check /world/inbox/%s/", msg.From, msg.To),
					}
					exec.Command("spwn", talkArgs...).Run()

					// Mark as delivered
					msg.Status = "delivered"
					updated, err := json.MarshalIndent(msg, "", "  ")
					if err == nil {
						writeCmd := fmt.Sprintf("cat > %s << 'MSGEOF'\n%s\nMSGEOF", f, string(updated))
						exec.Command("docker", "exec", u.ContainerID, "sh", "-c", writeCmd).Run()
					}
				}
			}

			// Update the last-check marker
			exec.Command("docker", "exec", u.ContainerID, "touch", "/tmp/.last-check").Run()

			time.Sleep(5 * time.Second)
		}
	},
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
