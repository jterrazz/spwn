package calls

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/images"
)

func TestAReleaseTagNeedsNoLocalBinaryWhetherPlainOrPreRelease(t *testing.T) {
	for _, tag := range []string{"v3.1.0", "v0.1.0-alpha.1", "v4.0.0-alpha.1", "v4.0.0-rc.2"} {
		if err := (&Builder{o: BuilderOptions{Version: tag}}).needsBinary(""); err != nil {
			t.Errorf("needsBinary(%q) = %v; want the release's binary fetched", tag, err)
		}
	}
}

func TestADevelopmentVersionNeedsABinaryGiven(t *testing.T) {
	for _, tag := range []string{"dev", "v4.0.0-", "v4.0.0-alpha..1", "v4.0", "4.0.0", "v4.0.0-alpha.1-dirty+x", "v4.0.0-dirty "} {
		var missing *images.NoLinuxBinaryError
		if err := (&Builder{o: BuilderOptions{Version: tag}}).needsBinary(""); !errors.As(err, &missing) {
			t.Errorf("needsBinary(%q) = %v; want NoLinuxBinaryError", tag, err)
		}
	}
}

func TestTheReleaseBinaryComesFromSpwnReleasesAndItsChecksums(t *testing.T) {
	// invariant: the base is the repository releases are published in.
	if releaseBase != "https://github.com/jterrazz/spwn/releases/download/" || releaseChecksums != "checksums.txt" {
		t.Errorf("release base %q, checksums %q", releaseBase, releaseChecksums)
	}
}
