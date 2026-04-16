package main

import (
	"errors"
	"os"

	"spwn.sh/apps/cli"
	"spwn.sh/packages/paths"
)

func main() {
	// Make docker resolvable when spwn is launched by a GUI (Tauri /
	// Finder / Spotlight) on macOS, where the inherited PATH does not
	// include /opt/homebrew/bin or /usr/local/bin by default. No-op
	// when PATH already contains the target locations, so CLI usage
	// from a terminal is unaffected.
	paths.EnsureDockerFriendlyPATH()

	if err := cli.Execute(); err != nil {
		// Allow commands to signal a non-default exit code via the
		// ExitCoder interface (e.g. exit 2 for "not yet implemented"
		// so scripts can distinguish a missing feature from a failure).
		var coder cli.ExitCoder
		if errors.As(err, &coder) {
			os.Exit(coder.ExitCode())
		}
		os.Exit(1)
	}
}
