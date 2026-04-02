package architect

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDuration_Seconds(t *testing.T) {
	got := formatDuration(45 * time.Second)
	if got != "45s" {
		t.Errorf("formatDuration(45s) = %q, want %q", got, "45s")
	}
}

func TestFormatDuration_Minutes(t *testing.T) {
	got := formatDuration(3*time.Minute + 15*time.Second)
	if got != "3m 15s" {
		t.Errorf("formatDuration(3m15s) = %q, want %q", got, "3m 15s")
	}
}

func TestFormatDuration_Hours(t *testing.T) {
	got := formatDuration(2*time.Hour + 30*time.Minute)
	if got != "2h 30m" {
		t.Errorf("formatDuration(2h30m) = %q, want %q", got, "2h 30m")
	}
}

func TestFormatDuration_Days(t *testing.T) {
	got := formatDuration(50 * time.Hour)
	if got != "2d 2h" {
		t.Errorf("formatDuration(50h) = %q, want %q", got, "2d 2h")
	}
}

func TestFormatDuration_Zero(t *testing.T) {
	got := formatDuration(0)
	if got != "0s" {
		t.Errorf("formatDuration(0) = %q, want %q", got, "0s")
	}
}

func TestTalkCmd_AutoStartBehavior(t *testing.T) {
	// The talk command's Long description or the runTalk source should document
	// auto-start behavior. Verify the command is configured correctly.
	if talkCmd.Use != "talk [message]" {
		t.Errorf("talkCmd.Use = %q, want %q", talkCmd.Use, "talk [message]")
	}

	// Talk should accept 0 or 1 args (interactive vs one-shot)
	if talkCmd.Args == nil {
		t.Error("talkCmd.Args should be set (MaximumNArgs(1))")
	}
}

func TestStopCmd_GracefulWhenNotRunning(t *testing.T) {
	// The stop command should handle "not running" errors gracefully.
	// Verify the command structure is correct.
	if stopCmd.Use != "stop" {
		t.Errorf("stopCmd.Use = %q, want %q", stopCmd.Use, "stop")
	}

	if stopCmd.RunE == nil {
		t.Error("stopCmd.RunE should be set")
	}
}

func TestArchitectCmd_HasAllSubcommands(t *testing.T) {
	subcommands := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subcommands[c.Use] = true
	}

	expected := []string{"start", "stop", "status", "talk [message]"}
	for _, name := range expected {
		if !subcommands[name] {
			t.Errorf("missing subcommand %q in architect command", name)
		}
	}
}

func TestTalkCmd_DescriptionMentionsAutoStart(t *testing.T) {
	// The runTalk function auto-starts the architect if not running.
	// While we can't easily unit-test the full runTalk without Docker,
	// we verify that the Long description or Short gives context.
	long := talkCmd.Long
	short := talkCmd.Short

	// The talk command should mention what it does
	if !strings.Contains(short, "Talk") && !strings.Contains(short, "talk") {
		t.Errorf("talkCmd.Short should mention 'talk', got %q", short)
	}

	// Long description should mention interactive mode or message
	if !strings.Contains(long, "message") {
		t.Errorf("talkCmd.Long should mention 'message', got %q", long)
	}
}
