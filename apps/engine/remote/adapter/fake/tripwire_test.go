package fake_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract/contracttest"
)

type said struct{ Line string }

func TestAnAnswerCrossingBackWithASecretTripsTheTripwire(t *testing.T) {
	token := "s" + "k-" + strings.Repeat("Ab3dE", 8)
	var tripped []string
	secret.Arm(nil, func(s string) { tripped = append(tripped, s) })
	t.Cleanup(secret.Disarm)
	far, err := call.NewRegistry([]call.Handler{call.Bind(call.Call[struct{}, struct{}, said]{
		Name: "machinefile.plan", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (said, error) {
			return said{Line: "export API_KEY=" + token}, nil
		},
	}, struct{}{})})
	if err != nil {
		t.Fatal(err)
	}
	d := fake.New()
	d.Serve(contracttest.Mini.Ref.Name, fake.Mac{Registry: far, Profile: contracttest.Mini, Version: contracttest.Version})
	ep, err := d.Dial(t.Context(), contracttest.Mini.Ref.Name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ep.Close() })
	if _, err := ep.Invoke(t.Context(), contracttest.Mini.Ref, "machinefile.plan", json.RawMessage(`{}`), nil); err != nil {
		t.Fatal(err)
	}
	if len(tripped) != 1 || strings.Contains(tripped[0], token) {
		t.Errorf("the far answer tripped %q, want one trip that says no secret", tripped)
	}
}
