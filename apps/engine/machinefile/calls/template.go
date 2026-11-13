package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func library(ctx context.Context, d Deps) (machinefile.Library, error) {
	trees := make([]machinefile.Tree, 0, 3)
	for _, s := range d.Sources() {
		t, err := s.Read(ctx)
		if err != nil {
			return machinefile.Library{}, err
		}
		trees = append(trees, t)
	}
	return machinefile.Load(trees)
}

func Template(ctx context.Context, d Deps, name string) (machinefile.Template, error) {
	// invariant: a template's lock is this machine's lock of it, its shared
	// source left out: the build VM reads every file of the chain from its
	// owner's folder, as this machine read it, and no shared source of its
	// own.
	lib, err := library(ctx, d)
	if err != nil {
		return machinefile.Template{}, err
	}
	held, ok, err := d.Locks.Read(ctx, name)
	if err != nil {
		return machinefile.Template{}, err
	}
	var lock []byte
	if ok {
		held.Source = nil
		if lock, err = catalog.EncodeLock(held); err != nil {
			return machinefile.Template{}, err
		}
	}
	return machinefile.TemplateOf(lib, name, lock)
}

func Templates(ctx context.Context, d Deps) ([]string, error) {
	lib, err := library(ctx, d)
	if err != nil {
		return nil, err
	}
	return lib.Templates(), nil
}
