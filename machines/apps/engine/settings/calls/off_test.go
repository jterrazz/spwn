package calls_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/calls"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type never struct{ contract.AppliedStore }

func (never) Get(context.Context, string) (settings.Applied, error) {
	return settings.Applied{}, store.ErrNotFound
}
func (never) Delete(context.Context, string) error { return nil }

type unreadable struct{ contract.Applier }

func (unreadable) Kind() catalog.EffectKind { return catalog.EffectTouch }
func (unreadable) Observe(context.Context, catalog.Setting) (settings.Observation, error) {
	return settings.Observation{}, errors.New("/root/.ssh/authorized_keys is a link, not touched")
}
func (unreadable) Kept(context.Context, catalog.Setting) error {
	return errors.New("/root/.ssh/authorized_keys is a link, not touched")
}

func TestOffOnASettingItCannotReadNeedsYouNeverDone(t *testing.T) {
	d := deps()
	d.Applied = never{}
	d.Appliers = contract.NewAppliers(unreadable{})
	changes, err := calls.Off.Run(context.Background(), d, calls.OffIn{Names: []string{touched.Name}, Confirm: "yes"}, nil)
	if err != nil || len(changes) != 1 || changes[0].Result != catalog.StateBroken || len(changes[0].Steps) != 1 || changes[0].Steps[0].Label != "needs you" {
		t.Fatalf("off on a setting it cannot read = %+v, %v; want it needing you", changes, err)
	}
}
