package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/foundation"
)

const versionCheckInterval = 24 * time.Hour
const versionCheckFile = ".version-check"

// pendingUpgrade is set by the background version check goroutine.
// If non-empty after command execution, a yellow hint is printed.
var pendingUpgrade string

// startVersionCheck launches a background goroutine that checks for a new
// spwn release. It reads a cached result first and only hits the network
// if the cache is older than 24 hours. The result is stored in pendingUpgrade.
func startVersionCheck() {
	// Skip for dev builds, quiet/json mode, or if SPWN_NO_UPDATE_CHECK is set
	if Version == "dev" || quiet || jsonOutput {
		return
	}
	if os.Getenv("SPWN_NO_UPDATE_CHECK") != "" {
		return
	}

	go func() {
		latest := checkVersionCached()
		if latest == "" {
			return
		}

		current := strings.TrimPrefix(Version, "v")
		latest = strings.TrimPrefix(latest, "v")

		if latest != current {
			pendingUpgrade = latest
		}
	}()
}

// printUpgradeHint prints a yellow message if a newer version was detected.
// Called after the command finishes.
func printUpgradeHint() {
	if pendingUpgrade == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "\n  %s %s\n  %s\n\n",
		ui.Yellow("!"),
		ui.Yellow(fmt.Sprintf("spwn %s is available (current: %s)", pendingUpgrade, strings.TrimPrefix(Version, "v"))),
		ui.Faint("Run \"spwn upgrade\" to update"),
	)
}

// checkVersionCached returns the latest version string, using a 24h file cache
// at ~/.spwn/.version-check. Returns "" on any error (network, parse, etc.).
func checkVersionCached() string {
	cacheDir := foundation.BaseDir()
	cachePath := filepath.Join(cacheDir, versionCheckFile)

	// Read cache
	if data, err := os.ReadFile(cachePath); err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
		if len(parts) == 2 {
			if ts, err := time.Parse(time.RFC3339, parts[0]); err == nil {
				if time.Since(ts) < versionCheckInterval {
					return parts[1] // cache is fresh
				}
			}
		}
	}

	// Fetch from GitHub with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-fsSL", "https://api.github.com/repos/jterrazz/spwn/releases/latest")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse tag_name (same logic as upgrade.go)
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

	// Write cache
	os.MkdirAll(cacheDir, 0755)
	cacheContent := time.Now().UTC().Format(time.RFC3339) + "\n" + latest
	os.WriteFile(cachePath, []byte(cacheContent), 0644)

	return latest
}
