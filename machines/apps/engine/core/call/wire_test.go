package call_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
)

func TestAFixLeavesWithItsTokenWithheld(t *testing.T) {
	token := "s" + "k-" + strings.Repeat("Ab3dE", 8)
	path := "specs/usecases/machinefile/apply/a-failing-line.spec.yaml"
	word := func(error) *fail.Error {
		return &fail.Error{Message: "refused", Fix: finding.Fix{
			Call:    "settings.apply",
			Command: "station login --token " + token + " --into " + path,
			Path:    "/Users/me/.config?token=" + token,
			Args:    map[string]string{"token": token, "into": path},
		}}
	}
	w := call.ToWire(errors.New("refused"), word)
	if w.Fix == nil {
		t.Fatal("the wire error dropped its fix")
	}
	if strings.Contains(w.Fix.Command, token) || strings.Contains(w.Fix.Path, token) || w.Fix.Args["token"] != "withheld" {
		t.Errorf("a fix left with its token: %q, %q, %v", w.Fix.Command, w.Fix.Path, w.Fix.Args)
	}
	if !strings.HasSuffix(w.Fix.Command, " --into "+path) || w.Fix.Call != "settings.apply" || w.Fix.Args["into"] != path {
		t.Errorf("the fix's other words were not kept: %+v", *w.Fix)
	}
}
