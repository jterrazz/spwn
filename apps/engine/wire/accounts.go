package wire

import (
	"context"
	"encoding/json"
	"path"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/cli"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/tailscale"
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	accountscontract "github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/guest"
)

func (m machine) accounts() *accountscalls.Reader {
	o := accountscalls.ReaderOptions{ThisMachine: m.kind, ThisMacName: m.where.Machine, Now: m.sys.Clock.Now}
	ids := m.identities(engine.ChildrenDir(m.paths.State))
	if m.kind != catalog.Mac {
		o.Keys = m.stationHostKeys
		return accountscalls.NewReader(m.pool, ids, accountscalls.ReaderPorts{}, o)
	}
	sec := m.secrets()
	d := m.engineAPI()
	sources := accountscalls.ReaderPorts{
		ShareList: sec.ShareList,
		VMs: func(ctx context.Context) ([]accounts.VM, error) {
			recs, err := m.machines(false).Records()
			if err != nil {
				return nil, err
			}
			shares, _ := m.stores.shares.List(ctx)
			live, _ := d.VMs(ctx)
			out := make([]accounts.VM, 0, len(recs))
			for _, r := range recs {
				vm := accounts.VM{Name: r.Name, Lifetime: string(r.Lifetime)}
				for _, v := range live {
					if v.Name == r.Name && v.State == engine.StateRunning {
						vm.Running = true
					}
				}
				for _, l := range shares[r.Name] {
					vm.Shares = append(vm.Shares, accounts.Loan(l))
				}
				out = append(out, vm)
			}
			return out, nil
		},
		Engine: func(ctx context.Context) ([]accounts.VMRead, string) {
			var view engine.AccountsView
			if err := d.do(ctx, engine.CallAccounts, nil, &view); err != nil {
				if _, missing := m.sys.FS.Stat(catalog.EnginePlist(m.sys.Env.Home())); missing != nil {
					return nil, accounts.EngineAbsent
				}
				return nil, accounts.EngineStopped
			}
			out := make([]accounts.VMRead, 0, len(view.VMs))
			for _, v := range view.VMs {
				read := accounts.VMRead{Name: v.Name, Guest: v.Guest}
				var rows []accounts.Account
				if json.Unmarshal(v.Accounts, &rows) == nil {
					for _, a := range rows {
						read.Accounts = append(read.Accounts, accounts.RedactAccount(a))
					}
				}
				out = append(out, read)
			}
			return out, accounts.EngineRunning
		},
	}
	return accountscalls.NewReader(m.pool, tailscale.New(m.tailnet(), ids), sources, o)
}

// value: inside a VM, who may ssh in is the station hosts block the guest
// writes into authorized_keys.
func (m machine) stationHostKeys() []string {
	data, err := m.sys.FS.ReadFile(path.Join(m.sys.Env.Home(), ".ssh", "authorized_keys"))
	if err != nil {
		return nil
	}
	var keys []string
	inside := false
	for _, line := range strings.Split(string(data), "\n") {
		switch strings.TrimSpace(line) {
		case guest.KeysBegin:
			inside = true
		case guest.KeysEnd:
			inside = false
		default:
			if inside && strings.TrimSpace(line) != "" {
				keys = append(keys, strings.TrimSpace(line))
			}
		}
	}
	return keys
}

// value: who each provider says is signed in here, read by its own app, each
// read's process recorded under track.
func (m machine) identities(track string) accountscontract.Identities {
	if m.world != nil {
		return m.world.identities()
	}
	return cli.New(m.sys.Run, m.sys.FS, cli.Options{Home: m.sys.Env.Home(), Track: track})
}
