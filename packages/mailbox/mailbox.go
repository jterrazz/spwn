// Package mailbox provides agent-to-agent communication via a
// filesystem-based inbox system.
package mailbox

import (
	"spwn.sh/packages/mailbox/internal/inbox"
	"spwn.sh/packages/mailbox/internal/models"
)

// Message is a single inbox message exchanged between agents.
type Message = models.Message

// Send writes a message to the recipient's inbox directory.
func Send(inboxDir, from, to, content, msgType string) (*Message, error) {
	return inbox.Send(inboxDir, from, to, content, msgType)
}

// Check returns all messages for a specific agent.
func Check(inboxDir, agentName string) ([]Message, error) {
	return inbox.Check(inboxDir, agentName)
}

// CheckUnread returns only unread messages for an agent.
func CheckUnread(inboxDir, agentName string) ([]Message, error) {
	return inbox.CheckUnread(inboxDir, agentName)
}

// MarkRead marks a message as read.
func MarkRead(inboxDir, agentName, messageID string) error {
	return inbox.MarkRead(inboxDir, agentName, messageID)
}

// ListAll returns all messages across all inboxes.
func ListAll(inboxDir string) ([]Message, error) {
	return inbox.ListAll(inboxDir)
}
