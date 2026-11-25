package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

type Sealer struct {
	// value: this Mac's device key, which seals each entry and checks
	// every seal; its public half is the Enclave's, never device-key.pub.
	Key contract.DeviceKey
	Mac string
}

func (s Sealer) Seal(ctx context.Context, d devices.Device) (devices.Device, error) {
	pub, err := s.Key.Public(ctx)
	if err != nil {
		return devices.Device{}, err
	}
	sig, err := s.Key.Sign(ctx, devices.SealMessage(s.Mac, pub, d))
	if err != nil {
		return devices.Device{}, err
	}
	d.Seal = sig
	return d, nil
}

func (s Sealer) Check(ctx context.Context, b devices.Book) (devices.Checked, error) {
	// invariant: an empty book asks nothing of the device key, so reading
	// it never makes one.
	if len(b.Devices) == 0 {
		return devices.Check(b, s.Mac, nil), nil
	}
	pub, err := s.Key.Public(ctx)
	if err != nil {
		return devices.Checked{}, err
	}
	return devices.Check(b, s.Mac, pub), nil
}
