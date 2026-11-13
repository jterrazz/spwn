package accounts

import (
	"regexp"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

var secretShape = regexp.MustCompile(catalog.LeakShapes)

// value: a key fingerprint is an identity, kept whole though it is long.
var fingerprintShape = regexp.MustCompile(`^SHA256:[A-Za-z0-9+/]{43}$`)

func Redact(v string) (kept string, redacted bool) {
	// invariant: a value shaped like a token is never kept: it reads Unknown,
	// and redacted says so.
	if fingerprintShape.MatchString(v) || !secretShape.MatchString(v) {
		return v, false
	}
	return Unknown, true
}
