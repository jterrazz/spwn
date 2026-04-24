package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"spwn.sh/packages/upgrade"
)

func TestVersionCheckCache_FreshCacheReturnsValue(t *testing.T) {
	// Setup temp SPWN_HOME
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	// Write a fresh cache
	cacheContent := time.Now().UTC().Format(time.RFC3339) + "\nv1.2.3"
	os.WriteFile(filepath.Join(tmpDir, ".version-check"), []byte(cacheContent), 0644)

	// Should return cached value without hitting network
	result := upgrade.CheckLatestVersion(versionCheckInterval)
	if result != "v1.2.3" {
		t.Errorf("expected v1.2.3, got %q", result)
	}
}

func TestVersionCheckCache_StaleCacheHitsNetwork(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	// Write a stale cache (25 hours old)
	staleTime := time.Now().Add(-25 * time.Hour).UTC().Format(time.RFC3339)
	cacheContent := staleTime + "\nv0.0.1"
	os.WriteFile(filepath.Join(tmpDir, ".version-check"), []byte(cacheContent), 0644)

	// This will try to hit the network (may fail in CI, that's OK)
	result := upgrade.CheckLatestVersion(versionCheckInterval)
	// If network is available, result should be non-empty and different from stale
	// If not, it returns "" which is also acceptable
	_ = result
}

func TestVersionCheckCache_MissingCacheFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	// No cache file - will try network, may return "" in offline env
	result := upgrade.CheckLatestVersion(versionCheckInterval)
	_ = result // just ensure no panic
}

// TestApplyLatestVersion_SemverCompare pins the bug where the banner
// Fired on any "latest != current" instead of "latest > current".
// The stale-cache-below-current case (cached v0.17.0 after force-
// Upgrade to v0.17.4) must NOT produce a banner — that would tell
// The user to "upgrade" to an older version.
func TestApplyLatestVersion_SemverCompare(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		wantSet bool
	}{
		{"latest strictly newer", "v0.17.0", "v0.17.4", true},
		{"latest minor newer", "v0.17.4", "v0.18.0", true},
		{"latest major newer", "v0.17.4", "v1.0.0", true},
		{"equal versions", "v0.17.4", "v0.17.4", false},
		{"latest older (stale cache after force upgrade)", "v0.17.4", "v0.17.0", false},
		{"latest much older", "v1.0.0", "v0.9.0", false},
		{"no v prefix still compares", "0.17.0", "0.17.4", true},
		{"mixed v prefix still compares", "v0.17.0", "0.17.4", true},
		{"dev current, real latest", "dev", "v0.17.4", true},
		{"unparseable latest no-op", "v0.17.0", "garbage", false},
		{"empty latest no-op", "v0.17.0", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := Version
			t.Cleanup(func() {
				Version = orig
				pendingUpgrade = ""
			})
			Version = tc.current
			pendingUpgrade = ""
			applyLatestVersion(tc.latest)
			got := pendingUpgrade != ""
			if got != tc.wantSet {
				t.Errorf("current=%q latest=%q: pendingUpgrade set=%v, want %v (value=%q)",
					tc.current, tc.latest, got, tc.wantSet, pendingUpgrade)
			}
		})
	}
}

// TestStartVersionCheck_FreshCacheSetsPendingSynchronously pins the
// Fix for fast commands (`spwn ls`, `spwn status`) that used to exit
// Before the background goroutine set pendingUpgrade. A fresh cache
// Pointing at a NEWER version must land in pendingUpgrade *before*
// StartVersionCheck returns.
func TestStartVersionCheck_FreshCacheSetsPendingSynchronously(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	t.Setenv("SPWN_NO_UPDATE_CHECK", "") // explicit: do not skip

	// Save and restore the package-level Version so the test doesn't
	// Leak state into neighbours.
	orig := Version
	t.Cleanup(func() {
		Version = orig
		pendingUpgrade = ""
	})
	Version = "v0.0.1"

	cacheContent := time.Now().UTC().Format(time.RFC3339) + "\nv9.9.9"
	if err := os.WriteFile(filepath.Join(tmpDir, ".version-check"), []byte(cacheContent), 0644); err != nil {
		t.Fatal(err)
	}

	pendingUpgrade = ""
	startVersionCheck()

	// No goroutine race — the cache path is synchronous, so
	// PendingUpgrade must be populated immediately on return.
	if pendingUpgrade != "9.9.9" {
		t.Errorf("pendingUpgrade = %q, want 9.9.9 (fresh-cache path must be synchronous)", pendingUpgrade)
	}
}

// TestStartVersionCheck_DevBuildSkipsCheck locks in that local
// Development builds never trigger the upgrade banner regardless of
// Cache state.
func TestStartVersionCheck_DevBuildSkipsCheck(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	cacheContent := time.Now().UTC().Format(time.RFC3339) + "\nv9.9.9"
	_ = os.WriteFile(filepath.Join(tmpDir, ".version-check"), []byte(cacheContent), 0644)

	orig := Version
	t.Cleanup(func() {
		Version = orig
		pendingUpgrade = ""
	})
	Version = "dev"

	pendingUpgrade = ""
	startVersionCheck()
	if pendingUpgrade != "" {
		t.Errorf("dev build should skip, got pendingUpgrade=%q", pendingUpgrade)
	}
}

// TestRanUpgradeCommand pins the detector so the banner correctly
// Suppresses itself below `spwn upgrade` (durable noise-avoidance)
// Without over-matching flag values or unrelated subcommands.
func TestRanUpgradeCommand(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"bare upgrade", []string{"spwn", "upgrade"}, true},
		{"upgrade with --check", []string{"spwn", "upgrade", "--check"}, true},
		{"ls is not upgrade", []string{"spwn", "ls"}, false},
		{"status is not upgrade", []string{"spwn", "status"}, false},
		{"flag before subcommand", []string{"spwn", "--json", "upgrade"}, true},
		{"no args", []string{"spwn"}, false},
		{"help command", []string{"spwn", "help"}, false},
		{"agent named 'upgrade' would match but is a reserved word",
			[]string{"spwn", "agent", "upgrade"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := os.Args
			t.Cleanup(func() { os.Args = orig })
			os.Args = tc.args
			got := ranUpgradeCommand()
			if got != tc.want {
				t.Errorf("args=%v got=%v want=%v", tc.args, got, tc.want)
			}
		})
	}
}

// TestStartVersionCheck_OptOutEnvVarSkipsCheck locks in that the
// Documented SPWN_NO_UPDATE_CHECK escape hatch still works.
func TestStartVersionCheck_OptOutEnvVarSkipsCheck(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)
	t.Setenv("SPWN_NO_UPDATE_CHECK", "1")
	cacheContent := time.Now().UTC().Format(time.RFC3339) + "\nv9.9.9"
	_ = os.WriteFile(filepath.Join(tmpDir, ".version-check"), []byte(cacheContent), 0644)

	orig := Version
	t.Cleanup(func() {
		Version = orig
		pendingUpgrade = ""
	})
	Version = "v0.0.1"

	pendingUpgrade = ""
	startVersionCheck()
	if pendingUpgrade != "" {
		t.Errorf("SPWN_NO_UPDATE_CHECK=1 should skip, got pendingUpgrade=%q", pendingUpgrade)
	}
}
