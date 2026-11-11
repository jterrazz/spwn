package catalog

import (
	"slices"
	"strconv"
	"strings"
)

func (m MethodSpec) Key() string {
	// value: the one manager or setting key a method names (`brew`, `script`,
	// `macos-defaults`); the decoder let through exactly one.
	keys := slices.Concat(m.appKeys(), m.settingKeys())
	if len(keys) != 1 {
		return ""
	}
	return keys[0]
}

// value: the method keys station runs no installer for yet whose manager
// installs a version asked: apt's package version, a release's tag.
var pinning = map[string]bool{"apt": true, "github-release": true}

func (m MethodSpec) Pins() bool {
	// value: whether this method installs the version a Machinefile asks:
	// never under `pin: none`; for a method station runs, whether its
	// install carries a version to its manager (mise's), whatever `pin`
	// claims; else its manager's.
	if m.Pin == PinNone {
		return false
	}
	if in := m.install(nil); in != nil {
		_, ok := Pinned(in, "")
		return ok
	}
	return m.Pin == PinVersion || pinning[m.Key()]
}

func (o Option) Check(v OptionValue) string {
	// value: why a value does not fit this option, empty when it does.
	return o.fits(string(v))
}

// value: a system, as the selector of a method judges it.

type System struct {
	OS      OS
	Version string
	Arch    Arch
	Machine MachineRole
}

func (s Selector) Matches(sys System) bool {
	// invariant: every key of a selector narrows: the os itself, or a family
	// it is in (Ubuntu and Debian are the debian family, both linux); a
	// version range against the system's version, which a system whose
	// version is unknown never meets; the arch and the kind of machine when
	// both are said.
	switch {
	case s.OS != "" && s.OS != OSAny && s.OS != sys.OS:
		return false
	case s.Family != "" && sys.OS != OSUbuntu && sys.OS != OSDebian:
		return false
	case s.Arch != "" && sys.Arch != "" && s.Arch != sys.Arch:
		return false
	case len(s.Machine) > 0 && sys.Machine != "" && !slices.Contains(s.Machine, sys.Machine):
		return false
	case s.Version != "":
		return InRange(sys.Version, s.Version)
	}
	return true
}

func InRange(version, ranges string) bool {
	// invariant: a range is comparisons joined by commas, every one met; a
	// bare version is that version and every one it prefixes (`15` is 15.x).
	if version == "" {
		return false
	}
	for _, part := range strings.Split(ranges, ",") {
		part = strings.TrimSpace(part)
		op := part[:len(part)-len(strings.TrimLeft(part, "<>="))]
		want := strings.TrimPrefix(part, op)
		c := compareVersions(version, want)
		var ok bool
		switch op {
		case ">=":
			ok = c >= 0
		case ">":
			ok = c > 0
		case "<=":
			ok = c <= 0
		case "<":
			ok = c < 0
		default:
			ok = version == want || strings.HasPrefix(version, want+".")
		}
		if !ok {
			return false
		}
	}
	return true
}

func compareVersions(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range max(len(as), len(bs)) {
		x, y := 0, 0
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
