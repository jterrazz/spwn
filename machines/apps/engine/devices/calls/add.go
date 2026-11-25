package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/devices"
)

type AddIn struct {
	Name string
	// value: the device's public key line, "<type> <base64> [comment]".
	Key string
	// value: the channels it may open; none is all three.
	Channels []string
}

var Add = call.Call[Deps, AddIn, devices.Row]{
	Name: "devices.add", Where: call.InEngine, Applies: devices.OnAMac, Journal: keptAdd,
	Errors: []fail.Code{devices.AddHereOnly, devices.BadName, devices.BadKey, devices.BadChannels, devices.Duplicate, devices.OwnKey, devices.PresenceRefused, devices.NoPresence},
	Run: func(ctx context.Context, d Deps, in AddIn, _ call.Progress) (devices.Row, error) {
		// invariant: a key is added only by the owner present at the Mac
		// that accepts it: a request from another machine is refused before
		// anything is read, and presence is asked every time, never riding
		// an open key session.
		if call.Far(ctx) {
			return devices.Row{}, devices.Refusal(fail.Refused, devices.AddHereOnly, in.Name)
		}
		entry, err := devices.Entry(in.Name, in.Key, in.Channels, d.Now())
		if err != nil {
			return devices.Row{}, err
		}
		own, err := d.Sealer.Key.Public(ctx)
		if err != nil {
			return devices.Row{}, err
		}
		line, err := devices.LineOf(own)
		if err != nil {
			return devices.Row{}, err
		}
		book, err := d.Store.Read(ctx)
		if err != nil {
			return devices.Row{}, err
		}
		if err = book.Admits(entry, line, d.Sealer.Mac, own); err != nil {
			return devices.Row{}, err
		}
		if answer := d.Presence(ctx, devices.Reason(entry)); answer != devices.Given {
			if answer == devices.Refused {
				return devices.Row{}, devices.Refusal(fail.Refused, devices.PresenceRefused, in.Name)
			}
			return devices.Row{}, devices.Refusal(fail.Failed, devices.NoPresence, in.Name)
		}
		sealed, err := d.Sealer.Seal(ctx, entry)
		if err != nil {
			return devices.Row{}, err
		}
		err = d.Store.Change(ctx, func(b *devices.Book) error {
			return b.Readmit(sealed, line, d.Sealer.Mac, own)
		})
		if err != nil {
			return devices.Row{}, err
		}
		d.changed()
		return devices.RowOf(sealed), nil
	},
}

func keptAdd(in AddIn, _ devices.Row, err error) []journal.Line {
	return kept(in.Name, "added", err)
}

func kept(name, verb string, err error) []journal.Line {
	if err != nil {
		return nil
	}
	return []journal.Line{{Text: name + " " + verb, Detail: "device book"}}
}
