package channel

import (
	"context"
	"testing"
)

func TestCLIChannelName(t *testing.T) {
	c := NewCLI()
	if c.Name() != "cli" {
		t.Errorf("Name() = %q, want %q", c.Name(), "cli")
	}
}

func TestCLIChannelReceiveError(t *testing.T) {
	c := NewCLI()
	ch, err := c.Receive(context.Background())
	if err == nil {
		t.Error("Receive() expected error, got nil")
	}
	if ch != nil {
		t.Error("Receive() expected nil channel")
	}
}

func TestCLIChannelClose(t *testing.T) {
	c := NewCLI()
	if err := c.Close(); err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}
}
