package platform_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/platform"
)

func TestLoadConfig_missingReturnsDefaults(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())

	cfg, err := platform.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	def := platform.DefaultConfig()
	if cfg.APIVersion != def.APIVersion {
		t.Errorf("APIVersion = %q, want %q", cfg.APIVersion, def.APIVersion)
	}
	if cfg.Runtime.DefaultBackend != def.Runtime.DefaultBackend {
		t.Errorf("DefaultBackend = %q, want %q", cfg.Runtime.DefaultBackend, def.Runtime.DefaultBackend)
	}
	if cfg.Telemetry.Enabled {
		t.Errorf("Telemetry.Enabled = true, want false (opt-in)")
	}
	if cfg.Update.Channel != "stable" {
		t.Errorf("Update.Channel = %q, want stable", cfg.Update.Channel)
	}
	if cfg.Onboarded {
		t.Errorf("Onboarded = true, want false on fresh install")
	}
}

func TestSaveLoadConfig_roundTrip(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())

	in := platform.Config{
		APIVersion: platform.CurrentConfigAPIVersion,
		Runtime: platform.RuntimeConfig{
			DefaultBackend:  "spwn:claude-code",
			DefaultProvider: "anthropic",
			DefaultModel:    "claude-4-7-sonnet",
		},
		Telemetry: platform.TelemetryConfig{Enabled: true},
		Update:    platform.UpdateConfig{Channel: "edge"},
		Onboarded: true,
	}
	if err := platform.SaveConfig(in); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	got, err := platform.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got != in {
		t.Errorf("round-trip mismatch:\n got=%+v\nwant=%+v", got, in)
	}
}

func TestSaveConfig_writesReadableHeader(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	if err := platform.SaveConfig(platform.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(platform.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "# spwn user config") {
		t.Errorf("config file missing comment header, got start: %q", string(data[:min(60, len(data))]))
	}
}

func TestSaveConfig_fillsAPIVersionIfMissing(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	// User forgot to set APIVersion; SaveConfig should stamp it.
	if err := platform.SaveConfig(platform.Config{Onboarded: true}); err != nil {
		t.Fatal(err)
	}
	got, err := platform.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.APIVersion != platform.CurrentConfigAPIVersion {
		t.Errorf("APIVersion not stamped: %q", got.APIVersion)
	}
}

func TestLoadConfig_malformedYAMLErrors(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	if err := os.MkdirAll(platform.BaseDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(platform.ConfigPath(), []byte("not: valid: yaml: [unbalanced"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := platform.LoadConfig()
	if err == nil {
		t.Fatal("expected parse error on malformed config, got nil")
	}
}

func TestConfigPath_respectsSPWN_HOME(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)
	want := filepath.Join(home, "config.yaml")
	if got := platform.ConfigPath(); got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}
