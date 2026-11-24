package upgrade

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fakeReleasesCurl(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	script := "#!/bin/sh\necho '{\"tag_name\": \"v9.9.9\"}'\n"
	if err := os.WriteFile(filepath.Join(bin, "curl"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
}

func TestCheckLatestVersion_FreshHomeWritesNothing(t *testing.T) {
	fakeReleasesCurl(t)
	home := filepath.Join(t.TempDir(), "spwn-home")
	t.Setenv("SPWN_HOME", home)

	if got := CheckLatestVersion(time.Hour); got != "v9.9.9" {
		t.Errorf("CheckLatestVersion = %q, want v9.9.9", got)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Errorf("the version check created %s (stat err %v)", home, err)
	}
}

func TestCheckLatestVersion_CachesInAnExistingHome(t *testing.T) {
	fakeReleasesCurl(t)
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)

	if got := CheckLatestVersion(time.Hour); got != "v9.9.9" {
		t.Errorf("CheckLatestVersion = %q, want v9.9.9", got)
	}
	if got := LatestVersionFromCache(time.Hour); got != "v9.9.9" {
		t.Errorf("LatestVersionFromCache = %q, want v9.9.9", got)
	}
}
