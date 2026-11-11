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

func (b *Bun) Outdated(context.Context, []string) (map[string]apps.Currency, error) {
	return nil, nil
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
