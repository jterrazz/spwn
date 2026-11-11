package apt

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const (
	command = "apt-get"
	query   = "dpkg-query"
	policy  = "apt-cache"
	name    = "apt"
)

// invariant: apt never asks a question: debconf takes its defaults, and a
// configuration file the owner changed is kept, the package's own beside it.
func quiet() []string {
	return []string{"env", "DEBIAN_FRONTEND=noninteractive", command, "-y", "-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold"}
}

// invariant: a removal takes back the packages it names and nothing apt
// would remove with them, and never what they alone pulled in.
func keep() []string { return []string{"-o", "APT::Get::AutomaticRemove=false"} }

// invariant: what station parses of apt is read in apt's own words,
// whatever language the machine speaks: `Installed:`, `Candidate:`, `Remv`.
func read(name string, args ...string) local.Cmd {
	c := local.Command(name, args...)
	c.Env = []string{"LC_ALL=C"}
	return c
}

var _ contract.Holder = (*Apt)(nil)

type Apt struct {
	run   local.Runner
	files local.FS
	// value: station's folder of the packages its installs added, or were
	// adding when cut short, one file per package naming the apps that hold
	// it.
	store string
	// value: the run has a terminal, where sudo may ask its password; off
	// one, sudo never asks and fails where it would.
	terminal bool
}

func New(run local.Runner, files local.FS, store string, terminal bool) *Apt {
	return &Apt{run: run, files: files, store: store, terminal: terminal}
}

func (a *Apt) Method() catalog.Method { return catalog.MethodApt }

func (a *Apt) Manager() contract.Manager { return contract.Manager{Name: name, App: name} }

func (a *Apt) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true, UpdateAll: true}
}

func (a *Apt) Ready(context.Context) error {
	for _, c := range []string{command, query} {
		if _, ok := a.run.LookPath(c); !ok {
			return contract.ErrMissing
		}
	}
	return nil
}

func (a *Apt) Inventory(ctx context.Context) (map[string]string, error) {
	// invariant: a package is held when dpkg says it is installed (`ii`);
	// one removed with its configuration kept (`rc`) is not.
	if err := a.Ready(ctx); err != nil {
		return nil, err
	}
	out, err := a.run.Output(ctx, read(query, "-W", "-f", `${Package}\t${Version}\t${db:Status-Abbrev}\n`))
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errors.New("dpkg-query exited " + strconv.Itoa(out.ExitCode))
	}
	held := map[string]string{}
	for line := range strings.Lines(out.Stdout) {
		fields := strings.Split(strings.TrimRight(line, "\n"), "\t")
		if len(fields) == 3 && strings.HasPrefix(fields[2], "ii") {
			held[fields[0]] = fields[1]
		}
	}
	return held, nil
}

func (a *Apt) Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error) {
	// invariant: a package is behind when apt's candidate is another
	// version than the one installed, as its lists last read it; an
	// unread policy leaves its currency unknown.
	out := map[string]apps.Currency{}
	if len(keys) == 0 {
		return out, nil
	}
	for _, k := range keys {
		out[k] = apps.Currency{Verdict: apps.Unknown}
	}
	var named []string
	for _, k := range keys {
		if catalog.CheckPackage(k) == nil {
			named = append(named, k)
		}
	}
	if len(named) == 0 {
		return out, nil
	}
	read, err := a.run.Output(ctx, read(policy, slices.Concat([]string{"policy", "--"}, named)...))
	if err == nil && read.ExitCode == 0 {
		candidates(read.Stdout, keys, out)
	}
	return out, nil
}

func candidates(policy string, keys []string, out map[string]apps.Currency) {
	pkg, installed := "", ""
	for line := range strings.Lines(policy) {
		line = strings.TrimRight(line, "\n")
		field := strings.TrimSpace(line)
		switch {
		case line != "" && !strings.HasPrefix(line, " ") && strings.HasSuffix(line, ":"):
			pkg, installed = strings.TrimSuffix(line, ":"), ""
		case strings.HasPrefix(field, "Installed: "):
			installed = strings.TrimPrefix(field, "Installed: ")
		case strings.HasPrefix(field, "Candidate: ") && slices.Contains(keys, pkg) && installed != "(none)" && installed != "":
			candidate := strings.TrimPrefix(field, "Candidate: ")
			out[pkg] = apps.Currency{Verdict: apps.Current}
			if candidate != installed && candidate != "(none)" {
				out[pkg] = apps.Currency{Verdict: apps.Behind, Newest: candidate}
			}
		}
	}
}

// invariant: apt installs and removes for an app, through InstallFor and
// RemoveFor, since what it records is held by that app; its plain verbs
// refuse, never guessing the app from a package.
var errServes = errors.New("apt installs and removes for an app: through InstallFor and RemoveFor")

func (a *Apt) Install(context.Context, []catalog.Install, contract.Logs) error { return errServes }

func (a *Apt) Remove(context.Context, []catalog.Install, contract.Logs) error { return errServes }

