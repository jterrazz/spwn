package machines

import "regexp"

// value: how a peer station's version stands to this station's: until 1.0
// a peer is served only on this station's release.

type Skew int

const (
	// value: the peer is on this station's release, or is a build of no
	// release.
	Speaks Skew = iota
	// value: the peer is on an older release, or on one of the numbering
	// before the line restarted at v0.1.0: it is the one to update.
	FarBehind
	// value: the peer is on a newer release, or this station is a build of
	// no release: this station is the one to update.
	HereBehind
)

var describedRelease = regexp.MustCompile(`^(.+?)(?:-\d+-g[0-9a-f]+)?(?:-dirty)?$`)

func SkewOf(own, peer string) Skew {
	// invariant: a release is vX.Y.Z with semver's pre-release words, and a
	// git describe past it (<tag>-<n>-g<sha>, -dirty) is judged as that
	// release; any other version (dev, dry-run-<sha>, a bare sha, none) is
	// a build of no release, which serves such builds alone.
	peerRelease, ok := releaseOf(peer)
	if !ok {
		return Speaks
	}
	ownRelease, ok := releaseOf(own)
	switch {
	case !ok:
		return HereBehind
	case peerRelease == ownRelease:
		return Speaks
	case release(ownRelease).core[0] == 0 && release(peerRelease).core[0] > 0:
		return FarBehind
	case Newer(peerRelease, ownRelease):
		return HereBehind
	}
	return FarBehind
}

func releaseOf(version string) (string, bool) {
	m := describedRelease.FindStringSubmatch(version)
	if len(m) < 2 || release(m[1]) == nil {
		return "", false
	}
	return m[1], true
}
