package brew

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const (
	command = "brew"
	bundle  = command + " bundle"
)

var manager = contract.Manager{Name: command, App: "homebrew", Needs: "homebrew"}

type Brew struct {
	run local.Runner
	env func() []string
}

func New(run local.Runner, env func() []string) *Brew {
	// invariant: env is asked at each bundle, so a manager installed by an
	// earlier step of the same run is on the bundle's PATH.
	return &Brew{run: run, env: env}
}

func (b *Brew) Formulae() contract.Installer {
	return &kind{brew: b, method: catalog.MethodBrew, flag: "--formula", list: local.Command(command, "list", "--formula", "--versions")}
}

func (b *Brew) Casks() contract.Installer {
	return &kind{brew: b, method: catalog.MethodCask, flag: "--cask", list: local.Command(command, "list", "--cask", "--versions")}
}

func (b *Brew) Name() string { return bundle }

func (b *Brew) Bundle(ctx context.Context, lines []string, update bool, logs contract.Logs) error {
	// hazard: brew's own prompts (a cask's sudo) read /dev/tty, not stdin, so
	// feeding the Brewfile on stdin (--file=-) does not steal them.
	mode := "--no-upgrade"
	if update {
		mode = "--upgrade"
	}
	c := local.Command(command, "bundle", "install", mode, "--file=-")
	c.Stdin = strings.Join(lines, "\n") + "\n"
	c.Logs = logs
	if b.env != nil {
		c.Env = b.env()
	}
	ran, err := b.run.Attach(ctx, c, local.ToStderr)
	switch {
	case err != nil:
		return fmt.Errorf("%s: %w", bundle, err)
	case ran.ExitCode != 0:
		failure := bundle + " exited " + strconv.Itoa(ran.ExitCode)
		if ran.LastStderr != "" {
			failure += ": " + ran.LastStderr
		}
		return errors.New(failure)
	}
	return nil
}

type kind struct {
	brew   *Brew
	method catalog.Method
	flag   string
	list   local.Cmd
}

func (k *kind) Method() catalog.Method    { return k.method }
func (k *kind) Manager() contract.Manager { return manager }

func (k *kind) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true, UpdateAll: true}
}

func (k *kind) Ready(context.Context) error {
	if _, ok := k.brew.run.LookPath(command); !ok {
		return contract.ErrMissing
	}
	return nil
}

func (k *kind) Inventory(ctx context.Context) (map[string]string, error) {
	if err := k.Ready(ctx); err != nil {
		return nil, err
	}
	out, err := k.brew.run.Output(ctx, k.list)
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errors.New("brew list exited " + strconv.Itoa(out.ExitCode))
	}
	return parseList(out.Stdout), nil
}

func (k *kind) Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error) {
	behind, err := k.brew.outdated(ctx, k.flag)
	if err != nil {
		return nil, err
	}
	currency := make(map[string]apps.Currency, len(keys))
	for _, key := range keys {
		currency[key] = apps.Currency{Verdict: apps.Current}
		if newest, ok := behind[key]; ok {
			currency[key] = apps.Currency{Verdict: apps.Behind, Newest: newest}
		}
	}
	return currency, nil
}

func (k *kind) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return k.brew.Bundle(ctx, k.lines(installs), false, logs)
}

func (k *kind) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return k.brew.Bundle(ctx, k.lines(installs), true, logs)
}

func (k *kind) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	for _, i := range installs {
		var c local.Cmd
		switch v := i.(type) {
		case catalog.BrewInstall:
			c = local.Command(command, "uninstall", v.Formula)
		case catalog.CaskInstall:
			c = local.Command(command, "uninstall", "--cask", v.Token)
		default:
			return errors.New(i.Key() + " is not installed by brew")
		}
		c.Logs = logs
		if err := local.Attach(ctx, k.brew.run, c); err != nil {
			return err
		}
	}
	return nil
}

func (k *kind) lines(installs []catalog.Install) []string {
	out := make([]string, 0, len(installs))
	for _, i := range installs {
		out = append(out, k.Line(i))
	}
	return out
}

func (k *kind) Line(i catalog.Install) string {
	trusted := func(line string, t bool) string {
		if t {
			return line + ", trusted: true"
		}
		return line
	}
	switch v := i.(type) {
	case catalog.BrewInstall:
		return trusted(`brew "`+v.Formula+`"`, v.Trusted)
	case catalog.CaskInstall:
		return trusted(`cask "`+v.Token+`"`, v.Trusted)
	}
	return ""
}

func parseList(stdout string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		out[fields[0]] = fields[len(fields)-1]
		if len(fields) == 1 {
			out[fields[0]] = ""
		}
	}
	return out
}

func (b *Brew) Manager() string { return command }

func (b *Brew) Behind(ctx context.Context) (int, error) {
	// invariant: how many installed formulae and casks have a newer version;
	// contract.ErrMissing where brew is not installed.
	if _, ok := b.run.LookPath(command); !ok {
		return 0, contract.ErrMissing
	}
	behind, err := b.outdated(ctx)
	return len(behind), err
}

type outdatedJSON struct {
	Formulae []struct {
		Name    string `json:"name"`
		Pinned  bool   `json:"pinned"`
		Current string `json:"current_version"`
	} `json:"formulae"`
	Casks []struct {
		Name    string `json:"name"`
		Current string `json:"current_version"`
	} `json:"casks"`
}

func (b *Brew) outdated(ctx context.Context, flags ...string) (map[string]string, error) {
	// invariant: a pinned formula is never behind: station does not update
	// what its owner pinned.
	out, err := b.run.Output(ctx, local.Command(command, append([]string{"outdated", "--json=v2"}, flags...)...))
	switch {
	case err != nil:
		return nil, err
	case out.ExitCode != 0:
		return nil, errors.New("brew outdated exited " + strconv.Itoa(out.ExitCode))
	}
	var parsed outdatedJSON
	if err := json.Unmarshal([]byte(out.Stdout), &parsed); err != nil {
		return nil, fmt.Errorf("brew outdated: %w", err)
	}
	behind := map[string]string{}
	for _, f := range parsed.Formulae {
		if !f.Pinned {
			behind[lastSegment(f.Name)] = f.Current
		}
	}
	for _, c := range parsed.Casks {
		behind[lastSegment(c.Name)] = c.Current
	}
	return behind, nil
}

func lastSegment(name string) string {
	return name[strings.LastIndex(name, "/")+1:]
}
