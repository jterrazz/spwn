package setup

import (
	"regexp"
	"strings"
)

var machineName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

func ValidName(name string) bool { return machineName.MatchString(name) }

func DefaultName(localName, hostname string) string {
	raw := strings.TrimSpace(localName)
	if raw == "" {
		raw = hostname
	}
	label, _, _ := strings.Cut(strings.ToLower(raw), ".")
	label = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, label)
	label = strings.Trim(label, "-")
	if len(label) > 40 {
		label = strings.TrimRight(label[:40], "-")
	}
	return label
}
