package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteEnvFile_WithCredentials(t *testing.T) {
	dir := t.TempDir()
	creds := map[Provider]*Credential{
		ProviderAnthropic: {Provider: ProviderAnthropic, Type: CredTypeAPIKey, Token: "sk-ant-test", EnvVar: "ANTHROPIC_API_KEY"},
		ProviderOpenAI:    {Provider: ProviderOpenAI, Type: CredTypeNone, Token: ""},
	}

	if err := writeEnvFile(dir, creds); err != nil {
		t.Fatalf("writeEnvFile: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "ANTHROPIC_API_KEY") {
		t.Error("expected ANTHROPIC_API_KEY in .env")
	}
	if !strings.Contains(content, "sk-ant-test") {
		t.Error("expected token value in .env")
	}
	if strings.Contains(content, "OPENAI_API_KEY") {
		t.Error("should not include provider with no credentials")
	}
}

func TestWriteEnvFile_Empty(t *testing.T) {
	dir := t.TempDir()
	creds := map[Provider]*Credential{
		ProviderAnthropic: {Provider: ProviderAnthropic, Type: CredTypeNone, Token: ""},
	}

	if err := writeEnvFile(dir, creds); err != nil {
		t.Fatalf("writeEnvFile: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".env"))
	if len(strings.TrimSpace(string(data))) > 0 {
		t.Errorf("expected empty .env, got: %s", string(data))
	}
}

func TestWriteEnvFile_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	creds := map[Provider]*Credential{
		ProviderAnthropic: {Provider: ProviderAnthropic, Type: CredTypeAPIKey, Token: "test", EnvVar: "ANTHROPIC_API_KEY"},
	}

	if err := writeEnvFile(dir, creds); err != nil {
		t.Fatalf("writeEnvFile: %v", err)
	}

	// .tmp file should not remain
	if _, err := os.Stat(filepath.Join(dir, ".env.tmp")); err == nil {
		t.Error(".env.tmp should not exist after atomic write")
	}

	// .env should exist
	if _, err := os.Stat(filepath.Join(dir, ".env")); err != nil {
		t.Error(".env should exist")
	}
}

func TestSyncRuntimeFiles_CodexAuth(t *testing.T) {
	dir := t.TempDir()

	// Create a fake codex auth.json
	home := t.TempDir()
	t.Setenv("HOME", home)
	codexDir := filepath.Join(home, ".codex")
	os.MkdirAll(codexDir, 0755)
	os.WriteFile(filepath.Join(codexDir, "auth.json"), []byte(`{"tokens":{"access_token":"test"}}`), 0600)

	if err := syncRuntimeFiles(dir); err != nil {
		t.Fatalf("syncRuntimeFiles: %v", err)
	}

	// Verify copied
	data, err := os.ReadFile(filepath.Join(dir, "openai", "auth.json"))
	if err != nil {
		t.Fatalf("read copied auth.json: %v", err)
	}
	if !strings.Contains(string(data), "access_token") {
		t.Error("copied auth.json should contain token data")
	}
}

func TestWriteEnvFile_ExportFormat(t *testing.T) {
	dir := t.TempDir()
	creds := map[Provider]*Credential{
		ProviderAnthropic: {Provider: ProviderAnthropic, Type: CredTypeAPIKey, Token: "sk-test", EnvVar: "ANTHROPIC_API_KEY"},
	}

	writeEnvFile(dir, creds)
	data, _ := os.ReadFile(filepath.Join(dir, ".env"))

	// Should use export format so `source` works
	if !strings.Contains(string(data), "export ANTHROPIC_API_KEY=") {
		t.Errorf("expected export format, got: %s", string(data))
	}
}
