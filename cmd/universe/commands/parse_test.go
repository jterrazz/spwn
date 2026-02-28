package commands

import (
	"testing"
)

func TestParseInteractions_Valid(t *testing.T) {
	raw := []string{"mcp/slack:slack-send:send,read"}
	interactions, err := parseInteractions(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(interactions))
	}

	ia := interactions[0]
	if ia.Source != "mcp/slack" {
		t.Errorf("expected source mcp/slack, got %s", ia.Source)
	}
	if ia.As != "slack-send" {
		t.Errorf("expected as slack-send, got %s", ia.As)
	}
	if len(ia.Capabilities) != 2 || ia.Capabilities[0] != "send" || ia.Capabilities[1] != "read" {
		t.Errorf("unexpected capabilities: %v", ia.Capabilities)
	}
}

func TestParseInteractions_NoCaps(t *testing.T) {
	raw := []string{"mcp/db:db-query"}
	interactions, err := parseInteractions(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(interactions) != 1 {
		t.Fatalf("expected 1 interaction, got %d", len(interactions))
	}
	if len(interactions[0].Capabilities) != 0 {
		t.Errorf("expected no capabilities, got %v", interactions[0].Capabilities)
	}
}

func TestParseInteractions_Multiple(t *testing.T) {
	raw := []string{"mcp/a:cmd-a", "mcp/b:cmd-b:read"}
	interactions, err := parseInteractions(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(interactions) != 2 {
		t.Fatalf("expected 2 interactions, got %d", len(interactions))
	}
}

func TestParseInteractions_InvalidFormat(t *testing.T) {
	raw := []string{"invalid-no-colon"}
	_, err := parseInteractions(raw)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestParseInteractions_EmptySource(t *testing.T) {
	raw := []string{":cmd-a"}
	_, err := parseInteractions(raw)
	if err == nil {
		t.Error("expected error for empty source")
	}
}

func TestParseInteractions_EmptyAs(t *testing.T) {
	raw := []string{"mcp/a:"}
	_, err := parseInteractions(raw)
	if err == nil {
		t.Error("expected error for empty as")
	}
}

func TestParseInteractions_Empty(t *testing.T) {
	interactions, err := parseInteractions(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(interactions) != 0 {
		t.Errorf("expected 0 interactions, got %d", len(interactions))
	}
}
