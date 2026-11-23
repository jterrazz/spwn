package calls

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

func TestAnEffectNobodyHandlesIsAnErrorNotASilentSuccess(t *testing.T) {
	svc := NewCatalogue(nil, CatalogueOptions{ThisMachine: catalog.Mac})
	if _, err := svc.handler(catalog.Setting{Name: "x", Effect: catalog.Procedure("nope")}); err == nil || err.Error() != "this station does not know nope: update station" {
		t.Fatalf("an unknown procedure dispatched with %v, want update station", err)
	}
	if _, err := svc.handler(catalog.Setting{Name: "y"}); err == nil {
		t.Fatal("a package with no effect dispatched")
	}
}

type appliedStub struct{ ran *int }

func (appliedStub) Kind() catalog.EffectKind { return catalog.ProcAutologin.Kind() }
func (appliedStub) Observe(context.Context, catalog.Setting) (settings.Observation, error) {
	return settings.Observation{}, nil
}
func (a appliedStub) Apply(context.Context, catalog.Setting, map[string]string) (settings.Change, error) {
	*a.ran++
	return settings.Change{}, nil
}
func (appliedStub) Undo(context.Context, catalog.Setting) (settings.Change, error) {
	return settings.Change{}, nil
}

func TestAnInputWithNoDefaultRefusesTheApplyThatLacksItAndNamesIt(t *testing.T) {
	ran := 0
	item := catalog.Setting{
		Name: "login", Kinds: []catalog.MachineKind{catalog.Mac}, Effect: catalog.ProcAutologin,
		Inputs: []catalog.Input{{Name: "user", Required: true}},
	}
	svc := NewCatalogue([]catalog.Setting{item}, CatalogueOptions{ThisMachine: catalog.Mac, Appliers: []settingscontract.Applier{appliedStub{ran: &ran}}})

	_, err := svc.Install(context.Background(), "login", map[string]string{"password": "p"}, true)

	var said *fail.Said
	if !errors.As(err, &said) || said.Message != "login needs user" || !strings.Contains(said.Hint, "user") || ran != 0 {
		t.Fatalf("an apply without the input: err %v, applied %d times", err, ran)
	}
}
