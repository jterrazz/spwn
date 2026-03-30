package world

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var (
	sendFrom string
	sendTo   string
	sendType string
)

func init() {
	sendCmd.Flags().StringVar(&sendFrom, "from", "", "Sender agent name (required)")
	sendCmd.Flags().StringVar(&sendTo, "to", "", "Recipient agent name (required)")
	sendCmd.Flags().StringVar(&sendType, "type", "task", "Message type: task, reply, question, announcement")
	sendCmd.MarkFlagRequired("from")
	sendCmd.MarkFlagRequired("to")

	Cmd.AddCommand(sendCmd)
}

var sendCmd = &cobra.Command{
	Use:   "send <world-id> [message]",
	Short: "Send a message between agents in a world",
	Long:  `Send a message to an agent's inbox inside a running world.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		worldID := args[0]
		s := newStepper(cmd)

		content := ""
		if len(args) > 1 {
			content = args[1]
		}

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return err
		}

		u, err := arc.Inspect(ctx, worldID)
		if err != nil {
			return fmt.Errorf("error: world %s not found.\nRun 'spwn world list' to see available worlds.", worldID)
		}

		// Build the message JSON
		now := time.Now()
		id := fmt.Sprintf("msg-%s-%s-%03d", sendFrom, now.Format("20060102-150405"), now.Nanosecond()/1000000)

		msg := map[string]interface{}{
			"id":        id,
			"from":      sendFrom,
			"to":        sendTo,
			"timestamp": now,
			"type":      sendType,
			"content":   content,
			"status":    "unread",
		}

		data, err := json.MarshalIndent(msg, "", "  ")
		if err != nil {
			return err
		}

		// Create the recipient inbox directory inside the container
		mkdirArgs := []string{"exec", u.ContainerID, "mkdir", "-p", "/world/inbox/" + sendTo}
		if out, err := exec.Command("docker", mkdirArgs...).CombinedOutput(); err != nil {
			return fmt.Errorf("error: cannot create inbox directory.\n%s\n%w", string(out), err)
		}

		// Write the message file via docker exec + sh -c
		path := fmt.Sprintf("/world/inbox/%s/%s.json", sendTo, id)
		writeCmd := fmt.Sprintf("cat > %s << 'MSGEOF'\n%s\nMSGEOF", path, string(data))
		writeArgs := []string{"exec", u.ContainerID, "sh", "-c", writeCmd}
		if out, err := exec.Command("docker", writeArgs...).CombinedOutput(); err != nil {
			return fmt.Errorf("error: cannot write message.\n%s\n%w", string(out), err)
		}

		s.Blank()
		s.Done("Sent message", fmt.Sprintf("%s → %s", sendFrom, sendTo))
		s.Blank()

		return nil
	},
}
