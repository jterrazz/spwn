package fake

import (
	"context"

	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/world"
)

type Waker struct {
	// invariant: the Wake-on-LAN of the world, sent from one Mac: each packet
	// is kept on it; the Mac whose Self names that address wakes when it was
	// asleep (told so, or on battery with its lid closed) and silent, and
	// answers again; a hold of Waker.Wake is a packet the Mac never heard.
	w    *world.World
	from string
}

func WakerOf(w *world.World, from string) *Waker { return &Waker{w: w, from: from} }

func NewWaker() *Waker { return WakerOf(&world.World{}, "this-mac") }

func (k *Waker) Wake(_ context.Context, ether string) error {
	addr, ok := model.Ether(ether)
	if !ok {
		return contract.ErrBadHardwareAddress
	}
	k.w.Lock()
	defer k.w.Unlock()
	from := k.w.Mac(k.from)
	from.WakeSent = append(from.WakeSent, model.Spelled(addr))
	if k.w.Holds("Waker.Wake") {
		return nil
	}
	for _, m := range k.w.Macs {
		if m.Self == nil || !m.Offline || m.Power == nil {
			continue
		}
		named, ok := model.Ether(m.Self.Hardware.Ether)
		if !ok || named != addr || (!m.Power.Sleeping && !m.Power.LidClosedOnBattery()) {
			continue
		}
		m.Offline, m.Power.Sleeping = false, false
	}
	return nil
}

func (k *Waker) Sent() []string {
	k.w.Lock()
	defer k.w.Unlock()
	return append([]string(nil), k.w.Mac(k.from).WakeSent...)
}
