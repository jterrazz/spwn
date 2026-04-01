package architect

import (
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
