package call_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestTheCatalogueNamesTheDocumentACallPrintsAtItsVersion(t *testing.T) {
	plan := call.Call[struct{}, struct{}, struct{}]{
		Name: "machinefile.plan", Where: call.AtTarget, Applies: machine.Anywhere, Document: 2,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (struct{}, error) { return struct{}{}, nil },
	}
	list := call.Watch[struct{}, struct{}, struct{}]{
		Name: "apps.list", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (<-chan call.Update[struct{}], error) {
			return nil, nil
		},
	}
	r, err := call.NewRegistry([]call.Handler{call.Bind(plan, struct{}{}), call.BindWatch(list, struct{}{})})
	if err != nil {
		t.Fatal(err)
	}
	data, err := r.Catalogue()
	if err != nil {
		t.Fatal(err)
	}
	var c call.Catalogue
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	listed := map[string]string{}
	for _, e := range c.Calls {
		listed[e.Name] = e.Schema
	}
	for name, printed := range map[string]string{plan.Name: plan.Schema(), list.Name: list.Schema()} {
		if listed[name] != printed {
			t.Errorf("the catalogue lists %s as %q, its surface prints %q", name, listed[name], printed)
		}
	}
	if plan.Schema() != "station.machinefile.plan/v2" || list.Schema() != "station.apps.list/v1" {
		t.Errorf("schemas %q and %q, want station.machinefile.plan/v2 and station.apps.list/v1", plan.Schema(), list.Schema())
	}
}
