package remote_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/fake"
)

const audienceAbility = "share.to"

type shareIn struct{ To string }

func (in shareIn) Asks() []string {
	if in.To == "" {
		return nil
	}
	return []string{audienceAbility}
}

var shareTo = call.Call[*int, shareIn, string]{
	Name: "share", Applies: machine.Anywhere, Abilities: []string{audienceAbility},
	Run: func(_ context.Context, ran *int, in shareIn, _ call.Progress) (string, error) {
		*ran++
		return "to " + in.To, nil
	},
}

// value: the laptop reaching a mini whose binary serves share, and lacks
// what lacks names of it.
func laptopSharingOn(t *testing.T, lacks ...string) (r *remote.Router, ran *int) {
	t.Helper()
	ran = new(int)
	near, err := call.NewRegistry([]call.Handler{call.Bind(shareTo, new(int))})
	if err != nil {
		t.Fatal(err)
	}
	far, err := call.NewRegistry([]call.Handler{call.Bind(shareTo, ran)})
	if err != nil {
		t.Fatal(err)
	}
	macs := fake.New()
	macs.Serve(mini, fake.Mac{Registry: far, Profile: machine.Profile{Ref: machine.Ref{Name: mini}, Kind: machine.Computer}, Version: "v3.2.0", Lacks: lacks})
	r = remote.NewRouter(near, laptop, remote.Reach{Macs: macs, Version: "v3.3.0"})
	t.Cleanup(r.Close)
	return r, ran
}

func TestAFarBinaryThatLacksAnAbilityTheInputAsksIsTooOldAndNeverRunsIt(t *testing.T) {
	r, ran := laptopSharingOn(t, audienceAbility)
	_, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, shareTo, shareIn{To: "private"}, nil)
	var e *fail.Error
	if !errors.As(err, &e) || e.Type != fail.FarTooOld || e.Subject != mini || e.Detail["version"] != "v3.2.0" || e.Detail["call"] != audienceAbility {
		t.Errorf("share to private on a mini that lacks its ability: %v, want far-too-old naming the mini, v3.2.0 and the ability", err)
	}
	if *ran != 0 {
		t.Errorf("the share ran %d times on the mini, want never: it would have shared with no audience", *ran)
	}
	if out, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, shareTo, shareIn{}, nil); err != nil || out != "to " || *ran != 1 {
		t.Errorf("a share that asks no audience on the same mini: %q, %v; want it served", out, err)
	}
}

func TestAFarBinaryThatListsTheAbilityRunsTheCallWithIt(t *testing.T) {
	r, ran := laptopSharingOn(t)
	out, err := call.Invoke(t.Context(), r, machine.Ref{Name: mini}, shareTo, shareIn{To: "private"}, nil)
	if err != nil || out != "to private" || *ran != 1 {
		t.Errorf("share to private on a mini that lists its ability: %q, %v; want it run there with the audience", out, err)
	}
}
