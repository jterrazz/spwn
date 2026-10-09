package gh

import (
	"path/filepath"
	"testing"
)

func TestCacheDir_LandsUnderCredentials(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	got := CacheDir()
	want := filepath.Join(tmp, "credentials", "gh")
	if got != want {
		t.Errorf("CacheDir = %q, want %q", got, want)
	}
}

func TestHostsPath_LandsUnderCacheDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	got := HostsPath()
	want := filepath.Join(tmp, "credentials", "gh", "hosts.yml")
	if got != want {
		t.Errorf("HostsPath = %q, want %q", got, want)
	}
}
