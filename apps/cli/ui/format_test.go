package ui

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "0s"},
		{"one second", time.Second, "1s"},
		{"thirty seconds", 30 * time.Second, "30s"},
		{"fifty nine seconds", 59 * time.Second, "59s"},
		{"one minute", time.Minute, "1m0s"},
		{"ninety seconds", 90 * time.Second, "1m30s"},
		{"five minutes", 5 * time.Minute, "5m0s"},
		{"five minutes thirty", 5*time.Minute + 30*time.Second, "5m30s"},
		{"fifty nine minutes", 59*time.Minute + 59*time.Second, "59m59s"},
		{"one hour", time.Hour, "1h0m"},
		{"one hour fifteen", time.Hour + 15*time.Minute, "1h15m"},
		{"two hours thirty", 2*time.Hour + 30*time.Minute, "2h30m"},
		{"twenty four hours", 24 * time.Hour, "24h0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.d)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
