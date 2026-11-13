package uv

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const command = "uv"

type Uv struct {
	run    local.Runner
	bundle contract.Bundler
}

func New(run local.Runner, bundle contract.Bundler) *Uv { return &Uv{run: run, bundle: bundle} }

func (u *Uv) Method() catalog.Method { return catalog.MethodUv }

func (u *Uv) Manager() contract.Manager {
	return contract.Manager{Name: command, App: command}
}

func (u *Uv) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true, UpdateAll: true}
}

func (u *Uv) Ready(context.Context) error {
	if _, ok := u.run.LookPath(command); !ok {
		return contract.ErrMissing
	}
	return nil
}

func (u *Uv) Inventory(ctx context.Context) (map[string]string, error) {
	if err := u.Ready(ctx); err != nil {
		return nil, err
	}
	out, err := u.run.Output(ctx, local.Command(command, "tool", "list"))
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errors.New("`uv tool list` exited " + strconv.Itoa(out.ExitCode))
	}
	list := map[string]string{}
	for _, line := range strings.Split(out.Stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "-") {
			continue
		}
		version := ""
		if len(fields) > 1 {
			version = strings.TrimPrefix(fields[1], "v")
		}
		list[fields[0]] = version
	}
	return list, nil
}

func (u *Uv) Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error) {
	// invariant: `uv tool list --outdated` lists every tool and marks the
	// behind ones `[latest: <version>]`; a tool it does not list at all is
	// unread, never current, since an index uv could not reach is answered
	// with an empty list. It names the index's newest version, not what the
	// tool's own constraint allows, so a tool installed at a pinned version
	// reads behind while `uv tool upgrade` leaves it: its pin is the owner's.
	if len(keys) == 0 {
		return map[string]apps.Currency{}, nil
	}
	out, err := u.run.Output(ctx, local.Cmd{Name: command, Args: []string{"tool", "list", "--outdated"}, Timeout: contract.CurrencyBudget})
	if err != nil || out.ExitCode != 0 {
		out.Stdout = ""
	}
	listed := map[string]string{}
	for _, line := range strings.Split(out.Stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "-") {
			continue
		}
		listed[fields[0]] = ""
		if open := strings.Index(line, "[latest: "); open >= 0 {
			listed[fields[0]] = strings.TrimSuffix(strings.TrimSpace(line[open+len("[latest: "):]), "]")
		}
	}
	verdicts := make(map[string]apps.Currency, len(keys))
	for _, k := range keys {
		newest, there := listed[k]
		switch {
		case !there:
			verdicts[k] = apps.Currency{Verdict: apps.Unknown}
		case newest != "":
			verdicts[k] = apps.Currency{Verdict: apps.Behind, Newest: newest}
		default:
			verdicts[k] = apps.Currency{Verdict: apps.Current}
		}
	}
	return verdicts, nil
}

func (u *Uv) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return u.bundle.Bundle(ctx, u.lines(installs), false, logs)
}

func (u *Uv) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return u.bundle.Bundle(ctx, u.lines(installs), true, logs)
}

func (u *Uv) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	for _, i := range installs {
		v, ok := i.(catalog.UvInstall)
		if !ok {
			return errors.New(i.Key() + " is not installed by uv")
		}
		c := local.Command(command, "tool", "uninstall", v.Key())
		c.Logs = logs
		if err := local.Attach(ctx, u.run, c); err != nil {
			return err
		}
	}
	return nil
}

func (u *Uv) lines(installs []catalog.Install) []string {
	out := make([]string, 0, len(installs))
	for _, i := range installs {
		out = append(out, u.Line(i))
	}
	return out
}

func (u *Uv) Line(i catalog.Install) string {
	v, ok := i.(catalog.UvInstall)
	if !ok {
		return ""
	}
	return `uv "` + v.Package + `"`
}
