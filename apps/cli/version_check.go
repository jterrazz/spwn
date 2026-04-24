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

// startVersionCheck seeds pendingUpgrade for the current command and
// Schedules a background refresh when the cache has gone stale.
//
// Cache hit path (the common case after the first command of any day):
// Synchronous read, sub-millisecond, sets pendingUpgrade before the
// Command body runs. The PostRun hook always finds it.
//
// Cache miss / stale path: we fire off a goroutine that hits the
// Network and refreshes the cache for the NEXT invocation. We
// Deliberately don't try to read its result on this invocation —
// Fast commands (`spwn ls`, `spwn status`) would finish before the
// Goroutine, and the pre-fix behaviour was racey exactly because of
// That. The user sees the banner one command later; that's the cost
// Of not blocking on curl.
func startVersionCheck() {
	if Version == "dev" {
		return
	}
	if os.Getenv("SPWN_NO_UPDATE_CHECK") != "" {
		return
	}

	if cached := upgrade.LatestVersionFromCache(versionCheckInterval); cached != "" {
		applyLatestVersion(cached)
		return
	}

	go func() {
		latest := upgrade.CheckLatestVersion(versionCheckInterval)
		applyLatestVersion(latest)
	}()
}

// applyLatestVersion compares a fetched/cached "latest" string
// Against the current build version and records a pending upgrade
// When they differ. Centralised so the sync and async paths agree
// On the comparison rule (both strip a leading "v").
func applyLatestVersion(latest string) {
	if latest == "" {
		return
	}
	current := strings.TrimPrefix(Version, "v")
	latest = strings.TrimPrefix(latest, "v")
	if latest != current {
		pendingUpgrade = latest
	}
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
