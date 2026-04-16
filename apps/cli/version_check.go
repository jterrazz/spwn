package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/upgrade"
)

const versionCheckInterval = 24 * time.Hour

// pendingUpgrade is set by the background version check goroutine.
// If non-empty after command execution, a yellow hint is printed.
var pendingUpgrade string

// startVersionCheck launches a background goroutine that checks for a new
// spwn release. It reads a cached result first and only hits the network
// if the cache is older than 24 hours. The result is stored in pendingUpgrade.
func startVersionCheck() {
	// Skip for dev builds or if SPWN_NO_UPDATE_CHECK is set
	if Version == "dev" {
		return
	}
	if os.Getenv("SPWN_NO_UPDATE_CHECK") != "" {
		return
	}

	go func() {
		latest := upgrade.CheckLatestVersion(versionCheckInterval)
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
