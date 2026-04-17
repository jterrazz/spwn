package migrations

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/platform"
)

func TestEnsureUserConfig_createsWhenMissing(t *testing.T) {
	base := t.TempDir()
	if err := EnsureUserConfig.Apply(context.Background(), base); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(base, platform.ConfigFileName))
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "apiVersion: "+platform.CurrentConfigAPIVersion) {
		t.Errorf("config missing apiVersion header:\n%s", content)
	}
	if !strings.Contains(content, "default_backend: spwn:claude-code") {
		t.Errorf("config missing default_backend:\n%s", content)
	}
	if !strings.Contains(content, "channel: stable") {
		t.Errorf("config missing update.channel:\n%s", content)
	}
}

func TestEnsureUserConfig_keepsExistingFile(t *testing.T) {
	base := t.TempDir()
	existing := "# user customisations preserved\napiVersion: spwn/v2\nonboarded: true\n"
	path := filepath.Join(base, platform.ConfigFileName)
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureUserConfig.Apply(context.Background(), base); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Errorf("existing config overwritten. got:\n%s\nwant:\n%s", got, existing)
	}
}

func TestEnsureUserConfig_idempotent(t *testing.T) {
	base := t.TempDir()
	if err := EnsureUserConfig.Apply(context.Background(), base); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	path := filepath.Join(base, platform.ConfigFileName)
	first, _ := os.ReadFile(path)

	if err := EnsureUserConfig.Apply(context.Background(), base); err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Errorf("second Apply changed file")
	}
}
