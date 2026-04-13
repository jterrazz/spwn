package models

import "time"

// Message represents a single message in an agent's inbox.
type Message struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`   // "task", "reply", "question", "announcement"
	Content   string    `json:"content"`
	Status    string    `json:"status"` // "unread", "read", "delivered"
}
