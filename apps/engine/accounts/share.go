package accounts

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const (
	DefaultShareTerm = 12 * time.Hour
	ShortestShare    = time.Minute
	LongestShare     = 7 * 24 * time.Hour
)

func DefaultShareList() []string {
	// value: the share list a Mac keeps when its preferences name none:
	// every session secret and the Claude token (decision A4).
	return []string{"session:*", catalog.ClaudeTokenName}
}

type Share struct {
	Name  string    `json:"name"`
	At    time.Time `json:"at"`
	Until time.Time `json:"until"`
}

func Expired(s []Share, now time.Time) []Share {
	// invariant: a share ends at its until: from that instant on it is
	// expired, never a moment later.
	var out []Share
	for _, sh := range s {
		if !now.Before(sh.Until) {
			out = append(out, sh)
		}
	}
	return out
}

func Shareable(name string, shareList []string) bool {
	// invariant: a name is shareable when the list holds it, or holds its
	// family as `<prefix>:*`; nothing else is ever copied into a VM.
	for _, entry := range shareList {
		if entry == name {
			return true
		}
		if prefix, ok := strings.CutSuffix(entry, "*"); ok && prefix != "" && strings.HasPrefix(name, prefix) && len(name) > len(prefix) {
			return true
		}
	}
	return false
}

func SecretOf(account string) (string, bool) {
	// value: the secret whose copy carries the account's right into a VM;
	// an account with none is shared with no VM.
	i := slices.IndexFunc(catalog.ShareScopes, func(sc catalog.ShareScope) bool { return sc.Provider == account })
	if i < 0 {
		return "", false
	}
	return catalog.ShareScopes[i].Name, true
}

func AccountOf(secret string) string {
	for _, sc := range catalog.ShareScopes {
		if sc.Name == secret {
			return sc.Provider
		}
	}
	return secret
}

func Label(account string) string {
	for _, p := range catalog.Providers {
		if p.ID == account {
			return p.Label
		}
	}
	return account
}

func Known(account string) bool {
	return slices.ContainsFunc(catalog.Providers, func(p catalog.Provider) bool { return p.ID == account && p.Mac })
}

func CheckTerm(term time.Duration) (time.Duration, bool) {
	// invariant: a share lasts 1 min to 7 d, and a term of 0 is the default:
	// every surface and the engine judge a term by this one rule.
	if term == 0 {
		return DefaultShareTerm, true
	}
	return term, term >= ShortestShare && term <= LongestShare
}

func ParseTerm(text string) (time.Duration, bool) {
	// value: how long a share lasts, Go-style with a unit, days as `7d`;
	// false when the text is no duration, whatever its span.
	term, err := time.ParseDuration(text)
	if days, ok := strings.CutSuffix(text, "d"); ok {
		n, convErr := strconv.Atoi(days)
		term, err = time.Duration(n)*24*time.Hour, convErr
	}
	return term, err == nil
}
