package claw_test

import (
	"testing"

	"spwn.sh/core/universe/internal/claw"

	// Register all claw adapters
	_ "spwn.sh/core/universe/internal/claw/hermes"
	_ "spwn.sh/core/universe/internal/claw/zeroclaw"
)

func TestAllClawsRegistered(t *testing.T) {
	expected := []string{"zeroclaw", "hermes"}
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

func TestAllClawsStartStop(t *testing.T) {
	names := []string{"zeroclaw", "hermes"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			c, _ := claw.Get(name)
			start := c.StartCommand()
			stop := c.StopCommand()
			if len(start) == 0 {
				t.Error("empty start command")
			}
			if len(stop) == 0 {
				t.Error("empty stop command")
			}
			// Start command should begin with the claw binary name
			if start[0] == "" {
				t.Error("empty binary in start command")
			}
		})
	}
}

func TestAllClawsMetadata(t *testing.T) {
	names := []string{"zeroclaw", "hermes"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			c, _ := claw.Get(name)
			if c.BaseImage() == "" {
				t.Error("empty base image")
			}
			if len(c.InstallCommands()) == 0 {
				t.Error("no install commands")
			}
			if len(c.SystemPackages()) == 0 {
				t.Error("no system packages")
			}
		})
	}
}

func TestGetUnknownClaw(t *testing.T) {
	_, err := claw.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown claw, got nil")
	}
}

func TestAllReturnsAllClaws(t *testing.T) {
	all := claw.All()
	if len(all) < 2 {
		t.Errorf("expected at least 2 claws, got %d", len(all))
	}
}
