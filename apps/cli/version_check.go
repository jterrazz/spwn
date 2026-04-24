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

// ranUpgradeCommand reports whether the user's invocation resolves to
// The top-level `spwn upgrade` command. Used by Execute() to suppress
// The update banner after a user has already run upgrade — stacking
// The banner below the upgrade command's own output would be redundant
// Noise. Heuristic: the first non-flag argument is "upgrade".
func ranUpgradeCommand() bool {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg == "upgrade"
	}
	return false
}

// applyLatestVersion compares a fetched/cached "latest" string
// Against the current build version and records a pending upgrade
// Only when latest is strictly NEWER (semver >). This matters because
// The cache may sit at a stale value below the current build — for
// Example after the user runs `spwn upgrade --force` inside the
// 24-hour cache window — and the old "latest != current" check would
// Then render an "upgrade to vOLD" banner. Semver precedence fixes
// Both directions: downgrades (current > latest) never prompt, and
// Equality (already current) never prompts.
//
// Unparseable inputs degrade to "no banner" rather than raising —
// This is a best-effort UX nudge, not a correctness-critical path.
func applyLatestVersion(latest string) {
	if latest == "" {
		return
	}
	latestV, err := upgrade.ParseVersion(latest)
	if err != nil {
		return
	}
	currentV, err := upgrade.ParseVersion(Version)
	if err != nil {
		return
	}
	if latestV.Compare(currentV) > 0 {
		pendingUpgrade = strings.TrimPrefix(latest, "v")
	}
}

// bannerBoxWidth is the target visible width of the upgrade banner's
// Dash lines, measured in terminal columns. 46 fits comfortably on an
// 80-col terminal, leaves breathing room around the content, and keeps
// The two dash sequences visually balanced.
const bannerBoxWidth = 46

// printUpgradeHint renders a boxed "Update available" callout on
// Stderr when a newer release is pending. Shape:
//
//	┌─ Update available ──────────────────────
//	│  spwn 0.17.1 is out  (you're on 0.0.1)
//	│  Run  spwn upgrade
//	└──────────────────────────────────────────
//
// The label is yellow-bold, the new version is green-bold, the current
// Version is faint, and `spwn upgrade` is cyan-bold. Borders render
// Faint grey so the callout is visible without screaming.
func printUpgradeHint() {
	if pendingUpgrade == "" {
		return
	}
	current := strings.TrimPrefix(Version, "v")
	latest := strings.TrimPrefix(pendingUpgrade, "v")
	w := os.Stderr

	// Top-border layout: "┌─ Update available " (20 visible chars)
	// Plus dashes to fill bannerBoxWidth. Using a constant label
	// Keeps the math trivial and the eye aligned across invocations.
	const topPrefixVisible = 2 /* ┌─ */ + 1 /* sp */ + 16 /* Update available */ + 1 /* sp */
	topDashes := bannerBoxWidth - topPrefixVisible
	if topDashes < 1 {
		topDashes = 1
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s %s %s\n",
		ui.Faint("┌─"),
		ui.Yellow(ui.Strong("Update available")),
		ui.Faint(strings.Repeat("─", topDashes)),
	)
	fmt.Fprintf(w, "  %s  spwn %s is out  %s\n",
		ui.Faint("│"),
		ui.Green(ui.Strong(latest)),
		ui.Faint(fmt.Sprintf("(you're on %s)", current)),
	)
	fmt.Fprintf(w, "  %s  Run  %s\n",
		ui.Faint("│"),
		ui.Cyan(ui.Strong("spwn upgrade")),
	)
	fmt.Fprintf(w, "  %s%s\n",
		ui.Faint("└"),
		ui.Faint(strings.Repeat("─", bannerBoxWidth-1)),
	)
	fmt.Fprintln(w)
}
