package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

var List = call.Call[Deps, struct{}, devices.Listing]{
	Name: "devices.list", Where: call.AtTarget, Applies: devices.OnAMac,
	Run: func(ctx context.Context, d Deps, _ struct{}, _ call.Progress) (devices.Listing, error) {
		book, anchored, err := d.Store.Read(ctx)
		if err != nil {
			return devices.Listing{}, err
		}
		out := devices.Listing{Mac: d.Sealer.Mac, Devices: []devices.Row{}, Broken: []devices.Row{}}
		own, err := d.Sealer.Key.Kept(ctx)
		var none *contract.NoDeviceKeyError
		switch {
		case errors.As(err, &none):
		case err != nil:
			return devices.Listing{}, err
		default:
			if out.Key, err = devices.LineOf(own); err != nil {
				return devices.Listing{}, err
			}
		}
		checked := devices.Check(book, anchored, d.Sealer.Mac, own)
		out.Stale = checked.Stale
		for _, e := range checked.Trusted {
			out.Devices = append(out.Devices, devices.RowOf(e))
		}
		for _, e := range checked.Broken {
			out.Broken = append(out.Broken, devices.RowOf(e))
		}
		return out, nil
	},
}
