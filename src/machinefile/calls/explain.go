package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

type Deps struct {
	// value: the sources, strongest first: the owner's files, the shared
	// source when one is named, station's built-in files.
	Sources func() []contract.Source
	// value: this machine, for what a Machinefile does not say, and its
	// name, the Machinefile read when none is named.
	Machine machinefile.Machine
	Name    string
}

const (
	NotFound       fail.Code = "machinefile.not-found"
	Invalid        fail.Code = "machinefile.invalid"
	NewerSchema    fail.Code = "machinefile.newer-schema"
	Mixin          fail.Code = "machinefile.mixin"
	VMOnComputer   fail.Code = "machinefile.vm-on-a-computer"
	Cycle          fail.Code = "machinefile.cycle"
	NoMethod       fail.Code = "machinefile.no-method"
	Unpinnable     fail.Code = "machinefile.unpinnable"
	UnknownItem    fail.Code = "machinefile.unknown-item"
	BadValue       fail.Code = "machinefile.bad-value"
	UnknownKey     fail.Code = "machinefile.unknown-key"
	Unavailable    fail.Code = "machinefile.source-unavailable"
	CatalogInvalid fail.Code = "catalog.invalid"
	CatalogNewer   fail.Code = "catalog.newer-schema"
)

func resolution() []fail.Code {
	// value: every code reading the Machinefile may end in.
	return []fail.Code{NotFound, Invalid, NewerSchema, Mixin, VMOnComputer, Cycle, NoMethod, Unpinnable, UnknownItem, BadValue, Unavailable, CatalogInvalid, CatalogNewer}
}

type ExplainIn struct {
	// value: the key explained and every key under it; empty, every value.
	Key string
	// value: the Machinefile read; empty, the one named after the machine.
	Machinefile string
}

type ExplainOut struct {
	Name   string
	Files  []machinefile.Origin
	Values []machinefile.Value
}

var Explain = call.Call[Deps, ExplainIn, ExplainOut]{
	// value: each value of a machine's Machinefile, resolved against the
	// catalog, with the layer and the file it was read from.
	Name: "machinefile.explain", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: append([]fail.Code{UnknownKey}, resolution()...),
	Run: func(ctx context.Context, d Deps, in ExplainIn, _ call.Progress) (ExplainOut, error) {
		name := in.Machinefile
		if name == "" {
			name = d.Name
		}
		r, err := resolve(ctx, d, name)
		if err != nil {
			return ExplainOut{}, err
		}
		values, err := machinefile.Explain(r, in.Key)
		if err != nil {
			return ExplainOut{}, Worded(err)
		}
		return ExplainOut{Name: r.Name, Files: r.Files, Values: values}, nil
	},
}

func resolve(ctx context.Context, d Deps, name string) (machinefile.Resolved, error) {
	var trees []machinefile.Tree
	for _, s := range d.Sources() {
		t, err := s.Read(ctx)
		if err != nil {
			return machinefile.Resolved{}, Worded(err)
		}
		trees = append(trees, t)
	}
	lib, err := machinefile.Load(trees)
	if err != nil {
		return machinefile.Resolved{}, Worded(err)
	}
	r, err := machinefile.Resolve(lib, name, d.Machine)
	return r, Worded(err)
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(Explain, d)}
}

func Worded(err error) error {
	// invariant: each refusal of the resolution is one code; its words are built from its subject and its detail alone,
	// so a far machine's refusal reads the same here.
	if err == nil {
		return nil
	}
	refused := func(t fail.Type, code fail.Code, subject string, detail map[string]string) error {
		e := fail.New(t, code, subject, detail)
		e.Cause = err
		return e
	}
	var (
		file      *machinefile.RefusedFileError
		notFound  *machinefile.NotFoundError
		mixin     *machinefile.MixinError
		vm        *machinefile.VMOnComputerError
		cycle     *machinefile.CycleError
		none      *machinefile.NoMethodError
		pin       *machinefile.UnpinnableError
		item      *machinefile.UnknownItemError
		value     *machinefile.BadValueError
		key       *machinefile.UnknownKeyError
		gone      *contract.UnavailableError
		newer     *catalog.NewerSchemaError
		fileError *catalog.FileError
	)
	switch {
	case errors.As(err, &file) && errors.As(err, &newer):
		code := NewerSchema
		if file.Format == "catalog" {
			code = CatalogNewer
		}
		return refused(fail.Refused, code, newer.File, map[string]string{"schema": newer.Schema, "reads": newer.Reads})
	case errors.As(err, &file) && errors.As(err, &fileError):
		code := Invalid
		if file.Format == "catalog" {
			code = CatalogInvalid
		}
		return refused(fail.Refused, code, fileError.File, map[string]string{"problem": fileError.Error()})
	case errors.As(err, &notFound):
		return refused(fail.NotFound, NotFound, notFound.Name, map[string]string{"by": notFound.By})
	case errors.As(err, &mixin):
		return refused(fail.Refused, Mixin, mixin.Name, nil)
	case errors.As(err, &vm):
		return refused(fail.Refused, VMOnComputer, vm.Name, map[string]string{"file": vm.File})
	case errors.As(err, &cycle):
		return refused(fail.Refused, Cycle, cycle.Said(), map[string]string{"of": cycle.Of})
	case errors.As(err, &none):
		return refused(fail.Refused, NoMethod, none.Item, map[string]string{"system": none.System, "manager": none.Manager, "why": none.Why})
	case errors.As(err, &pin):
		return refused(fail.Refused, Unpinnable, pin.Item, map[string]string{"version": pin.Version, "method": pin.Method, "alternative": pin.Alternative})
	case errors.As(err, &item):
		return refused(fail.NotFound, UnknownItem, item.Name, map[string]string{"by": item.By, "is": item.Is})
	case errors.As(err, &value):
		return refused(fail.Refused, BadValue, value.Item, map[string]string{"key": value.Key, "problem": value.Problem})
	case errors.As(err, &key):
		return refused(fail.NotFound, UnknownKey, key.Key, map[string]string{"name": key.Name})
	case errors.As(err, &gone):
		return refused(fail.Failed, Unavailable, gone.Source, map[string]string{"commit": gone.Commit, "cause": gone.Cause.Error()})
	}
	return err
}
