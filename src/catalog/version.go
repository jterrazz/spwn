package catalog

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// value: how far a version a Machinefile asks lets an app move: every
// version (nothing, `latest`), the minors and patches of a major (`24`,
// `^24.2.0`), the patches of a minor (`24.11`, `~24.11.0`), or one version
// (`24.11.3`, and a lock's version, verbatim).

type Range struct {
	form  rangeForm
	low   [3]int
	high  [3]int
	exact string
	text  string
}

type rangeForm int

const (
	RangeAny rangeForm = iota
	RangeCaret
	RangeTilde
	RangeExact
)

// value: what a method can hold of a range: only the newest (brew, a
// cask, a script), one exact version as well (apt's `package=<version>`,
// a release's tag), or any range (mise's prefix, bun's specifier). npm
// and uv install through brew's bundle, whose lines carry no version.

type Hold int

const (
	HoldLatest Hold = iota
	HoldExact
	HoldAny
)

func (h Hold) String() string {
	switch h {
	case HoldExact:
		return "an exact version"
	case HoldAny:
		return "any range"
	case HoldLatest:
	}
	return "only its newest"
}

var (
	rangeWord   = regexp.MustCompile(`^([\^~]?)v?(\d+)(?:\.(\d+))?(?:\.(\d+))?$`)
	exactWord   = regexp.MustCompile(`^v?\d+\.\d+\.\d+[-+][0-9A-Za-z.+-]+$`)
	versionHead = regexp.MustCompile(`^(?:[A-Za-z][A-Za-z0-9]*-)*v?(\d+)(?:\.(\d+))?(?:\.(\d+))?`)
	vendorHead  = regexp.MustCompile(`^((?:[A-Za-z][A-Za-z0-9]*-)+)v?\d`)
)

const RangeForms = "latest, 24, 24.11, 24.11.3, ^24.2.0 or ~24.11.0" // value: every form a version takes, as a refusal names them.

func ParseRange(text string) (Range, error) {
	// invariant: npm's semver forms and three bare ones: a major alone is
	// ^major, a minor alone ~major.minor, three numbers exact; `=` before
	// a word that names a number is that word exactly, as a lock hands it
	// on (`=1:2.43.0-1ubuntu7`, `=temurin-21.0.13+9.0.LTS`).
	text = strings.TrimSpace(text)
	switch {
	case text == "" || text == "latest":
		return Range{text: text}, nil
	case strings.HasPrefix(text, "=") && strings.ContainsAny(text, "0123456789") && !strings.ContainsAny(text, " \t"):
		return Exactly(text[1:]), nil
	case exactWord.MatchString(text):
		return Exactly(text), nil
	}
	m := rangeWord.FindStringSubmatch(text)
	if m == nil {
		return Range{}, fmt.Errorf("version %q is no version: %s", text, RangeForms)
	}
	parts := 1
	var v [3]int
	for i, s := range m[2:] {
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return Range{}, fmt.Errorf("version %q is no version: %s", text, RangeForms)
		}
		v[i], parts = n, i+1
	}
	r := Range{low: v, text: text}
	switch {
	case m[1] == "" && parts == 3:
		return Exactly(text), nil
	case m[1] == "^" || (m[1] == "" && parts == 1):
		r.form, r.high = RangeCaret, caretHigh(v, parts)
	default:
		r.form, r.high = RangeTilde, tildeHigh(v, parts)
	}
	return r, nil
}

func Exactly(version string) Range {
	// value: one version, verbatim, as a lock or an exact entry writes it.
	return Range{form: RangeExact, exact: version, text: version}
}

func caretHigh(v [3]int, parts int) [3]int {
	// invariant: npm's caret: the first number written that is not zero
	// is held, `^0.2.3` holding 0.2.x and `^0.0.3` 0.0.3 alone.
	switch {
	case v[0] > 0 || parts == 1:
		return [3]int{v[0] + 1, 0, 0}
	case v[1] > 0 || parts == 2:
		return [3]int{0, v[1] + 1, 0}
	}
	return [3]int{0, 0, v[2] + 1}
}

