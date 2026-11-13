package fake

import (
	"context"
	"io/fs"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Ledger struct {
	// invariant: what one Mac of the world keeps of the installs station made.
	w   *world.World
	mac string
}

var (
	_ contract.Ledger = (*Ledger)(nil)
	_ store.Migrator  = (*Ledger)(nil)
	_ contract.Owned  = Owned{}
)

func LedgerOf(w *world.World, mac string) *Ledger { return &Ledger{w: w, mac: mac} }

func (l *Ledger) List(context.Context) ([]machinefile.Installed, error) {
	l.w.Lock()
	defer l.w.Unlock()
	return slices.Clone(l.w.Mac(l.mac).Store().Ledger), nil
}

func (l *Ledger) Put(_ context.Context, in machinefile.Installed) error {
	l.w.Lock()
	defer l.w.Unlock()
	st := l.w.Mac(l.mac).Store()
	st.Ledger = append(slices.DeleteFunc(st.Ledger, func(e machinefile.Installed) bool { return e.Package == in.Package }), in)
	slices.SortFunc(st.Ledger, func(a, b machinefile.Installed) int { return strings.Compare(a.Package, b.Package) })
	return nil
}

func (l *Ledger) Delete(_ context.Context, pkg string) error {
	l.w.Lock()
	defer l.w.Unlock()
	st := l.w.Mac(l.mac).Store()
	st.Ledger = slices.DeleteFunc(st.Ledger, func(e machinefile.Installed) bool { return e.Package == pkg })
	return nil
}

func (l *Ledger) Night(context.Context) (machinefile.Night, bool, error) {
	l.w.Lock()
	defer l.w.Unlock()
	n := l.w.Mac(l.mac).Store().Night
	if n == nil {
		return machinefile.Night{}, false, nil
	}
	return *n, true, nil
}

func (l *Ledger) Write(_ context.Context, n machinefile.Night) error {
	l.w.Lock()
	defer l.w.Unlock()
	l.w.Mac(l.mac).Store().Night = &n
	return nil
}

func (l *Ledger) Blocks(context.Context) ([]machinefile.Written, error) {
	l.w.Lock()
	defer l.w.Unlock()
	return slices.Clone(l.w.Mac(l.mac).Store().Blocks), nil
}

func (l *Ledger) PutBlock(_ context.Context, w machinefile.Written) error {
	l.w.Lock()
	defer l.w.Unlock()
	st := l.w.Mac(l.mac).Store()
	st.Blocks = append(slices.DeleteFunc(st.Blocks, func(e machinefile.Written) bool { return e.ID == w.ID }), w)
	slices.SortFunc(st.Blocks, func(a, b machinefile.Written) int { return strings.Compare(a.ID, b.ID) })
	return nil
}

func (l *Ledger) DeleteBlock(_ context.Context, id string) error {
	l.w.Lock()
	defer l.w.Unlock()
	st := l.w.Mac(l.mac).Store()
	st.Blocks = slices.DeleteFunc(st.Blocks, func(e machinefile.Written) bool { return e.ID == id })
	return nil
}

func (l *Ledger) Migrate(context.Context) error { return nil }

// value: the owner's folder of a Mac of the world: a file named by its
// path is a station file of that Mac under that path (`world edit <mac>
// <path>`), and a Machinefile kept lands where the owner's source reads it.

type Owned struct {
	w   *world.World
	mac string
}

func OwnedOf(w *world.World, mac string) Owned { return Owned{w: w, mac: mac} }

func (o Owned) Read(_ context.Context, path string) ([]byte, error) {
	o.w.Lock()
	defer o.w.Unlock()
	body, ok := o.w.Mac(o.mac).Files[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}
	return []byte(body), nil
}

func (o Owned) Keep(_ context.Context, name string, body []byte) error {
	o.w.Lock()
	defer o.w.Unlock()
	m := o.w.Mac(o.mac)
	at := "machinefiles/" + name + catalog.MachinefileSuffix
	if there, ok := m.Files[at]; ok && there != string(body) {
		return &contract.KeptError{Name: name, Path: at}
	}
	if m.Files == nil {
		m.Files = map[string]string{}
	}
	m.Files[at] = string(body)
	return nil
}
