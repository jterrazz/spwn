package gh

import (
	"os"
	"strings"
)

// IsAuthenticated reports whether spwn has a usable gh hosts.yml.
// "Usable" means: the file exists AND contains an inline
// oauth_token (not a "keyring" sentinel that we couldn't read
// from inside a container anyway).
func IsAuthenticated() bool {
	b, err := os.ReadFile(HostsPath())
	if err != nil {
		return false
	}
	// hosts.yml stored via gh's keyring path is functionally
	// useless to us — the container can't reach the host keychain.
	// Require an explicit oauth_token: line.
	return strings.Contains(string(b), "oauth_token:")
}
