package world

import (
	"errors"
	"testing"
)

func TestSpawnHintNamesTheWorldImageRequirementOverTheImageNotFoundHint(t *testing.T) {
	err := errors.New(`the world image needs a spwn user and passwordless sudo on a Linux host (exit 127: exec: "sudo": executable file not found in $PATH)`)

	if got, want := spawnHint(err, "neo", nil), "Give the image a spwn user and passwordless sudo, or derive it from spwn-world:latest"; got != want {
		t.Errorf("hint:\n got  %q\n want %q", got, want)
	}
}
