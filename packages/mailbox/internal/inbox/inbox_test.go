package inbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSend(t *testing.T) {
	dir := t.TempDir()
	msg, err := Send(dir, "morpheus", "neo", "implement webhooks", "task")
	if err != nil {
		t.Fatal(err)
	}
	if msg.From != "morpheus" {
		t.Error("wrong from")
	}
	if msg.To != "neo" {
		t.Error("wrong to")
	}
	if msg.Status != "unread" {
		t.Error("should be unread")
	}

	// Verify file exists
	files, _ := os.ReadDir(filepath.Join(dir, "neo"))
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
}

func TestCheck(t *testing.T) {
	dir := t.TempDir()
	Send(dir, "morpheus", "neo", "task 1", "task")
	Send(dir, "trinity", "neo", "question", "question")

	msgs, err := Check(dir, "neo")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2, got %d", len(msgs))
	}
}

func TestCheckUnread(t *testing.T) {
	dir := t.TempDir()
	Send(dir, "morpheus", "neo", "task 1", "task")
	msg2, _ := Send(dir, "trinity", "neo", "task 2", "task")
	MarkRead(dir, "neo", msg2.ID)

	unread, err := CheckUnread(dir, "neo")
	if err != nil {
		t.Fatal(err)
	}
	if len(unread) != 1 {
		t.Errorf("expected 1 unread, got %d", len(unread))
	}
}

func TestMarkRead(t *testing.T) {
	dir := t.TempDir()
	msg, _ := Send(dir, "morpheus", "neo", "task", "task")
	MarkRead(dir, "neo", msg.ID)

	msgs, _ := Check(dir, "neo")
	if msgs[0].Status != "read" {
		t.Error("should be read")
	}
}

func TestListAll(t *testing.T) {
	dir := t.TempDir()
	Send(dir, "morpheus", "neo", "task 1", "task")
	Send(dir, "neo", "morpheus", "reply", "reply")

	all, err := ListAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}