func tildeHigh(v [3]int, parts int) [3]int {
	if parts == 1 {
		return [3]int{v[0] + 1, 0, 0}
	}
	return [3]int{v[0], v[1] + 1, 0}
}

func (r Range) Any() bool   { return r.form == RangeAny }
func (r Range) Exact() bool { return r.form == RangeExact }

func (r Range) String() string {
	// value: the range as its file writes it, `latest` for every version.
	if r.Any() {
		return "latest"
	}
	return r.text
}

func (r Range) Spec() string {
	// value: the range in one unambiguous word, which ParseRange reads back
	// as the same range: an exact version after `=`, else npm's form; empty
	// for every version.
	switch r.form {
	case RangeExact:
		return "=" + r.exact
	case RangeAny:
		return ""
	case RangeCaret, RangeTilde:
	}
	return r.Npm()
}

func (r Range) Npm() string {
	// value: the range as npm and bun write it: `^24.2.0`, `~24.11.0`, an
	// exact version; empty for every version.
	switch r.form {
	case RangeCaret:
		return "^" + dotted(r.low)
	case RangeTilde:
		return "~" + dotted(r.low)
	case RangeExact:
		return strings.TrimPrefix(r.exact, "v")
	case RangeAny:
	}
	return ""
}

func (r Range) Head() string {
	// value: the prefix whose newest version is the newest the range holds,
	// as mise resolves `node@24`: the numbers the upper bound keeps, the
	// version itself when exact; empty for every version.
	switch r.form {
	case RangeExact:
		return r.exact
	case RangeAny:
		return ""
	case RangeCaret, RangeTilde:
	}
	switch {
	case r.high[0] != r.low[0]:
		return strconv.Itoa(r.low[0])
	case r.high[1] != r.low[1]:
		return strconv.Itoa(r.low[0]) + "." + strconv.Itoa(r.low[1])
	}
	return dotted(r.low)
}

func (r Range) Contains(version string) bool {
	// invariant: every version is in latest; an exact range holds its one
	// version (a `v` before it aside); any other holds the released
	// versions from its lower bound up to, not including, its upper one,
	// never a prerelease (`25.0.0-rc.1`) nor a version that names no number.
	switch r.form {
	case RangeAny:
		return true
	case RangeExact:
		return exactly(r.exact, version)
	case RangeCaret, RangeTilde:
	}
	v, ok := numbers(version)
	return ok && !prerelease(version) && compare(v, r.low) >= 0 && compare(v, r.high) < 0
}

func exactly(exact, version string) bool {
	// invariant: a version is the one an exact range names, a `v` before
	// either aside; a vendor before its numbers is read through when the
	// range names none (`9.8.7` holds temurin-9.8.7, never zulu's when it
	// names temurin's).
	exact, version = strings.TrimPrefix(exact, "v"), strings.TrimPrefix(version, "v")
	if version == exact || vendorHead.MatchString(exact) {
		return version == exact
	}
	if m := vendorHead.FindStringSubmatch(version); m != nil {
		return strings.TrimPrefix(version[len(m[1]):], "v") == exact
	}
	return false
}

func (r Range) Past(version string) bool {
	// value: whether a released version is newer than all the range holds:
	// a major past `24`, a minor past `24.11`, any newer one past an exact
	// range; a prerelease is never past, so never waits.
	if prerelease(version) {
		return false
	}
	switch r.form {
	case RangeAny:
		return false
	case RangeExact:
		v, ok := numbers(version)
		at, known := numbers(r.exact)
		return ok && known && compare(v, at) > 0
	case RangeCaret, RangeTilde:
	}
	v, ok := numbers(version)
	return ok && compare(v, r.high) >= 0
}

func (r Range) Moved(version string) string {
	// value: the range written as its file writes it, moved to hold a newer
	// version: `24` to `25`, `~24.11.0` to `~25.1.0`, an exact one to that
	// version.
	v, ok := numbers(version)
	if !ok || r.Any() {
		return r.String()
	}
	if r.Exact() {
		return strings.TrimPrefix(version, "v")
	}
	m := rangeWord.FindStringSubmatch(r.text)
	parts := 1
	for i, s := range m[2:] {
		if s != "" {
			parts = i + 1
		}
	}
	words := []string{strconv.Itoa(v[0]), strconv.Itoa(v[1]), strconv.Itoa(v[2])}
	return m[1] + strings.Join(words[:parts], ".")
}

