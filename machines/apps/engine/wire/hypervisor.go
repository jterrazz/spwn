package wire

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/tart"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
)

func (m machine) hypervisor() contract.Hypervisor {
	app := tart.New(m.sys.Run, m.sys.FS, m.sys.Clock, processes{darwin.NewTables(m.vars)}, m.paths.State, filepath.Join(m.hypervisorHome(), "vms"))
	if m.world != nil {
		pkg, _ := m.cat.App(catalog.HypervisorApp)
		install, _ := pkg.InstallOn(catalog.Mac)
		return m.world.hypervisor(app.Name(), install)
	}
	return app
}

// value: the folder the launcher gives the hypervisor as its home
// (TART_HOME), which keeps each VM's folder under vms/.
func (m machine) hypervisorHome() string { return filepath.Join(m.paths.Data, "vms") }

type engineHypervisor struct{ contract.Hypervisor }

func (h engineHypervisor) Boot(ctx context.Context, vm string, n model.Network, mounts []model.Mount, sync model.DiskSync) (process.Run, error) {
	return h.Hypervisor.Boot(ctx, vm, contract.BootOptions{Network: n, Mounts: mounts, Sync: sync})
}

func (h engineHypervisor) Adopt(ctx context.Context, vm string) (process.Run, bool) {
	return h.Hypervisor.Adopt(ctx, vm)
}

type buildHypervisor struct{ contract.Hypervisor }

// invariant: ADR-104: a build VM is thrown away on any failure and made
// again from its Machinefile, so it never waits on a flush; its seal is a
// clean poweroff the build waits for before it records the image.
func (h buildHypervisor) BootBuild(ctx context.Context, vm string) error {
	_, err := h.Boot(ctx, vm, contract.BootOptions{Sync: model.SyncNone})
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
