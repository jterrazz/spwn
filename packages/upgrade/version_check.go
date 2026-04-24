package upgrade

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"spwn.sh/packages/platform"
)

// CLIVersion is set at build time via -ldflags. Defaults to "dev" for local builds.
var CLIVersion = "dev"

const (
	versionCheckFile     = ".version-check"
	defaultCheckInterval = 24 * time.Hour
	githubReleasesURL    = "https://api.github.com/repos/jterrazz/spwn/releases/latest"
)

// VersionInfo contains current and latest version information.
type VersionInfo struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ReleaseURL      string `json:"releaseUrl"`
}

// InvalidateVersionCache removes the cached remote-version file so
// The next CheckLatestVersion / LatestVersionFromCache call performs
// A fresh network lookup. Called by `spwn upgrade` on successful
// Completion so the banner doesn't cling to the pre-upgrade "latest"
// Value until its 24-hour TTL expires. Safe to call when the cache
// File is missing — not-found errors are swallowed.
func InvalidateVersionCache() error {
	path := filepath.Join(platform.BaseDir(), versionCheckFile)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// LatestVersionFromCache returns the cached latest version without
// Making any network call. Returns "" when the cache is missing,
// Stale (older than maxAge), or malformed. Used by the CLI's
// Per-command upgrade hint which must not block on the network.
func LatestVersionFromCache(maxAge time.Duration) string {
	cachePath := filepath.Join(platform.BaseDir(), versionCheckFile)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return ""
	}
	parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	if len(parts) != 2 {
		return ""
	}
	ts, err := time.Parse(time.RFC3339, parts[0])
	if err != nil || time.Since(ts) >= maxAge {
		return ""
	}
	return parts[1]
}

// CheckLatestVersion returns the latest release version using a file cache.
// MaxAge controls cache staleness (e.g. 1h for web, 24h for CLI).
// Returns "" on any error (network, parse, etc.).
func CheckLatestVersion(maxAge time.Duration) string {
	cacheDir := platform.BaseDir()
	cachePath := filepath.Join(cacheDir, versionCheckFile)

	if cached := LatestVersionFromCache(maxAge); cached != "" {
		return cached
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-fsSL", githubReleasesURL)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	str := string(output)
	idx := strings.Index(str, `"tag_name"`)
	if idx == -1 {
		return ""
	}
	rest := str[idx+len(`"tag_name"`):]
	start := strings.Index(rest, `"`) + 1
	end := strings.Index(rest[start:], `"`)
	if start < 1 || end < 0 {
		return ""
	}
	latest := rest[start : start+end]

	os.MkdirAll(cacheDir, 0755)
	cacheContent := time.Now().UTC().Format(time.RFC3339) + "\n" + latest
	os.WriteFile(cachePath, []byte(cacheContent), 0644)

	return latest
}

// GetVersionInfo returns full version info including update availability.
// maxAge controls cache staleness for the GitHub check.
func GetVersionInfo(maxAge time.Duration) VersionInfo {
	current := strings.TrimPrefix(CLIVersion, "v")

	info := VersionInfo{
		Current: current,
	}

	latest := CheckLatestVersion(maxAge)
	if latest == "" {
		info.Latest = current
		return info
	}

	latestClean := strings.TrimPrefix(latest, "v")
	info.Latest = latestClean
	info.UpdateAvailable = latestClean != current && current != "dev"
	if info.UpdateAvailable {
		info.ReleaseURL = "https://github.com/jterrazz/spwn/releases/tag/" + latest
	}

	return info
}
