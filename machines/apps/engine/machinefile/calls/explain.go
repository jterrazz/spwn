package calls

import (
	"cmp"
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines"
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
	// value: every Mac and the VMs on each, as This Mac lists them: the
	// machines that may follow a Machinefile.
	AllMachines func(ctx context.Context) (machines.AllMachines, error)
	// value: the drift of each follower of a file, asked a few at a time,
	// each within its budget (remote/each): one that does not answer in
	// time is its own error, never the others' delay.
	Judge func(ctx context.Context, keys []string, ask func(ctx context.Context, key string) (machinefile.Drift, error)) map[string]Judged
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
	NotATemplate   fail.Code = "machinefile.not-a-template"
	VMOnComputer   fail.Code = "machinefile.vm-on-a-computer"
	Cycle          fail.Code = "machinefile.cycle"
	SiblingsDiffer fail.Code = "machinefile.siblings-differ"
	ChainTooLong   fail.Code = "machinefile.chain-too-long"
	NoMethod       fail.Code = "machinefile.no-method"
	Unpinnable     fail.Code = "machinefile.unpinnable"
	UnknownPackage fail.Code = "machinefile.unknown-package"
	BadValue       fail.Code = "machinefile.bad-value"
	UnknownKey     fail.Code = "machinefile.unknown-key"
	Unavailable    fail.Code = "machinefile.source-unavailable"
	// value: a shared source read through a way that leaves its clone.
	OutsideTheSource fail.Code = "machinefile.outside-the-source"
	// value: a computer's chain reading run.upgrade from the shared source.
	UpgradeOnComputer fail.Code = "machinefile.upgrade-on-a-computer"
	// value: a computer planning, applying or upgrading a Machinefile of
	// another kind.
	AnotherKind    fail.Code = "machinefile.another-kind"
	Busy           fail.Code = "machinefile.busy"
	CatalogInvalid fail.Code = "catalog.invalid"
	CatalogNewer   fail.Code = "catalog.newer-schema"
)

