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
	return strings.Contains(string(b), "oauth_token:")
}
