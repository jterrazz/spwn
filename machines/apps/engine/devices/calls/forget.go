package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/devices"
)

type ForgetIn struct {
	Name    string
	Confirm string
}

type Forgotten struct {
	Name string `json:"name"`
}

var Forget = call.Call[Deps, ForgetIn, Forgotten]{
	Name: "devices.forget", Where: call.InEngine, Applies: devices.OnAMac, Confirm: confirm.ForgetDevice, Journal: keptForget,
	Errors: []fail.Code{devices.Unknown, confirm.Needed, confirm.Mismatch},
	Run: func(ctx context.Context, d Deps, in ForgetIn, _ call.Progress) (Forgotten, error) {
		// invariant: taking trust away asks no presence and runs from afar,
		// so a Mac with nobody at it drops a stolen device at once; a broken
		// entry is forgotten like a trusted one.
		book, err := d.Store.Read(ctx)
		if err != nil {
			return Forgotten{}, err
		}
		if _, ok := book.Find(in.Name); !ok {
			return Forgotten{}, devices.Refusal(fail.NotFound, devices.Unknown, in.Name)
		}
		if err = confirm.Require(confirm.For(confirm.ForgetDevice, machine.Computer), in.Name, in.Confirm); err != nil {
			return Forgotten{}, err
		}
		err = d.Store.Change(ctx, func(b *devices.Book) error {
			if !b.Forget(in.Name) {
				return devices.Refusal(fail.NotFound, devices.Unknown, in.Name)
			}
			return nil
		})
		if err != nil {
			return Forgotten{}, err
		}
		return Forgotten{Name: in.Name}, nil
	},
}

func keptForget(in ForgetIn, _ Forgotten, err error) []journal.Line {
	return kept(in.Name, "forgotten", err)
}