func (a *Apt) InstallFor(ctx context.Context, f contract.For, installs []catalog.Install, logs contract.Logs) error {
	// invariant: before apt runs, each of the app's packages dpkg does not
	// hold is recorded the app's, as what this install may add, so an
	// install cut short leaves the record of all it may have added; one dpkg
	// held before is the owner's and stays unrecorded, unless station's
	// record of it is there already, where the app joins its holders. Once
	// apt has run, interrupted or not, a package so recorded that dpkg does
	// not hold is let go by the app.
	if err := checkApp(f.App); err != nil {
		return err
	}
	names, asked, err := packages(installs, true)
	if err != nil {
		return err
	}
	before, err := a.Inventory(ctx)
	if err != nil {
		return err
	}
	var intent []string
	held := map[string][]string{}
	for _, v := range asked {
		for _, p := range v.Packages {
			if slices.Contains(intent, p) {
				continue
			}
			h, recorded, err := a.holders(p)
			if err != nil {
				return err
			}
			if _, had := before[p]; had && !recorded {
				continue
			}
			held[p] = add(h, f.App)
			intent = append(intent, p)
		}
	}
	if err := a.refresh(ctx, logs); err != nil {
		return err
	}
	if err := a.files.MkdirAll(a.store, 0o755); err != nil {
		return err
	}
	for _, p := range intent {
		if err := a.record(p, held[p]); err != nil {
			return err
		}
	}
	ran := a.root(ctx, logs, slices.Concat(quiet(), []string{"install", "--"}, names)...)
	past, cancel := context.WithTimeout(context.WithoutCancel(ctx), settleBudget)
	defer cancel()
	if err := a.settle(past, f.App, intent); err != nil && ran == nil {
		return err
	}
	return ran
}

// value: how long the read after an install may take, past an interrupt.
const settleBudget = 30 * time.Second

func (a *Apt) settle(ctx context.Context, app string, intent []string) error {
	now, err := a.Inventory(ctx)
	if err != nil {
		return err
	}
	for _, p := range intent {
		if _, there := now[p]; there {
			continue
		}
		held, _, err := a.holders(p)
		if err != nil {
			return err
		}
		if err := a.let(p, drop(held, app)); err != nil {
			return err
		}
	}
	return nil
}

func (a *Apt) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	names, _, err := packages(installs, false)
	if err != nil {
		return err
	}
	if err := a.refresh(ctx, logs); err != nil {
		return err
	}
	return a.root(ctx, logs, slices.Concat(quiet(), []string{"install", "--only-upgrade", "--"}, names)...)
}

func (a *Apt) RemoveFor(ctx context.Context, f contract.For, installs []catalog.Install, logs contract.Logs) error {
	// invariant: a removal takes back the packages station recorded adding
	// for the app, whatever the app lists now, each once no other app holds
	// it and only while dpkg holds it: one the owner had before stays, and
	// one another app's install holds stays while that app does. An app
	// station recorded adding nothing for is the owner's: on the owner's
	// word for it, its removal takes the package its key names alone, unless
	// a record names it another app's; without that word nothing of it is
	// taken. apt removes every package that depends on one it removes, so a
	// removal that would take any package it was not asked for is refused
	// before it runs.
	if err := checkApp(f.App); err != nil {
		return err
	}
	_, asked, err := packages(installs, false)
	if err != nil {
		return err
	}
	records, err := a.records()
	if err != nil {
		return err
	}
	now, err := a.Inventory(ctx)
	if err != nil {
		return err
	}
	var names []string
	left := map[string][]string{}
	for p, held := range records {
		if !slices.Contains(held, f.App) {
			continue
		}
		left[p] = drop(held, f.App)
		if _, there := now[p]; there && len(left[p]) == 0 {
			names = append(names, p)
		}
	}
	if len(left) == 0 {
		if !f.Word {
			return &apps.UnrecordedError{App: f.App}
		}
		for _, v := range asked {
			key := v.Packages[0]
			_, recorded := records[key]
			if _, there := now[key]; there && !recorded && !slices.Contains(names, key) {
				names = append(names, key)
			}
		}
	}
	slices.Sort(names)
	if len(names) > 0 {
		if err := a.take(ctx, logs, names); err != nil {
			return err
		}
	}
	for _, p := range slices.Sorted(maps.Keys(left)) {
		if err := a.let(p, left[p]); err != nil {
			return err
		}
	}
	return nil
}

func (a *Apt) Holds(_ context.Context, app string) (bool, error) {
	records, err := a.records()
	if err != nil {
		return false, err
	}
	for _, held := range records {
		if slices.Contains(held, app) {
			return true, nil
		}
	}
	return false, nil
}

