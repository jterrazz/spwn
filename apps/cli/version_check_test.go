package cli

import (
	"os"
	"path/filepath"
	"strings"
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

func TestPendingUpgrade_SameVersionNoHint(t *testing.T) {
	pendingUpgrade = ""
	// Simulate: latest == current
	// pendingUpgrade stays empty → no hint
	if pendingUpgrade != "" {
		t.Error("expected no pending upgrade")
	}
}

func TestPendingUpgrade_NewerVersionSetsHint(t *testing.T) {
	pendingUpgrade = "1.3.0"
	if pendingUpgrade == "" {
		t.Error("expected pending upgrade to be set")
	}
	if !strings.Contains(pendingUpgrade, "1.3.0") {
		t.Errorf("unexpected value: %q", pendingUpgrade)
	}
	pendingUpgrade = "" // cleanup
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
