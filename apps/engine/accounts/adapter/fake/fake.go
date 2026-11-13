package fake

import (
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func mac(w *world.World, name string) *world.Mac {
	if w.Macs == nil {
		w.Macs = map[string]*world.Mac{}
	}
	m := w.Macs[name]
	if m == nil {
		m = &world.Mac{}
		w.Macs[name] = m
	}
	return m
}
