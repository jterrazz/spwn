package bun

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const command = "bun"

type Bun struct{ run local.Runner }

func New(run local.Runner) *Bun { return &Bun{run: run} }

func (b *Bun) Method() catalog.Method    { return catalog.MethodBun }
func (b *Bun) Manager() contract.Manager { return contract.Manager{Name: command, App: command} }

func (b *Bun) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true, UpdateAll: true}
}

func (b *Bun) Ready(context.Context) error {
	if _, ok := b.run.LookPath(command); !ok {
		return contract.ErrMissing
	}
	return nil
}

func (b *Bun) Inventory(ctx context.Context) (map[string]string, error) {
	if err := b.Ready(ctx); err != nil {
		return nil, err
	}
	out, err := b.run.Output(ctx, local.Command(command, "pm", "ls", "-g"))
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errors.New("bun pm ls exited " + strconv.Itoa(out.ExitCode))
	}
	return parseList(out.Stdout), nil
}

func (b *Bun) Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error) {
	// invariant: a package is behind when `bun update -g` would move it: its
	// Update column is another version than the one installed. Its Latest
	// column is not read: a package installed at an exact version is held
	// there by its own range, and an update never moves it. A bun that does
	// not answer, or has no `outdated`, leaves every key unknown.
	if len(keys) == 0 {
		return map[string]apps.Currency{}, nil
	}
	out, err := b.run.Output(ctx, local.Cmd{Name: command, Args: []string{"outdated", "-g"}, Timeout: contract.CurrencyBudget})
	if err == nil && out.ExitCode == 0 {
		return apps.Verdicts(keys, parseOutdated(out.Stdout)), nil
	}
	return apps.Unread(keys), nil
}

func parseOutdated(stdout string) map[string]string {
	// outside: bun prints a table, `| Package | Current | Update | Latest |`,
	// one row a package, and nothing but its banner when none is behind.
	behind := map[string]string{}
	for _, line := range strings.Split(stdout, "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "| "), "|")
		if len(cells) != 4 {
			continue
		}
		name, current, update := strings.TrimSpace(cells[0]), strings.TrimSpace(cells[1]), strings.TrimSpace(cells[2])
		if name == "Package" || strings.Trim(name, "-") == "" || update == "" || update == current {
			continue
		}
		behind[name] = update
	}
	return behind
}

func (b *Bun) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return b.each(ctx, installs, logs, func(v catalog.BunInstall) local.Cmd { return local.Command(command, "install", "-g", v.Package) })
}

func (b *Bun) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return b.each(ctx, installs, logs, func(v catalog.BunInstall) local.Cmd { return local.Command(command, "update", "-g", v.Key()) })
}

func (b *Bun) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return b.each(ctx, installs, logs, func(v catalog.BunInstall) local.Cmd { return local.Command(command, "remove", "-g", v.Key()) })
}

func (b *Bun) each(ctx context.Context, installs []catalog.Install, logs contract.Logs, argv func(catalog.BunInstall) local.Cmd) error {
	for _, i := range installs {
		v, ok := i.(catalog.BunInstall)
		if !ok {
			return errors.New(i.Key() + " is not a bun package")
		}
		c := argv(v)
		c.Logs = logs
		if err := local.Attach(ctx, b.run, c); err != nil {
			return err
		}
	}
	return nil
}

func parseList(stdout string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(stdout, "\n") {
		entry := strings.TrimLeftFunc(line, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '@' })
		if entry == line || entry == "" {
			continue
		}
		at := strings.LastIndex(entry, "@")
		if at <= 0 {
			out[entry] = ""
			continue
		}
		out[entry[:at]] = entry[at+1:]
	}
	return out
}
