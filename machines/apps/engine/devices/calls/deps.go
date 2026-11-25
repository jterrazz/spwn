package calls

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
)

type Deps struct {
	Store    contract.Store
	Sealer   Sealer
	Presence devices.Presence
	Now      func() time.Time
	// value: tells the engine the book changed, so its VMs trust what it
	// says now; nil tells nobody.
	Changed func()
}

func (d Deps) changed() {
	if d.Changed != nil {
		d.Changed()
	}
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(List, d), call.Bind(Add, d), call.Bind(Forget, d)}
}
