package fake

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Uptime struct {
	// invariant: the uptime records one Mac of the world keeps, by server.
	w   *world.World
	mac string
}

var (
	_ contract.UptimeStore = (*Uptime)(nil)
	_ store.Migrator       = (*Uptime)(nil)
)

func NewUptime() *Uptime { return UptimeOf(&world.World{}, "this-mac") }

func UptimeOf(w *world.World, mac string) *Uptime { return &Uptime{w: w, mac: mac} }

func (u *Uptime) Get(_ context.Context, server string) (machines.Uptime, error) {
	u.w.Lock()
	defer u.w.Unlock()
	return u.w.Mac(u.mac).Store().Uptime[server], nil
}

func (u *Uptime) Put(_ context.Context, server string, record machines.Uptime) error {
	u.w.Lock()
	defer u.w.Unlock()
	s := u.w.Mac(u.mac).Store()
	if s.Uptime == nil {
		s.Uptime = map[string]machines.Uptime{}
	}
	s.Uptime[server] = record
	return nil
}

func (u *Uptime) Delete(_ context.Context, server string) error {
	u.w.Lock()
	defer u.w.Unlock()
	delete(u.w.Mac(u.mac).Store().Uptime, server)
	return nil
}

func (u *Uptime) Migrate(context.Context) error { return nil }