func resolution() []fail.Code {
	// value: every code reading the Machinefile may end in.
	return []fail.Code{NotFound, Invalid, NewerSchema, Mixin, VMOnComputer, Cycle, SiblingsDiffer, ChainTooLong, NoMethod, Unpinnable, UnknownPackage, BadValue, Unavailable, OutsideTheSource, UpgradeOnComputer, AnotherKind, CatalogInvalid, CatalogNewer}
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
	// catalog, with the origin and the file it was read from.
	Name: "machinefile.explain", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: append([]fail.Code{UnknownKey}, slices.DeleteFunc(resolution(), func(c fail.Code) bool { return c == Mixin })...),
	Run: func(ctx context.Context, d Deps, in ExplainIn, _ call.Progress) (ExplainOut, error) {
		r, err := resolveValues(ctx, d, in.Machinefile)
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

func IsPath(s string) bool {
	// invariant: a name is lower-case words joined by dashes, so a slash or
	// a .yaml suffix names a Machinefile by its path.
	return strings.ContainsRune(s, '/') || strings.HasSuffix(s, ".yaml")
}

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
	if !IsPath(asked) {
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
	return mf.Name, &machinefile.Tree{Kind: machinefile.User, Root: filepath.Dir(asked), Files: []machinefile.File{{Path: "machinefiles/" + mf.Name + catalog.MachinefileSuffix, Body: body, At: asked}}}, nil
}

// value: the Machinefile resolved, with the catalog it was resolved
// against, which judges what the ledger holds that it lists no longer.
func resolveIn(ctx context.Context, d Deps, this machinefile.Machine, asked string) (machinefile.Resolved, catalog.Catalog, error) {
	name, lib, source, lock, err := load(ctx, d, asked)
	if err != nil {
		return machinefile.Resolved{}, catalog.Catalog{}, err
	}
	r, err := machinefile.Resolve(lib, name, this, lock)
	r.Source = source
	return r, lib.Catalog, Worded(err)
}

// invariant: this machine as a read of a Machinefile sees it (explain,
// plan, drift): with no kind, so a computer reads a Machinefile of
// another kind as that kind does. What carries a plan out (apply,
// upgrade and the night that runs it) resolves against Deps.Machine,
// the kind its profile says, and a computer refuses there a Machinefile
// that is not a computer's (machinefile.OtherKindError).
func (d Deps) reading() machinefile.Machine {
	return machinefile.Machine{Name: d.Machine.Name, System: d.Machine.System}
}

// value: what explain reads: the Machinefile resolved, or, for a mixin,
// the values its own chain says.
func resolveValues(ctx context.Context, d Deps, asked string) (machinefile.Resolved, error) {
	name, lib, source, lock, err := load(ctx, d, asked)
	if err != nil {
		return machinefile.Resolved{}, err
	}
	r, err := machinefile.Resolve(lib, name, d.reading(), lock)
	var mixin *machinefile.MixinError
	if errors.As(err, &mixin) {
		r, err = machinefile.Own(lib, name)
	}
	r.Source = source
	return r, Worded(err)
}

func load(ctx context.Context, d Deps, asked string) (name string, lib machinefile.Library, source *catalog.LockSource, lock machinefile.Lock, err error) {
	name, path, err := named(ctx, d, asked)
	if err != nil {
		return "", lib, nil, lock, err
	}
	var trees []machinefile.Tree
	if path != nil {
		trees = append(trees, *path)
	}
	for _, s := range d.Sources() {
		t, read := s.Read(ctx)
		if read != nil {
			return "", lib, nil, lock, Worded(read)
		}
		trees = append(trees, t)
	}
	for _, t := range trees {
		source = cmp.Or(source, t.Source)
	}
	if lib, err = machinefile.Load(trees); err != nil {
		return "", lib, nil, lock, Worded(err)
	}
	lock = machinefile.Lock{File: d.Locks.Where(name)}
	if lock.LockFile, _, err = d.Locks.Read(ctx, name); err != nil {
		return "", lib, nil, lock, Worded(err)
	}
	return name, lib, source, lock, nil
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(Explain, d), call.Bind(Plan, d), call.Bind(Apply, d), call.Bind(Drift, d), call.Bind(Upgrade, d), call.Bind(LastNight, d), call.Bind(ListFiles, d)}
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
		notImage  *machinefile.NotTemplateError
		cycle     *machinefile.CycleError
		siblings  *machinefile.SiblingConflictError
		tooLong   *machinefile.ChainTooLongError
		none      *machinefile.NoMethodError
		pin       *machinefile.UnpinnableError
		pkg       *machinefile.UnknownPackageError
		value     *machinefile.BadValueError
		key       *machinefile.UnknownKeyError
		gone      *contract.UnavailableError
		outside   *contract.OutsideError
		upgrade   *machinefile.UpgradeOnComputerError
		other     *machinefile.OtherKindError
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
		if fileError.Unknown != "" {
			detail["unknown"] = fileError.Unknown
		}
		return refused(fail.Refused, code, fileError.File, detail)
	case errors.As(err, &notFound):
		return refused(fail.NotFound, NotFound, notFound.Name, map[string]string{"by": notFound.By})
	case errors.As(err, &mixin):
		return refused(fail.Refused, Mixin, mixin.Name, nil)
	case errors.As(err, &notImage):
		return refused(fail.Refused, NotATemplate, notImage.Name, nil)
	case errors.As(err, &vm):
		return refused(fail.Refused, VMOnComputer, vm.Name, map[string]string{"file": vm.File})
	case errors.As(err, &cycle):
		return refused(fail.Refused, Cycle, cycle.Said(), map[string]string{"of": cycle.Of})
	case errors.As(err, &siblings):
		return refused(fail.Refused, SiblingsDiffer, siblings.Name, map[string]string{"kind": siblings.Kind, "first": siblings.First.Name, "first_at": siblings.First.At(), "second": siblings.Second.Name, "second_at": siblings.Second.At(), "top": siblings.Top})
	case errors.As(err, &tooLong):
		return refused(fail.Refused, ChainTooLong, tooLong.Path[0], map[string]string{"said": tooLong.Said()})
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
	case errors.As(err, &outside):
		return refused(fail.Refused, OutsideTheSource, outside.Source, map[string]string{"at": outside.At, "problem": outside.Problem})
	case errors.As(err, &upgrade):
		return refused(fail.Refused, UpgradeOnComputer, upgrade.Name, map[string]string{"says": upgrade.Says, "file": upgrade.File})
	case errors.As(err, &other):
		return refused(fail.Refused, AnotherKind, other.Name, map[string]string{"kind": other.Kind, "this": other.This})
	}
	return err
}
