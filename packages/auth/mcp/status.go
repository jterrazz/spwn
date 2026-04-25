package mcp

import "os"

// IsAuthenticated reports whether a tokens.json exists for p. The
// presence of the file is the spwn-side signal — mcp2cli refreshes
// it transparently on each call, so any stale-token concerns live
// inside mcp2cli, not here.
func IsAuthenticated(p Provider) bool {
	_, err := os.Stat(ProviderTokenPath(p))
	return err == nil
}
