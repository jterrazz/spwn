package world

import (
	"strings"
	"testing"
	"time"

	"spwn.sh/packages/project"
	"spwn.sh/packages/world"
)

func TestBuildDeclaredRows_StopAndRunning(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	declared := map[string]project.World{
		"alpha": {Agents: []string{"a1", "a2"}},
		"beta":  {Agents: []string{"b1"}},
		"gamma": {},
	}
	running := map[string]world.World{
		"alpha": {CreatedAt: now.Add(-5 * time.Minute)},
	}

	rows := buildDeclaredRows(declared, running, now)
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}
	// Sorted alphabetically: alpha, beta, gamma.
	if rows[0][0] != "alpha" || rows[1][0] != "beta" || rows[2][0] != "gamma" {
		t.Errorf("unexpected row order: %+v", rows)
	}
	if !strings.Contains(rows[0][1], "running") || !strings.Contains(rows[0][1], "5m") {
		t.Errorf("alpha STATUS = %q, want running (5m)", rows[0][1])
	}
	if rows[0][2] != "a1, a2" {
		t.Errorf("alpha AGENTS = %q, want %q", rows[0][2], "a1, a2")
	}
	if !strings.Contains(rows[1][1], "stopped") {
		t.Errorf("beta STATUS = %q, want stopped", rows[1][1])
	}
	if rows[1][2] != "b1" {
		t.Errorf("beta AGENTS = %q, want %q", rows[1][2], "b1")
	}
	if rows[2][2] != "\u2014" {
		t.Errorf("gamma AGENTS (no agents) = %q, want em dash", rows[2][2])
	}
}

func TestDurationSince(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"seconds", 30 * time.Second, "30s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"hours", 3 * time.Hour, "3h"},
		{"days", 48 * time.Hour, "2d"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := durationSince(now, now.Add(-tc.ago)); got != tc.want {
				t.Errorf("durationSince(%v) = %q, want %q", tc.ago, got, tc.want)
			}
		})
	}
	if got := durationSince(now, time.Time{}); got != "?" {
		t.Errorf("durationSince(zero) = %q, want ?", got)
	}
}

func TestTimeAgo_Seconds(t *testing.T) {
	got := timeAgo(time.Now().Add(-30 * time.Second))
	if got != "30s ago" {
		t.Errorf("timeAgo(30s) = %q, want %q", got, "30s ago")
	}
}

func TestTimeAgo_Minutes(t *testing.T) {
	got := timeAgo(time.Now().Add(-5 * time.Minute))
	if got != "5m ago" {
		t.Errorf("timeAgo(5m) = %q, want %q", got, "5m ago")
	}
}

func TestTimeAgo_Hours(t *testing.T) {
	got := timeAgo(time.Now().Add(-2 * time.Hour))
	if got != "2h ago" {
		t.Errorf("timeAgo(2h) = %q, want %q", got, "2h ago")
	}
}

func TestTimeAgo_Days(t *testing.T) {
	got := timeAgo(time.Now().Add(-48 * time.Hour))
	if got != "2d ago" {
		t.Errorf("timeAgo(48h) = %q, want %q", got, "2d ago")
	}
}

func TestTimeAgo_Zero(t *testing.T) {
	got := timeAgo(time.Time{})
	if got != "\u2014" {
		t.Errorf("timeAgo(zero) = %q, want %q", got, "\u2014")
	}
}

func TestTimeAgo_Boundaries(t *testing.T) {
	tests := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"just under a minute", 59 * time.Second, "59s ago"},
		{"exactly one minute", 60 * time.Second, "1m ago"},
		{"just under an hour", 59 * time.Minute, "59m ago"},
		{"exactly one hour", 60 * time.Minute, "1h ago"},
		{"just under a day", 23 * time.Hour, "23h ago"},
		{"exactly one day", 24 * time.Hour, "1d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeAgo(time.Now().Add(-tt.ago))
			if got != tt.want {
				t.Errorf("timeAgo(%v ago) = %q, want %q", tt.ago, got, tt.want)
			}
		})
	}
}
