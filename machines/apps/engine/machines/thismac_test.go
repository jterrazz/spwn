package machines_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines"
)

func TestThisMacName(t *testing.T) {
	read := 0
	defaultName := func() string { read++; return "mac-mini" }
	if got := machines.ThisMacName("studio-laptop", defaultName); got != "studio-laptop" || read != 0 {
		t.Errorf("a registered Mac: %q, its default read %d times; want studio-laptop, never read", got, read)
	}
	if got := machines.ThisMacName("", defaultName); got != "mac-mini" {
		t.Errorf("a Mac with no name set: %q, want its default name", got)
	}
}
