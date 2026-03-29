package universe

import (
	"testing"
	"time"
)

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
	if got != "unknown" {
		t.Errorf("timeAgo(zero) = %q, want %q", got, "unknown")
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
