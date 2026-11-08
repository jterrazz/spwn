package udp

import (
	"context"

	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
)

// value: the port Wake-on-LAN's magic packet is sent to, by convention.
const wakePort = 9

type Waker struct{ net local.Net }

func New(net local.Net) *Waker { return &Waker{net: net} }

func (w *Waker) Wake(ctx context.Context, ether string) error {
	addr, ok := model.Ether(ether)
	if !ok {
		return contract.ErrBadHardwareAddress
	}
	return w.net.Broadcast(ctx, wakePort, model.MagicPacket(addr))
}
