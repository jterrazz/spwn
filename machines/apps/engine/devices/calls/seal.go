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
	pub, err := s.Key.Kept(ctx)
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
