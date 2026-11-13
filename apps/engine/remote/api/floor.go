package api

import (
	"regexp"
	"slices"
	"strconv"
)

const (
	// value: the first release a station speaks to, the one whose wire says
	// package where v3.1.0 said item (ADR-047); a peer of an older release
	// is refused, never translated.
	Floor = "v3.2.0"
)

var releaseShape = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

func BeforeFloor(version string) bool {
	// invariant: only a release, vX.Y.Z as the release workflow stamps it,
	// is judged; a build of no release (dev, a git describe past a tag, no
	// version said) passes.
	at, ok := release(version)
	if !ok {
		return false
	}
	floor, _ := release(Floor)
	return slices.Compare(at, floor) < 0
}

func release(version string) ([]int, bool) {
	m := releaseShape.FindStringSubmatch(version)
	if m == nil {
		return nil, false
	}
	at := make([]int, 0, 3)
	for _, part := range m[1:] {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		at = append(at, n)
	}
	return at, true
}
