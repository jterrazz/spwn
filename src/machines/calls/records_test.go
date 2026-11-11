package calls_test

import (
	"context"
	"testing"
	"time"

	model "github.com/jterrazz/studio-station/src/machines"
	machinesfake "github.com/jterrazz/studio-station/src/machines/adapter/fake"
	machines "github.com/jterrazz/studio-station/src/machines/calls"
)

// invariant: the engine's save of a VM's record keeps the birth stored,
// read in the same update as its write, and writes its own env.
func TestASaveKeepsTheStoredBirthAndWritesItsEnv(t *testing.T) {
	ctx := context.Background()
	store := machinesfake.NewStore()
	booting := &model.Birth{Progress: 400, Step: model.BornBooting, At: time.Unix(0, 0).UTC()}
	if err := store.Put(ctx, model.Record{Name: "linux-4", Birth: booting}); err != nil {
		t.Fatal(err)
	}
	m := machines.New(nil, nil, machines.Prefs{}, store, nil, nil, nil, nil, machines.Options{})
	if err := m.Save(model.VMRecord{Name: "linux-4", Env: map[string]string{"PORT": "40001"}, Birth: &model.Birth{Step: "a stale one"}}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "linux-4")
	if err != nil || got.Birth == nil || *got.Birth != *booting || got.Env["PORT"] != "40001" {
		t.Errorf("record after the save = %+v, %v; want the stored birth and PORT 40001", got, err)
	}
}
