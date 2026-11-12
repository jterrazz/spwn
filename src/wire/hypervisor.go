package wire

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine/process"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/adapter/tart"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/platform/darwin"
)

func (m machine) hypervisor() contract.Hypervisor {
	app := tart.New(m.sys.Run, m.sys.FS, m.sys.Clock, processes{darwin.NewTables(m.vars)}, m.paths.State)
	if m.world != nil {
		item, _ := m.cat.App(catalog.HypervisorApp)
		install, _ := item.InstallOn(catalog.Mac)
		return m.world.hypervisor(app.Name(), install)
	}
	return app
}

type engineHypervisor struct{ contract.Hypervisor }

func (h engineHypervisor) Boot(ctx context.Context, vm string, n model.Network, mounts []model.Mount) (process.Run, error) {
	return h.Hypervisor.Boot(ctx, vm, contract.BootOptions{Network: n, Mounts: mounts})
}

func (h engineHypervisor) Adopt(ctx context.Context, vm string) (process.Run, bool) {
	return h.Hypervisor.Adopt(ctx, vm)
}

type buildHypervisor struct{ contract.Hypervisor }

func (h buildHypervisor) BootBuild(ctx context.Context, vm string) error {
	_, err := h.Boot(ctx, vm, contract.BootOptions{})
	return err
}

// value: this Mac's kernel tables as tart's adapter asks them.
type processes struct{ darwin.Tables }

func (k processes) Named(prefix string) ([]int, error) {
	procs, err := k.Processes()
	if err != nil {
		return nil, err
	}
	var pids []int
	for _, p := range procs.List {
		if strings.HasPrefix(p.Name, prefix) {
			pids = append(pids, p.PID)
		}
	}
	return pids, nil
}
