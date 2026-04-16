package update

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

// CheckLatestVersion returns the latest release version using a file cache.
// maxAge controls cache staleness (e.g. 1h for web, 24h for CLI).
// Returns "" on any error (network, parse, etc.).
func CheckLatestVersion(maxAge time.Duration) string {
	cacheDir := platform.BaseDir()
	cachePath := filepath.Join(cacheDir, versionCheckFile)

	if data, err := os.ReadFile(cachePath); err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
		if len(parts) == 2 {
			if ts, err := time.Parse(time.RFC3339, parts[0]); err == nil {
				if time.Since(ts) < maxAge {
					return parts[1]
				}
			}
		}
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
