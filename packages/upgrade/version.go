// Package upgrade implements the spwn CLI self-update system.
//
// The update system is GitHub-only: it fetches release metadata via the
// GitHub API, downloads platform-appropriate binaries from release assets,
// verifies them against a SHA256SUMS file also in the release, and
// atomically swaps them into place.
//
// This file contains version parsing and comparison. The CLI uses
// semver-style tags (v1.2.3, v1.2.3-beta.1). Comparison is tolerant of
// a leading "v" and of the special "dev" string used for local builds.
package upgrade

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a parsed semver tag with an optional prerelease label.
// Compare() returns -1/0/1 using semver precedence rules.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string // "beta.1", "nightly.20260405", "" for stable
	Raw        string // original string including any leading "v"
}

// ParseVersion parses a version tag. Accepts a leading "v" prefix.
// Returns an error for strings that don't match the {major}.{minor}.{patch}
// shape. The special string "dev" parses to a zero Version and is
// considered older than any real release.
func ParseVersion(s string) (Version, error) {
	raw := s
	s = strings.TrimPrefix(s, "v")
	if s == "dev" || s == "" {
		return Version{Raw: raw}, nil
	}

	// Split off prerelease: {core}-{prerelease}
	core := s
	pre := ""
	if idx := strings.Index(s, "-"); idx >= 0 {
		core = s[:idx]
		pre = s[idx+1:]
	}

	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("not a semver-shaped tag: %q", raw)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major in %q: %w", raw, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor in %q: %w", raw, err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch in %q: %w", raw, err)
	}

	return Version{Major: major, Minor: minor, Patch: patch, Prerelease: pre, Raw: raw}, nil
}

// IsDev reports whether this is the "dev" placeholder used for local builds.
// Dev builds are considered older than every real release so `spwn upgrade`
// always reports an update available on them.
func (v Version) IsDev() bool {
	return strings.TrimPrefix(v.Raw, "v") == "dev" || v.Raw == ""
}

// IsPrerelease returns true when the version carries a prerelease label
// (anything with a hyphen: "beta.1", "rc.0", "nightly.20260405").
func (v Version) IsPrerelease() bool {
	return v.Prerelease != ""
}

// Compare returns -1, 0, or 1 depending on whether v is older, equal to, or
// newer than other (semver precedence). Dev builds are strictly older than
// any real release.
func (v Version) Compare(other Version) int {
	if v.IsDev() && !other.IsDev() {
		return -1
	}
	if !v.IsDev() && other.IsDev() {
		return 1
	}
	if v.IsDev() && other.IsDev() {
		return 0
	}

	if c := cmpInt(v.Major, other.Major); c != 0 {
		return c
	}
	if c := cmpInt(v.Minor, other.Minor); c != 0 {
		return c
	}
	if c := cmpInt(v.Patch, other.Patch); c != 0 {
		return c
	}

	// Per semver: a version WITH a prerelease is older than one WITHOUT.
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	return strings.Compare(v.Prerelease, other.Prerelease)
}

// String returns a canonical "vMAJOR.MINOR.PATCH[-PRE]" rendering.
func (v Version) String() string {
	if v.IsDev() {
		return "dev"
	}
	base := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		base += "-" + v.Prerelease
	}
	return base
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
