package calls

import (
	"cmp"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

type Deps struct {
	// value: the sources, strongest first: the owner's files, the shared
	// source when one is named, station's built-in files.
	Sources func() []contract.Source
	// value: this machine, for what a Machinefile does not say, and its
	// name, the Machinefile read when none is named and none is bound.
	Machine machinefile.Machine
	Name    string
	Kind    machine.Kind
	// value: the Machinefile this machine is bound to, the ledger of what
	// station installed, the lock of what it carries, the owner's folder a
	// path is copied into, the vault a `vault:` reference is read from,
	// whether a line run as root asks a password here, the turn one apply,
	// upgrade or night holds at a time, and the clock.
	Prefs    contract.SourcePrefs
	Ledger   contract.Ledger
	Locks    contract.Locks
	Owned    contract.Owned
	Vault    Vault
	Password contract.Password
	Turn     contract.Turn
	// value: the shell the owner's lines of a phase run in.
	Shell contract.Shell
	Now   func() time.Time
	// value: this machine's ~/.zshrc, and the home its variables name.
	Zshrc Zshrc
	Home  string
	// value: this machine's own calls: apps and settings read and act
	// through them, never through their adapters.
	Calls call.Router
	// value: the calls of every machine, by target: a drift's packages
	// judged on another machine, and a VM's shares read from its Mac.
	Far call.Router
	// value: the Mac a VM runs on; ok false for a Mac's name or a name no
	// Mac holds.
	MacOf func(ctx context.Context, name string) (mac string, ok bool)
}

type Vault interface {
	// value: the secret a `vault:` reference names.
	Get(ctx context.Context, name string) ([]byte, error)
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
	UnknownPackage fail.Code = "machinefile.unknown-package"
	BadValue       fail.Code = "machinefile.bad-value"
	UnknownKey     fail.Code = "machinefile.unknown-key"
	Unavailable    fail.Code = "machinefile.source-unavailable"
	Busy           fail.Code = "machinefile.busy"
	CatalogInvalid fail.Code = "catalog.invalid"
	CatalogNewer   fail.Code = "catalog.newer-schema"
)

func resolution() []fail.Code {
	// value: every code reading the Machinefile may end in.
	return []fail.Code{NotFound, Invalid, NewerSchema, Mixin, VMOnComputer, Cycle, NoMethod, Unpinnable, UnknownPackage, BadValue, Unavailable, CatalogInvalid, CatalogNewer}
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
		r, err := resolve(ctx, d, in.Machinefile)
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

// value: whether the owner named a Machinefile by its path rather than
// its name: a name is lower-case words joined by dashes.
func isPath(s string) bool { return strings.ContainsRune(s, '/') || strings.HasSuffix(s, ".yaml") }

// invariant: the Machinefile read is the one named, else the one this
// machine is bound to, else the one named after the machine. One named by
// its path is read as the owner's file of its name, above every source.
func named(ctx context.Context, d Deps, asked string) (name string, path *machinefile.Tree, err error) {
	if asked == "" {
		if asked, err = d.Prefs.Bound(); err != nil {
			return "", nil, err
		}
	}
	if asked == "" {
		return d.Name, nil, nil
	}
	if !isPath(asked) {
		return asked, nil, nil
	}
	body, err := d.Owned.Read(ctx, asked)
	if err != nil {
		return "", nil, fail.New(fail.NotFound, NotFound, asked, map[string]string{"path": "yes"})
	}
	mf, err := catalog.DecodeMachinefile(asked, body)
	if err != nil {
		return "", nil, Worded(&machinefile.RefusedFileError{Format: "machinefile", Err: err})
	}
	if mf.Name == "" {
		return "", nil, Worded(&machinefile.RefusedFileError{Format: "machinefile", Err: &catalog.FileError{File: asked, Problem: "names no Machinefile: a file named by its path says its name"}})
	}
	return mf.Name, &machinefile.Tree{Layer: machinefile.User, Root: filepath.Dir(asked), Files: []machinefile.File{{Path: "machinefiles/" + mf.Name + catalog.MachinefileSuffix, Body: body, At: asked}}}, nil
}

func resolve(ctx context.Context, d Deps, asked string) (machinefile.Resolved, error) {
	r, _, err := resolveIn(ctx, d, asked)
	return r, err
}

// value: the Machinefile resolved, with the catalog it was resolved
// against, which judges what the ledger holds that it lists no longer.
func resolveIn(ctx context.Context, d Deps, asked string) (machinefile.Resolved, catalog.Catalog, error) {
	name, path, err := named(ctx, d, asked)
	if err != nil {
		return machinefile.Resolved{}, catalog.Catalog{}, err
	}
	var trees []machinefile.Tree
	if path != nil {
		trees = append(trees, *path)
	}
	for _, s := range d.Sources() {
		t, read := s.Read(ctx)
		if read != nil {
			return machinefile.Resolved{}, catalog.Catalog{}, Worded(read)
		}
		trees = append(trees, t)
	}
	var source *catalog.LockSource
	for _, t := range trees {
		source = cmp.Or(source, t.Source)
	}
	lib, err := machinefile.Load(trees)
	if err != nil {
		return machinefile.Resolved{}, catalog.Catalog{}, Worded(err)
	}
	lock := machinefile.Lock{File: d.Locks.Where(name)}
	if lock.LockFile, _, err = d.Locks.Read(ctx, name); err != nil {
		return machinefile.Resolved{}, catalog.Catalog{}, Worded(err)
	}
	r, err := machinefile.Resolve(lib, name, d.Machine, lock)
	r.Source = source
	return r, lib.Catalog, Worded(err)
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(Explain, d), call.Bind(Plan, d), call.Bind(Apply, d), call.Bind(Drift, d), call.Bind(Upgrade, d), call.Bind(LastNight, d)}
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
		pkg       *machinefile.UnknownPackageError
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
		detail := map[string]string{"problem": fileError.Error()}
		if fileError.Rename != "" {
			detail["rename"] = fileError.Rename
		}
		return refused(fail.Refused, code, fileError.File, detail)
	case errors.As(err, &notFound):
		return refused(fail.NotFound, NotFound, notFound.Name, map[string]string{"by": notFound.By})
	case errors.As(err, &mixin):
		return refused(fail.Refused, Mixin, mixin.Name, nil)
	case errors.As(err, &vm):
		return refused(fail.Refused, VMOnComputer, vm.Name, map[string]string{"file": vm.File})
	case errors.As(err, &cycle):
		return refused(fail.Refused, Cycle, cycle.Said(), map[string]string{"of": cycle.Of})
	case errors.As(err, &none):
		return refused(fail.Refused, NoMethod, none.Package, map[string]string{"system": none.System, "manager": none.Manager, "why": none.Why, "at": none.At})
	case errors.As(err, &pin):
		return refused(fail.Refused, Unpinnable, pin.Package, map[string]string{"version": pin.Version, "method": pin.Method, "holds": pin.Holds, "alternative": pin.Alternative, "at": pin.At})
	case errors.As(err, &pkg):
		return refused(fail.NotFound, UnknownPackage, pkg.Name, map[string]string{"by": pkg.By, "is": pkg.Is, "at": pkg.At})
	case errors.As(err, &value):
		return refused(fail.Refused, BadValue, value.Package, map[string]string{"key": value.Key, "problem": value.Problem, "at": value.At})
	case errors.As(err, &key):
		return refused(fail.NotFound, UnknownKey, key.Key, map[string]string{"name": key.Name})
	case errors.As(err, &gone):
		return refused(fail.Failed, Unavailable, gone.Source, map[string]string{"cause": gone.Cause.Error()})
	}
	return err
}
