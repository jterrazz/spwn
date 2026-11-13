package calls

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/images"
)

func TestEachRefusalOfABuildCarriesItsCode(t *testing.T) {
	for _, c := range []struct {
		err     error
		code    fail.Code
		subject string
	}{
		{&images.UnknownError{Image: "nope"}, Unknown, "nope"},
		{&images.NoLinuxBinaryError{}, NoLinuxBinary, ""},
		{&images.BinaryNotFoundError{Path: "/tmp/station"}, BinaryNotFound, "/tmp/station"},
		{&images.NoRecipeError{Image: "macos"}, NoRecipe, "macos"},
		{&images.NoEngineError{Hypervisor: "hv"}, NoHypervisor, "hv"},
	} {
		var got *fail.Error
		if !errors.As(Worded(c.err), &got) {
			t.Fatalf("%v: no code", c.err)
		}
		if got.Code != c.code || got.Subject != c.subject || !errors.Is(got, c.err) {
			t.Errorf("%v: got %s %q", c.err, got.Code, got.Subject)
		}
	}
}
