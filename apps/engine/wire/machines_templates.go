package wire

import (
	"context"
	"errors"
	"path"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	grantscalls "github.com/jterrazz/studio-station/apps/engine/grants/calls"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	vaultcalls "github.com/jterrazz/studio-station/apps/engine/vault/calls"
)

// value: the Machinefiles this Mac makes workspaces from, read from every
// source as plan reads them, and the drift of the images built from them.
type machineTemplates struct{ m machine }

func (t machineTemplates) read(ctx context.Context, name string) (machinefile.Template, error) {
	got, err := machinefilecalls.Template(ctx, t.m.machinefileDeps(nil), name)
	if err != nil {
		return machinefile.Template{}, machinefilecalls.Worded(err)
	}
	return got, nil
}

func (t machineTemplates) Template(ctx context.Context, name string) (model.Template, error) {
	got, err := t.read(ctx, name)
	if err != nil {
		return model.Template{}, err
	}
	size, err := model.ParseResources(got.CPUs, got.Memory, got.Disk)
	if err != nil {
		return model.Template{}, err
	}
	out := model.Template{
		Name: got.Name, OS: got.OS, Server: got.Kind == catalog.RoleServer, Lifetime: model.Lifetime(got.Lifetime),
		Network: got.Network, Size: size,
	}
	for _, mt := range got.Mounts {
		out.Mounts = append(out.Mounts, model.Mount{Computer: t.onMac(mt.Computer), Guest: mt.Guest, ReadOnly: mt.ReadOnly})
	}
	out.SignIns, out.Grants = got.SignIns, got.Grants
	for _, f := range got.Files {
		out.Files = append(out.Files, model.TemplateFile{Path: f.Path, Body: f.Body})
	}
	if got.Lock != nil {
		out.Files = append(out.Files, model.TemplateFile{Path: "machinefiles/" + got.Name + catalog.LockSuffix, Body: got.Lock})
	}
	return out, nil
}

// value: the folders of this Mac a VM shares at a start, read from the
// Machinefile it was made from as it says them now; none for a VM made
// from an image alone.
//
// invariant: a start never needs its Machinefile. One that cannot be read
// (missing, unparsable) shares the folders the last start shared, those
// still on this Mac, and says why. A folder of this Mac that is not there
// refuses the start of a workspace, naming it, since tart alone refuses a
// missing shared folder naming none ("A directory sharing device
// configuration is invalid"); an always-on server starts without it,
// saying so, since nothing may keep a server down.
func (m machine) vmMounts(ctx context.Context, vm string) (model.MountsRead, error) {
	rec, err := m.stores.machines.Get(ctx, vm)
	if err != nil {
		return model.MountsRead{}, err
	}
	if rec.Template == "" {
		return model.MountsRead{}, nil
	}
	t, err := machineTemplates{m}.Template(ctx, rec.Template)
	if err != nil {
		kept, gone := m.onThisMac(rec.Mounts)
		why := vm + "'s Machinefile " + rec.Template + " could not be read (" + firstLineOf(err.Error()) + "): it shares the folders of its last start"
		if len(rec.Mounts) == 0 {
			why = vm + "'s Machinefile " + rec.Template + " could not be read (" + firstLineOf(err.Error()) + "): it shares no folder"
		}
		if len(gone) > 0 {
			why += ", but " + gone[0].Computer + ", no folder of this Mac now"
		}
		return model.MountsRead{Mounts: kept, Why: why}, nil
	}
	kept, gone := m.onThisMac(t.Mounts)
	if len(gone) == 0 {
		return model.MountsRead{Mounts: kept}, nil
	}
	missing := gone[0].Computer + ", which " + vm + "'s Machinefile " + rec.Template + " shares at " + gone[0].Guest + ", is no folder of this Mac"
	if rec.Lifetime != model.AlwaysOn {
		return model.MountsRead{}, errors.New(missing + ": make it, or take it out of vm.mounts")
	}
	return model.MountsRead{Mounts: kept, Why: missing + ": the server starts without it"}, nil
}

// value: the mounts whose folder is on this Mac, and those whose is not.
func (m machine) onThisMac(mounts []model.Mount) (kept, gone []model.Mount) {
	for _, mt := range mounts {
		if info, err := m.sys.FS.Stat(mt.Computer); err != nil || !info.IsDir() {
			gone = append(gone, mt)
			continue
		}
		kept = append(kept, mt)
	}
	return kept, gone
}

func firstLineOf(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	return line
}

// value: a folder of a mount on this Mac, its `~` opened at the home.
func (t machineTemplates) onMac(folder string) string {
	if folder == "~" {
		return t.m.sys.Env.Home()
	}
	if rest, ok := strings.CutPrefix(folder, "~/"); ok {
		return path.Join(t.m.sys.Env.Home(), rest)
	}
	return folder
}

// value: a Machinefile's term as a share takes it: weeks in days; none, or
// one a share cannot last, is the share's own default.

func (t machineTemplates) Drift(ctx context.Context, name, version string) ([]string, error) {
	rec, err := t.m.stores.images.Get(ctx, name, version)
	if err != nil {
		return nil, err
	}
	now, err := t.read(ctx, name)
	if err != nil {
		return nil, err
	}
	return machinefile.ImageDrift([]byte(rec.Lock), now), nil
}

// value: a workspace's first start grants a program from this Mac's one
// password for it, through grants.add: Touch ID is asked as for any grant.
func (m machine) grantProgram(ctx context.Context, program, vm string) error {
	r := m.far.router
	if r == nil {
		return errors.New("this Mac's engine does not answer")
	}
	read, err := call.Read(ctx, r, coremachine.Ref{}, vaultcalls.VaultList, vaultcalls.VaultListIn{}, nil)
	if err != nil {
		return err
	}
	var entries []accounts.VaultEntry
	for _, e := range read.Entries {
		if string(e.Program) == program {
			entries = append(entries, e)
		}
	}
	switch len(entries) {
	case 0:
		return errors.New("no password for " + program + " in this Mac's vault")
	case 1:
	default:
		return errors.New(strconv.Itoa(len(entries)) + " passwords for " + program + " in this Mac's vault: grant one from the window")
	}
	_, err = call.Invoke(ctx, r, coremachine.Ref{}, grantscalls.GrantAdd, grantscalls.GrantAddIn{Entry: entries[0].ID, ToKind: accounts.ToMachine, To: vm}, nil)
	return err
}

// value: the programs a VM of this Mac is signed in to, read through
// this Mac as accounts.list reads them.
func (m machine) signedIn(ctx context.Context, vm string) (map[string]bool, error) {
	r := m.far.router
	if r == nil {
		return nil, errors.New("this Mac's engine does not answer")
	}
	read, err := call.Read(ctx, r, coremachine.Ref{}, accountscalls.Programs, accountscalls.ProgramsIn{Machine: vm}, nil)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, p := range read.Programs {
		out[string(p.Program)] = p.State == accounts.ProgramSignedIn
	}
	return out, nil
}
