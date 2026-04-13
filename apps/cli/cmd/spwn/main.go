package main

import (
	"os"

	"spwn.sh/apps/cli"
	"spwn.sh/packages/foundation"
)

func main() {
	// Make docker resolvable when spwn is launched by a GUI (Tauri /
	// Finder / Spotlight) on macOS, where the inherited PATH does not
	// include /opt/homebrew/bin or /usr/local/bin by default. No-op
	// when PATH already contains the target locations, so CLI usage
	// from a terminal is unaffected.
	foundation.EnsureDockerFriendlyPATH()

	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
