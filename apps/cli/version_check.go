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

var pendingUpgrade string

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

func ranUpgradeCommand() bool {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg == "upgrade"
	}
	return false
}

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

const bannerBoxWidth = 46

func printUpgradeHint() {
	if pendingUpgrade == "" {
		return
	}
	current := strings.TrimPrefix(Version, "v")
	latest := strings.TrimPrefix(pendingUpgrade, "v")
	w := os.Stderr

	const (
		cornerAndDash    = 2
		spaceWidth       = 1
		titleWidth       = 16
		topPrefixVisible = cornerAndDash + spaceWidth + titleWidth + spaceWidth
	)
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
