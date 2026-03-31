package journal

import (
	"fmt"
	"testing"
	"time"
)

func TestAppend(t *testing.T) {
	tmp := t.TempDir()
	err := Append(tmp, "w-default-12345", 0, 3*time.Minute+12*time.Second)
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	entries, err := List(tmp, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].UniverseID != "w-default-12345" {
		t.Errorf("expected universe w-default-12345, got %s", entries[0].UniverseID)
	}
	if entries[0].ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", entries[0].ExitCode)
	}
	if entries[0].Outcome != "completed" {
		t.Errorf("expected completed, got %s", entries[0].Outcome)
	}
}

func TestAppendFailed(t *testing.T) {
	tmp := t.TempDir()
	Append(tmp, "w-test-99999", 1, 5*time.Second)

	entries, _ := List(tmp, 10)
	if entries[0].Outcome != "failed" {
		t.Errorf("expected failed, got %s", entries[0].Outcome)
	}
	if entries[0].ExitCode != 1 {
		t.Errorf("expected exit 1, got %d", entries[0].ExitCode)
	}
}

func TestListNewestFirst(t *testing.T) {
	tmp := t.TempDir()

	Append(tmp, "w-first-00001", 0, time.Second)
	time.Sleep(1100 * time.Millisecond) // ensure different second in filename
	Append(tmp, "w-second-00002", 0, time.Second)

	entries, _ := List(tmp, 10)
	if len(entries) != 2 {
		t.Fatalf("expected 2, got %d", len(entries))
	}
	// Newest first
	if entries[0].UniverseID != "w-second-00002" {
		t.Errorf("expected newest first, got %s", entries[0].UniverseID)
	}
}

func TestListLimit(t *testing.T) {
	tmp := t.TempDir()
	// Create entries with different second timestamps
	for i := 0; i < 3; i++ {
		Append(tmp, fmt.Sprintf("w-test-%05d", i), 0, time.Second)
		time.Sleep(1100 * time.Millisecond)
	}

	entries, _ := List(tmp, 2)
	if len(entries) != 2 {
		t.Errorf("expected 2 (limited), got %d", len(entries))
	}
}

func TestListEmpty(t *testing.T) {
	tmp := t.TempDir()
	entries, err := List(tmp, 10)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0, got %d", len(entries))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{65 * time.Minute, "1h5m"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
