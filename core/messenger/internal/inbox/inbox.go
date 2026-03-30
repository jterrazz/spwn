package inbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"spwn.sh/core/messenger/internal/models"
)

// Send creates a message file in the recipient's inbox.
func Send(inboxDir, from, to, content, msgType string) (*models.Message, error) {
	recipientDir := filepath.Join(inboxDir, to)
	if err := os.MkdirAll(recipientDir, 0755); err != nil {
		return nil, fmt.Errorf("create inbox for %s: %w", to, err)
	}

	now := time.Now()
	id := fmt.Sprintf("msg-%s-%s-%03d", from, now.Format("20060102-150405"), now.Nanosecond()/1000000)

	msg := &models.Message{
		ID:        id,
		From:      from,
		To:        to,
		Timestamp: now,
		Type:      msgType,
		Content:   content,
		Status:    "unread",
	}

	data, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return nil, err
	}

	path := filepath.Join(recipientDir, id+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}

	return msg, nil
}

// Check returns all messages for an agent, sorted newest first.
func Check(inboxDir, agentName string) ([]models.Message, error) {
	agentDir := filepath.Join(inboxDir, agentName)
	return readMessages(agentDir)
}

// CheckUnread returns only unread messages.
func CheckUnread(inboxDir, agentName string) ([]models.Message, error) {
	all, err := Check(inboxDir, agentName)
	if err != nil {
		return nil, err
	}
	var unread []models.Message
	for _, m := range all {
		if m.Status == "unread" {
			unread = append(unread, m)
		}
	}
	return unread, nil
}

// MarkRead updates a message's status to "read".
func MarkRead(inboxDir, agentName, messageID string) error {
	agentDir := filepath.Join(inboxDir, agentName)
	path := filepath.Join(agentDir, messageID+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("message %s not found: %w", messageID, err)
	}

	var msg models.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	msg.Status = "read"
	updated, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, updated, 0644)
}

// ListAll returns all messages from all agent inboxes.
func ListAll(inboxDir string) ([]models.Message, error) {
	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var all []models.Message
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		msgs, err := readMessages(filepath.Join(inboxDir, e.Name()))
		if err != nil {
			continue
		}
		all = append(all, msgs...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Timestamp.After(all[j].Timestamp)
	})
	return all, nil
}

func readMessages(dir string) ([]models.Message, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var msgs []models.Message
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var msg models.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		msgs = append(msgs, msg)
	}

	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].Timestamp.After(msgs[j].Timestamp)
	})
	return msgs, nil
}
