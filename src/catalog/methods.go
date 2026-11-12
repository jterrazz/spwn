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

func (m MethodSpec) Holds() Hold {
	// value: what this method holds of a range a Machinefile asks: only its
	// newest under `pin: none`; else what its install carries to its
	// manager (HoldOf), whatever `pin` claims.
	if m.Pin == PinNone {
		return HoldLatest
	}
	return HoldOf(m.install(nil))
}

func (m MethodSpec) Fits(r Range) bool {
	// value: whether this method installs within r: what it holds (Holds),
	// or the one major a versioned formula of brew's names (Fits).
	if m.Pin == PinNone {
		return HoldLatest.Holds(r)
	}
	return Fits(m.install(nil), r)
}

func (m MethodSpec) HoldWords() string {
	// value: what this method holds, as a refusal says it: `only its newest
	// of 18 (postgresql@18)` for a versioned formula of brew's.
	if b, ok := m.install(nil).(BrewInstall); ok && m.Pin != PinNone {
		if _, major, versioned := strings.Cut(lastSegment(b.Formula), "@"); versioned {
			return "only its newest of " + major + " (" + lastSegment(b.Formula) + ")"
		}
	}
	return m.Holds().String()
}

func (m MethodSpec) Pins() bool {
	// value: whether this method installs one version it is handed, as a
	// lock hands it one.
	return m.Holds() >= HoldExact
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
