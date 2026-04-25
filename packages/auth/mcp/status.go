package mcp

import "os"

// IsAuthenticated reports whether a tokens.json exists for p. The
// presence of the file is the spwn-side signal: an actual access
// token's freshness is the responsibility of Refresh (called from
// auth.SyncCredentials on every world spawn), so callers don't need
// to think about expiry here.
func IsAuthenticated(p Provider) bool {
	_, err := os.Stat(ProviderTokenPath(p))
	return err == nil
}