func Newest(versions ...string) string {
	// value: the newest of versions by their numbers, a release before a
	// prerelease of the same numbers, else the first of a tie; empty when
	// none names a number.
	best, at := "", [3]int{}
	for _, s := range versions {
		v, ok := numbers(s)
		if !ok {
			continue
		}
		if c := compare(v, at); best == "" || c > 0 || (c == 0 && prerelease(best) && !prerelease(s)) {
			best, at = s, v
		}
	}
	return best
}

func prerelease(version string) bool {
	// value: whether a version is a prerelease of its numbers, as semver
	// writes one after a dash (`25.0.0-rc.1`); build words after a plus
	// (`21.0.13+9`) are none.
	head := versionHead.FindString(strings.TrimSpace(version))
	return head != "" && strings.HasPrefix(strings.TrimSpace(version)[len(head):], "-")
}

func numbers(version string) ([3]int, bool) {
	m := versionHead.FindStringSubmatch(strings.TrimSpace(version))
	if m == nil {
		return [3]int{}, false
	}
	var v [3]int
	for i, s := range m[1:] {
		if s != "" {
			v[i], _ = strconv.Atoi(s)
		}
	}
	return v, true
}

func compare(a, b [3]int) int {
	for i := range a {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func dotted(v [3]int) string {
	return strconv.Itoa(v[0]) + "." + strconv.Itoa(v[1]) + "." + strconv.Itoa(v[2])
}

func (r *Range) UnmarshalYAML(n *yaml.Node) error {
	// invariant: a version is read once, into its range, and a word that is
	// none of its forms is refused at its line.
	if n.Kind != yaml.ScalarNode {
		return &yaml.TypeError{Errors: []string{fmt.Sprintf("line %d: a version is one word: %s", n.Line, RangeForms)}}
	}
	parsed, err := ParseRange(n.Value)
	if err != nil {
		return &yaml.TypeError{Errors: []string{fmt.Sprintf("line %d: %s", n.Line, err)}}
	}
	*r = parsed
	return nil
}

func (r Range) MarshalYAML() (any, error) { return r.text, nil }

func (r Range) IsZero() bool { return r.form == RangeAny && r.text == "" }

func HoldOf(in Install) Hold {
	// value: what an install carries of a range to its manager: a runtime of
	// mise's by its prefix and a package of bun's by its specifier hold any
	// range, apt's package and a release's tag one
	// version; every other carries none.
	switch in.(type) {
	case MiseInstall, BunInstall:
		return HoldAny
	case AptInstall, GitHubReleaseInstall:
		return HoldExact
	}
	return HoldLatest
}

func Fits(in Install, r Range) bool {
	// value: whether an install installs within r: its method holds r
	// (HoldOf), or it is a versioned formula of brew's (`postgresql@18`) and
	// r is the one major its name holds (`18`, `^18.0.0`), whose newest an
	// update then keeps.
	if HoldOf(in).Holds(r) {
		return true
	}
	b, ok := in.(BrewInstall)
	if !ok {
		return false
	}
	_, at, versioned := strings.Cut(lastSegment(b.Formula), "@")
	major, whole := r.Major()
	return versioned && whole && at == strconv.Itoa(major)
}

func (r Range) Major() (int, bool) {
	// value: the one major r holds whole, every minor and patch of it
	// (`18`, `^18.0.0`, `~18`); false for any other range.
	if r.form != RangeCaret && r.form != RangeTilde {
		return 0, false
	}
	return r.low[0], r.low[1] == 0 && r.low[2] == 0 && r.high == [3]int{r.low[0] + 1, 0, 0}
}

func (h Hold) Holds(r Range) bool {
	// value: whether a method holding h installs within r: every method
	// within latest, an exact one by a method that pins one version.
	switch {
	case r.Any():
		return true
	case r.Exact():
		return h >= HoldExact
	}
	return h == HoldAny
}
