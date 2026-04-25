package gh

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestIsAuthenticated_FreshHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	if IsAuthenticated() {
		t.Fatal("expected unauthenticated on fresh tempdir")
	}
}

func TestIsAuthenticated_RequiresInlineToken(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	if err := os.MkdirAll(CacheDir(), 0o700); err != nil {
		t.Fatal(err)
	}

	// hosts.yml that points at the keyring is functionally useless
	// inside a container — IsAuthenticated must reject it.
	keyringOnly := []byte("github.com:\n    user: jterrazz\n")
	if err := os.WriteFile(HostsPath(), keyringOnly, 0o600); err != nil {
		t.Fatal(err)
	}
	if IsAuthenticated() {
		t.Error("expected unauthenticated when hosts.yml has no oauth_token (keyring-only)")
	}

	// Now write the inline-token form.
	withToken := []byte("github.com:\n    oauth_token: gho_x\n    user: jterrazz\n    git_protocol: https\n")
	if err := os.WriteFile(HostsPath(), withToken, 0o600); err != nil {
		t.Fatal(err)
	}
	if !IsAuthenticated() {
		t.Error("expected authenticated after writing hosts.yml with oauth_token")
	}
}

func TestLogout_RemovesCache(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	if err := os.MkdirAll(CacheDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(HostsPath(), []byte("github.com:\n    oauth_token: gho_x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !IsAuthenticated() {
		t.Fatal("expected authenticated before logout")
	}
	if err := Logout(); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if IsAuthenticated() {
		t.Error("expected unauthenticated after Logout")
	}
	if _, err := os.Stat(filepath.Dir(HostsPath())); err == nil {
		t.Error("expected cache dir to be removed")
	}
}

func TestLogout_NotLoggedIn(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	err := Logout()
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}
