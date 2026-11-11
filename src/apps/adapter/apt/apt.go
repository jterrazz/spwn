package apt

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

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

type Apt struct {
	run local.Runner
	// value: the run has a terminal, where sudo may ask its password; off
	// one, sudo never asks and fails where it would.
	terminal bool
}

func New(run local.Runner, terminal bool) *Apt { return &Apt{run: run, terminal: terminal} }

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
	out, err := a.run.Output(ctx, local.Command(query, "-W", "-f", `${Package}\t${Version}\t${db:Status-Abbrev}\n`))
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
	read, err := a.run.Output(ctx, local.Command(policy, slices.Concat([]string{"policy", "--"}, named)...))
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

func (a *Apt) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	names, err := packages(installs, true)
	if err != nil {
		return err
	}
	if err := a.refresh(ctx, logs); err != nil {
		return err
	}
	return a.root(ctx, logs, slices.Concat(quiet(), []string{"install", "--"}, names)...)
}

func (a *Apt) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	names, err := packages(installs, false)
	if err != nil {
		return err
	}
	if err := a.refresh(ctx, logs); err != nil {
		return err
	}
	return a.root(ctx, logs, slices.Concat(quiet(), []string{"install", "--only-upgrade", "--"}, names)...)
}

func (a *Apt) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	// invariant: apt removes every package that depends on one it
	// removes, so a removal that would take any package it was not asked
	// for is refused before it runs: what station reads it installed is
	// all it takes back.
	names, err := packages(installs, false)
	if err != nil {
		return err
	}
	sim, err := a.run.Output(ctx, local.Command(command, slices.Concat([]string{"-s", "remove"}, keep(), []string{"--"}, names)...))
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
func packages(installs []catalog.Install, pinned bool) ([]string, error) {
	var names []string
	for _, i := range installs {
		v, ok := i.(catalog.AptInstall)
		if !ok {
			return nil, errors.New(i.Key() + " is not an apt package")
		}
		for _, p := range v.Packages {
			if err := catalog.CheckPackage(p); err != nil {
				return nil, err
			}
		}
		at := len(names)
		names = append(names, v.Packages...)
		if pinned && v.Version != "" {
			if err := catalog.CheckPackageVersion(v.Version); err != nil {
				return nil, err
			}
			names[at] += "=" + v.Version
		}
	}
	return names, nil
}

type TakesMoreError struct {
	Asked, Also []string
}

func (e *TakesMoreError) Error() string {
	return "removing " + strings.Join(e.Asked, ", ") + " would remove " + strings.Join(e.Also, ", ") + " too: nothing removed"
}
