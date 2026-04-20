package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_missingFileReturnsEmpty(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	c := LoadConfig()
	if c == nil {
		t.Fatal("LoadConfig returned nil")
	}
	if c.Version != currentConfigVersion {
		t.Errorf("Version = %d, want %d", c.Version, currentConfigVersion)
	}
	if len(c.Providers) != 0 {
		t.Errorf("Providers = %v, want empty map", c.Providers)
	}
	if c.Pref(ProviderAnthropic).Disabled {
		t.Error("fresh config should not disable any provider")
	}
	if c.Pref(ProviderAnthropic).Method != "" {
		t.Error("fresh config should have empty Method (auto-select)")
	}
}

func TestSaveAndLoadConfig_roundtrip(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())

	c := &Config{
		Version: currentConfigVersion,
		Providers: map[Provider]ProviderPref{
			ProviderAnthropic: {Method: MethodOAuth, Disabled: false},
			ProviderOpenAI:    {Method: MethodAPIKey, Disabled: true},
		},
	}
	if err := SaveConfig(c); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	got := LoadConfig()
	if got.Pref(ProviderAnthropic).Method != MethodOAuth {
		t.Errorf("anthropic method: got %q, want oauth", got.Pref(ProviderAnthropic).Method)
	}
	if got.Pref(ProviderOpenAI).Method != MethodAPIKey {
		t.Errorf("openai method: got %q, want api_key", got.Pref(ProviderOpenAI).Method)
	}
	if !got.Pref(ProviderOpenAI).Disabled {
		t.Error("openai should be disabled")
	}
	if got.Pref(ProviderAnthropic).Disabled {
		t.Error("anthropic should not be disabled")
	}
}

func TestDisableProvider_persists(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())

	if IsProviderDisabled(ProviderAnthropic) {
		t.Fatal("fresh home: provider unexpectedly disabled")
	}
	if err := DisableProvider(ProviderAnthropic); err != nil {
		t.Fatalf("DisableProvider: %v", err)
	}
	if !IsProviderDisabled(ProviderAnthropic) {
		t.Error("after DisableProvider, IsProviderDisabled should be true")
	}

	// A second LoadConfig (fresh instance) must still see the disabled flag.
	c := LoadConfig()
	if !c.Pref(ProviderAnthropic).Disabled {
		t.Error("reload: disabled flag did not persist")
	}
}

func TestEnableProvider_reverses(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	if err := DisableProvider(ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	if err := EnableProvider(ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	if IsProviderDisabled(ProviderOpenAI) {
		t.Error("EnableProvider should clear the disabled flag")
	}
}

func TestMigrateDisabledMarkers_absorbsLegacyFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	// Simulate a legacy install: `.disabled-anthropic` sitting in the
	// credentials dir from an older spwn that hadn't moved to auth.yaml.
	credsDir := filepath.Join(tmp, "credentials")
	if err := os.MkdirAll(credsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(credsDir, ".disabled-anthropic")
	if err := os.WriteFile(marker, []byte("disabled"), 0o600); err != nil {
		t.Fatal(err)
	}

	c := LoadConfig()
	if !c.Pref(ProviderAnthropic).Disabled {
		t.Error("legacy marker should have migrated to auth.yaml")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("legacy marker should have been deleted post-migration; stat err=%v", err)
	}

	// Ensure it stayed persisted across a fresh load (migration ran once).
	again := LoadConfig()
	if !again.Pref(ProviderAnthropic).Disabled {
		t.Error("post-migration reload should still report disabled")
	}
}

func TestSetActiveMethod_persists(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())

	if got := ActiveMethod(ProviderAnthropic); got != "" {
		t.Errorf("fresh home: ActiveMethod = %q, want empty", got)
	}
	if err := SetActiveMethod(ProviderAnthropic, MethodAPIKey); err != nil {
		t.Fatal(err)
	}
	if got := ActiveMethod(ProviderAnthropic); got != MethodAPIKey {
		t.Errorf("ActiveMethod = %q, want api_key", got)
	}

	// Reset to auto-select.
	if err := SetActiveMethod(ProviderAnthropic, ""); err != nil {
		t.Fatal(err)
	}
	if got := ActiveMethod(ProviderAnthropic); got != "" {
		t.Errorf("after reset: ActiveMethod = %q, want empty", got)
	}
}

func TestDisableProvider_doesNotTouchMethod(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	if err := SetActiveMethod(ProviderOpenAI, MethodOAuth); err != nil {
		t.Fatal(err)
	}
	if err := DisableProvider(ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	// Method survives a disable — re-enabling restores the prior choice
	// rather than forcing the user to pick again.
	if got := ActiveMethod(ProviderOpenAI); got != MethodOAuth {
		t.Errorf("DisableProvider clobbered Method: got %q, want oauth", got)
	}
}
