package fake

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Password struct {
	// invariant: whether one Mac of the world asks a password for a line
	// run as root: every machine does but one the world says asks none.
	w   *world.World
	mac string
}

var _ contract.Password = (*Password)(nil)

func PasswordOf(w *world.World, mac string) *Password { return &Password{w: w, mac: mac} }

func (p *Password) Asked(context.Context) (bool, error) {
	p.w.Lock()
	defer p.w.Unlock()
	return !p.w.Mac(p.mac).Passwordless, nil
}
