package claw_test

import (
	"testing"

	"spwn.sh/core/universe/internal/claw"

	// Register all claw adapters
	_ "spwn.sh/core/universe/internal/claw/hermes"
	_ "spwn.sh/core/universe/internal/claw/openclaw"
	_ "spwn.sh/core/universe/internal/claw/zeroclaw"
)

func TestAllClawsRegistered(t *testing.T) {
	expected := []string{"zeroclaw", "hermes", "openclaw"}
	for _, name := range expected {
		c, err := claw.Get(name)
		if err != nil {
			t.Errorf("claw %q not registered: %v", name, err)
			continue
		}
		if c.Name() != name {
			t.Errorf("expected %q, got %q", name, c.Name())
		}
	}
}

func TestGetUnknownClaw(t *testing.T) {
	_, err := claw.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown claw, got nil")
	}
}
