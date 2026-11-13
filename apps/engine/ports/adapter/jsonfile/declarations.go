package jsonfile

import (
	"context"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

const DeclarationsSchema = "station.public/v1"

type Declarations struct{ list listFile[ports.Declaration] }

var (
	_ contract.DeclarationStore = (*Declarations)(nil)
	_ store.Migrator            = (*Declarations)(nil)
)

func NewDeclarations(files Files, file string) *Declarations {
	return &Declarations{list: listFile[ports.Declaration]{files: files, path: file, schema: DeclarationsSchema, key: "ports", version: 1}}
}

func (d *Declarations) Get(_ context.Context, machine string, port int) (ports.Declaration, error) {
	all, err := d.list.load()
	if err != nil {
		return ports.Declaration{}, err
	}
	if i := slices.IndexFunc(all, func(x ports.Declaration) bool { return x.Machine == machine && x.Port == port }); i >= 0 {
		return all[i], nil
	}
	return ports.Declaration{}, store.ErrNotFound
}

func (d *Declarations) Put(_ context.Context, decl ports.Declaration) error {
	all, err := d.list.load()
	if err != nil {
		return err
	}
	all = slices.DeleteFunc(all, func(x ports.Declaration) bool { return x.Machine == decl.Machine && x.Port == decl.Port })
	return d.save(append(all, decl))
}

func (d *Declarations) List(context.Context) ([]ports.Declaration, error) {
	all, err := d.list.load()
	if err != nil {
		return nil, err
	}
	sortDeclarations(all)
	return all, nil
}

func (d *Declarations) Delete(_ context.Context, machine string, port int) error {
	all, err := d.list.load()
	if err != nil {
		return err
	}
	kept := slices.DeleteFunc(slices.Clone(all), func(x ports.Declaration) bool { return x.Machine == machine && x.Port == port })
	if len(kept) == len(all) {
		return nil
	}
	return d.save(kept)
}

func (d *Declarations) Migrate(context.Context) error {
	_, err := d.list.load()
	return store.OnlyNewer(err)
}

func (d *Declarations) save(all []ports.Declaration) error {
	sortDeclarations(all)
	return d.list.save(all)
}

func sortDeclarations(all []ports.Declaration) {
	slices.SortFunc(all, func(a, b ports.Declaration) int {
		if c := strings.Compare(a.Machine, b.Machine); c != 0 {
			return c
		}
		return a.Port - b.Port
	})
}
