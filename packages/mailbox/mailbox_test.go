package mailbox

import (
	"testing"
)

func TestSend_CreatesMessage(t *testing.T) {
	dir := t.TempDir()

	msg, err := Send(dir, "morpheus", "neo", "implement webhooks", "task")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if msg.From != "morpheus" {
		t.Errorf("From = %q, want %q", msg.From, "morpheus")
	}
	if msg.To != "neo" {
		t.Errorf("To = %q, want %q", msg.To, "neo")
	}
	if msg.Content != "implement webhooks" {
		t.Errorf("Content = %q, want %q", msg.Content, "implement webhooks")
	}
	if msg.Type != "task" {
		t.Errorf("Type = %q, want %q", msg.Type, "task")
	}
	if msg.Status != "unread" {
		t.Errorf("Status = %q, want %q", msg.Status, "unread")
	}
	if msg.ID == "" {
		t.Error("Expected non-empty message ID")
	}
}

func TestCheck_ReturnsAllMessages(t *testing.T) {
	dir := t.TempDir()

	Send(dir, "morpheus", "neo", "task 1", "task")
	Send(dir, "trinity", "neo", "question", "question")

	msgs, err := Check(dir, "neo")
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(msgs))
	}
}

func TestCheck_EmptyInbox(t *testing.T) {
	dir := t.TempDir()

	msgs, err := Check(dir, "neo")
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(msgs))
	}
}

func TestCheckUnread_FiltersReadMessages(t *testing.T) {
	dir := t.TempDir()

	Send(dir, "morpheus", "neo", "task 1", "task")
	msg2, _ := Send(dir, "trinity", "neo", "task 2", "task")

	MarkRead(dir, "neo", msg2.ID)

	unread, err := CheckUnread(dir, "neo")
	if err != nil {
		t.Fatalf("CheckUnread failed: %v", err)
	}
	if len(unread) != 1 {
		t.Errorf("Expected 1 unread message, got %d", len(unread))
	}
}

func TestMarkRead_ChangesStatus(t *testing.T) {
	dir := t.TempDir()

	msg, _ := Send(dir, "morpheus", "neo", "task", "task")

	err := MarkRead(dir, "neo", msg.ID)
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	msgs, _ := Check(dir, "neo")
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Status != "read" {
		t.Errorf("Status = %q, want %q", msgs[0].Status, "read")
	}
}

func TestMarkRead_NonexistentMessage(t *testing.T) {
	dir := t.TempDir()

	err := MarkRead(dir, "neo", "nonexistent-msg-id")
	if err == nil {
		t.Fatal("Expected error for nonexistent message, got nil")
	}
}

func TestListAll_AcrossInboxes(t *testing.T) {
	dir := t.TempDir()

	Send(dir, "morpheus", "neo", "task 1", "task")
	Send(dir, "neo", "morpheus", "reply", "reply")
	Send(dir, "trinity", "neo", "help", "question")

	all, err := ListAll(dir)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(all))
	}
}

func TestListAll_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	all, err := ListAll(dir)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(all))
	}
}
