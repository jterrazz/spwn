package accounts

import "github.com/jterrazz/studio-station/apps/engine/core/secret"

func Redact(v string) (kept string, redacted bool) {
	// invariant: a value shaped like a secret (core/secret) is never kept: it
	// reads Unknown, and redacted says so.
	if !secret.Shaped(v) {
		return v, false
	}
	return Unknown, true
}
