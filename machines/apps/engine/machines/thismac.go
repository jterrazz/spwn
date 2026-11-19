package machines

import "strings"

func ThisMacName(preference, hostname string) string {
	// invariant: this Mac has one name on every call: the one it was
	// registered under, else its hostname lowercased.
	if preference != "" {
		return preference
	}
	return strings.ToLower(hostname)
}