func (a *Apt) take(ctx context.Context, logs contract.Logs, names []string) error {
	sim, err := a.run.Output(ctx, read(command, slices.Concat([]string{"-s", "remove"}, keep(), []string{"--"}, names)...))
	if err != nil {
		return err
	}
	if sim.ExitCode != 0 {
		return errors.New("apt-get -s remove exited " + strconv.Itoa(sim.ExitCode) + ": " + strings.TrimSpace(sim.Stderr))
	}
	var also []string
	for line := range strings.Lines(sim.Stdout) {
		if f := strings.Fields(line); len(f) > 1 && f[0] == "Remv" && !slices.Contains(names, f[1]) {
			also = append(also, f[1])
		}
	}
	if len(also) > 0 {
		return &TakesMoreError{Asked: names, Also: also}
	}
	return a.root(ctx, logs, slices.Concat(quiet(), []string{"remove"}, keep(), []string{"--"}, names)...)
}

// value: the apps that hold a package station's install added (or was
// adding, cut short), by their catalogue names, one to a line of the
// package's file in station's folder; none recorded is the owner's.
func (a *Apt) holders(pkg string) (held []string, recorded bool, err error) {
	data, err := a.files.ReadFile(path.Join(a.store, pkg))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	}
	for line := range strings.Lines(string(data)) {
		if app := strings.TrimSpace(line); app != "" {
			held = append(held, app)
		}
	}
	return held, true, nil
}

// value: every package station's folder records, with its holders; a file
// whose name is no package is no record of station's.
func (a *Apt) records() (map[string][]string, error) {
	entries, err := a.files.ReadDir(a.store)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return map[string][]string{}, nil
	case err != nil:
		return nil, err
	}
	out := map[string][]string{}
	for _, e := range entries {
		if e.IsDir() || catalog.CheckPackage(e.Name()) != nil {
			continue
		}
		held, _, err := a.holders(e.Name())
		if err != nil {
			return nil, err
		}
		out[e.Name()] = held
	}
	return out, nil
}

func (a *Apt) record(pkg string, held []string) error {
	return a.files.WriteFile(path.Join(a.store, pkg), []byte(strings.Join(held, "\n")+"\n"), 0o644)
}

// value: the record of a package once an app let go of it: removed when
// no app holds it any longer.
func (a *Apt) let(pkg string, held []string) error {
	if len(held) > 0 {
		return a.record(pkg, held)
	}
	if err := a.files.Remove(path.Join(a.store, pkg)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func checkApp(app string) error {
	if app == "" || strings.ContainsAny(app, "\n\r") {
		return errors.New("apt records a package by an app's name, and " + strconv.Quote(app) + " is none")
	}
	return nil
}

func add(held []string, key string) []string {
	if slices.Contains(held, key) {
		return held
	}
	return append(slices.Clone(held), key)
}

func drop(held []string, key string) []string {
	return slices.DeleteFunc(slices.Clone(held), func(k string) bool { return k == key })
}

func (a *Apt) refresh(ctx context.Context, logs contract.Logs) error {
	return a.root(ctx, logs, slices.Concat(quiet(), []string{"update"})...)
}

func (a *Apt) root(ctx context.Context, logs contract.Logs, argv ...string) error {
	// invariant: a run fails in apt's words, `apt-get install exited 100`,
	// sudo's own last line kept (`a password is required` off a terminal).
	verb := command + " " + argv[len(quiet())]
	if !a.terminal {
		argv = append([]string{"-n"}, argv...)
	}
	c := local.Command(catalog.Sudo, argv...)
	c.Logs = logs
	ran, err := a.run.Attach(ctx, c, local.ToStderr)
	switch {
	case err != nil:
		return fmt.Errorf("%s: %w", catalog.Sudo, err)
	case ran.ExitCode != 0:
		return &local.ExitError{Step: verb, Code: ran.ExitCode, LastStderr: ran.LastStderr}
	}
	return nil
}

// invariant: apt takes an argument opening with `-` as an option, and
// runs as root: every name is checked by the catalog's rule before an
// argv carries it, and every argv ends its options with `--` before them.
func packages(installs []catalog.Install, pinned bool) ([]string, []catalog.AptInstall, error) {
	var names []string
	var asked []catalog.AptInstall
	for _, i := range installs {
		v, ok := i.(catalog.AptInstall)
		if !ok {
			return nil, nil, errors.New(i.Key() + " is not an apt package")
		}
		if len(v.Packages) == 0 {
			return nil, nil, errors.New("an apt install names no package")
		}
		for _, p := range v.Packages {
			if err := catalog.CheckPackage(p); err != nil {
				return nil, nil, err
			}
		}
		asked = append(asked, v)
		at := len(names)
		names = append(names, v.Packages...)
		if pinned && v.Version != "" {
			if err := catalog.CheckPackageVersion(v.Version); err != nil {
				return nil, nil, err
			}
			names[at] += "=" + v.Version
		}
	}
	return names, asked, nil
}

type TakesMoreError struct {
	Asked, Also []string
}

func (e *TakesMoreError) Error() string {
	return "removing " + strings.Join(e.Asked, ", ") + " would remove " + strings.Join(e.Also, ", ") + " too: nothing removed"
}
