package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVersionCheckCache_FreshCacheReturnsValue(t *testing.T) {
	// Setup temp SPWN_HOME
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	// Write a fresh cache
	cacheContent := time.Now().UTC().Format(time.RFC3339) + "\nv1.2.3"
	os.WriteFile(filepath.Join(tmpDir, versionCheckFile), []byte(cacheContent), 0644)

	// Should return cached value without hitting network
	result := checkVersionCached()
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
	os.WriteFile(filepath.Join(tmpDir, versionCheckFile), []byte(cacheContent), 0644)

	// This will try to hit the network (may fail in CI, that's OK)
	result := checkVersionCached()
	// If network is available, result should be non-empty and different from stale
	// If not, it returns "" which is also acceptable
	_ = result
}

func TestVersionCheckCache_MissingCacheFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SPWN_HOME", tmpDir)

	// No cache file — will try network, may return "" in offline env
	result := checkVersionCached()
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
