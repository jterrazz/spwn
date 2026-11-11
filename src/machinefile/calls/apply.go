package calls

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/apps"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
)

const (
	// value: an apply off a terminal whose plan asks the owner something (a
	// password, a license, a sign-in, a question): it changes nothing.
	NeedsTerminal fail.Code = "machinefile.needs-terminal"
)

type ApplyIn struct {
	// value: the Machinefile, by its name or its path, which this machine
	// is then bound to; empty, the one it is bound to.
	Machinefile string
	// value: also remove what station installed for this machine that its
	// Machinefile lists no longer, and only that.
	Remove      bool
	Interactive bool
	Confirm     string
}

var Apply = call.Call[Deps, ApplyIn, machinefile.Applied]{
	// value: the plan carried out: each item installed by apps.install or
	// set by settings.apply, one at a time in the plan's order.
	Name: "machinefile.apply", Where: call.AtTarget, Applies: machine.Anywhere, Confirm: confirm.Remove,
	Errors: []fail.Code{NotFound, NeedsTerminal, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in ApplyIn, p call.Progress) (machinefile.Applied, error) {
		// invariant: everything is judged before any effect: the
		// Machinefile, the needs a run off a terminal cannot answer, the
		// removal's confirmation. Then the machine is bound, so an
		// interrupted run is resumed by a bare `station apply`.
		plan, err := judged(ctx, d, in.Machinefile)
		if err != nil {
			return machinefile.Applied{}, err
		}
		out := machinefile.Applied{Plan: plan}
		if asked := plan.Interactive(); !in.Interactive && len(asked) > 0 {
			return out, refusedOffTerminal(asked)
		}
		var extra []string
		for _, e := range plan.Extra {
			extra = append(extra, e.Item)
		}
		if in.Remove && len(extra) > 0 {
			if err := confirm.Require(confirm.For(confirm.Remove, d.Kind), strings.Join(extra, ", "), in.Confirm); err != nil {
				return out, err
			}
		}
		if in.Machinefile != "" {
			if err := bind(ctx, d, in.Machinefile, plan.Name); err != nil {
				return out, err
			}
			out.Bound = plan.Name
		}
		r := run{d: d, p: p, in: in, name: plan.Name, ended: map[string]catalog.State{}}
		for i, l := range plan.Lines {
			if ctx.Err() != nil {
				for _, rest := range plan.Lines[i:] {
					out.Done = append(out.Done, machinefile.Done{Line: rest, State: catalog.StateAbsent, Reason: machinefile.NotRun})
				}
				break
			}
			done := r.line(ctx, l)
			r.ended[l.Step.Item] = done.State
			out.Done = append(out.Done, done)
		}
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if in.Remove {
			out.Removed = r.remove(ctx, plan.Extra)
		}
		return out, ctx.Err()
	},
}

func refusedOffTerminal(asked []machinefile.Asked) error {
	items, needs := []string{}, make([]string, 0, len(asked))
	for _, a := range asked {
		needs = append(needs, string(a.Need))
		for _, it := range a.Items {
			if !slices.Contains(items, it) {
				items = append(items, it)
			}
		}
	}
	return fail.New(fail.NeedsTerminal, NeedsTerminal, strings.Join(items, ", "), map[string]string{"needs": strings.Join(needs, ", ")})
}

// invariant: a Machinefile named by its path is copied into the owner's
// folder under its name, so every later read finds it in the user layer;
// the machine is bound to the name, never to the path.
func bind(ctx context.Context, d Deps, asked, name string) error {
	if isPath(asked) {
		_, tree, err := named(ctx, d, asked)
		if err != nil {
			return err
		}
		if err := d.Owned.Keep(ctx, name, tree.Files[0].Body); err != nil {
			return err
		}
	}
	return d.Prefs.Bind(name)
}

type run struct {
	d     Deps
	p     call.Progress
	in    ApplyIn
	name  string
	ended map[string]catalog.State
}

func (r *run) line(ctx context.Context, l machinefile.Line) machinefile.Done {
	// invariant: a line whose step waits on one that failed fails without
	// running, naming it; every other branch goes on.
	done := machinefile.Done{Line: l, State: l.Seen.State, Version: l.Seen.Version, Detail: l.Seen.Detail}
	if !l.Acts() {
		return done
	}
	for _, w := range l.Step.Waits {
		if r.ended[w] == catalog.StateBroken {
			done.State, done.Detail = catalog.StateBroken, "requires "+w+", which failed"
			return done
		}
	}
	if l.Action == machinefile.Set {
		return r.set(ctx, l, done)
	}
	return r.install(ctx, l, done)
}

func (r *run) install(ctx context.Context, l machinefile.Line, done machinefile.Done) machinefile.Done {
	outcomes, err := call.Invoke(ctx, r.d.Calls, machine.Ref{}, appscalls.Install, appscalls.InstallIn{Names: []string{l.Step.Item}, Interactive: r.in.Interactive}, r.p)
	switch {
	case ctx.Err() != nil:
		done.State, done.Reason, done.Detail = catalog.StateBroken, machinefile.Interrupted, ""
		return done
	case err != nil:
		done.State, done.Detail = catalog.StateBroken, err.Error()
		return done
	}
	for _, o := range outcomes {
		fresh := o.Result == catalog.StateOK && o.Detail != apps.AlreadyInstalled
		if fresh {
			if err := r.d.Ledger.Put(ctx, machinefile.Installed{Item: o.Name, Method: string(o.Method), Machinefile: r.name, At: r.d.Now().UTC()}); err != nil {
				done.State, done.Detail = catalog.StateBroken, err.Error()
				return done
			}
		}
		if o.Name != l.Step.Item {
			continue
		}
		done.State, done.Version, done.Detail, done.Log, done.Installed = o.Result, o.Version, o.Detail, o.Log, fresh
		if o.Why == apps.WhyInterrupted {
			done.Reason, done.Detail = machinefile.Interrupted, ""
		}
	}
	return done
}

func (r *run) set(ctx context.Context, l machinefile.Line, done machinefile.Done) machinefile.Done {
	values, secrets, err := r.values(ctx, l.Step.Values)
	if err != nil {
		done.State, done.Detail = catalog.StateBroken, err.Error()
		return done
	}
	if r.p != nil {
		r.p.Step("Setting "+l.Step.Item, "")
	}
	in := settingscalls.ApplyIn{Names: []string{l.Step.Item}, Interactive: r.in.Interactive}
	if values != nil {
		in.Values = map[string]map[string]string{l.Step.Item: values}
	}
	changes, err := call.Invoke(ctx, r.d.Calls, machine.Ref{}, settingscalls.Apply, in, nil)
	switch {
	case ctx.Err() != nil:
		done.State, done.Reason, done.Detail = catalog.StateBroken, machinefile.Interrupted, ""
		return done
	case err != nil:
		done.State, done.Detail = catalog.StateBroken, masked(err.Error(), secrets)
		return done
	}
	for _, c := range changes {
		done.State, done.Detail = c.Result, ""
		var said []string
		for _, s := range c.Steps {
			said = append(said, strings.TrimSuffix(s.Label+": "+s.Detail, ": "))
		}
		done.Detail = masked(strings.Join(said, ", "), secrets)
	}
	return done
}

// invariant: a `vault:` reference is read from the vault only to hand the
// setting its value; the run's words never carry what it names.
func (r *run) values(ctx context.Context, given map[string]catalog.OptionValue) (values, secrets map[string]string, err error) {
	if len(given) == 0 {
		return nil, nil, nil
	}
	values, secrets = map[string]string{}, map[string]string{}
	for k, v := range given {
		ref, ok := strings.CutPrefix(string(v), catalog.VaultPrefix)
		if !ok {
			values[k] = string(v)
			continue
		}
		if r.d.Vault == nil {
			return nil, nil, errors.New(string(v) + " cannot be read: this machine has no vault")
		}
		secret, err := r.d.Vault.Get(ctx, ref)
		if err != nil {
			return nil, nil, errors.New(string(v) + " is not in this machine's vault")
		}
		values[k], secrets[string(secret)] = string(secret), string(v)
	}
	return values, secrets, nil
}

func masked(words string, secrets map[string]string) string {
	for secret, ref := range secrets {
		if secret != "" {
			words = strings.ReplaceAll(words, secret, ref)
		}
	}
	return words
}

// invariant: only what the ledger says station installed is removed, by
// the method that installed it; an extra already gone leaves the ledger.
func (r *run) remove(ctx context.Context, extra []machinefile.Installed) []machinefile.Done {
	var out []machinefile.Done
	for _, e := range extra {
		line := machinefile.Line{Step: machinefile.Step{Item: e.Item, Kind: catalog.KindApp, Method: e.Method}, Seen: machinefile.Seen{Condition: machinefile.Extra}, Action: machinefile.Remove}
		done := machinefile.Done{Line: line, State: catalog.StateOK}
		outcomes, err := call.Invoke(ctx, r.d.Calls, machine.Ref{}, appscalls.Remove, appscalls.RemoveIn{Names: []string{e.Item}, Confirm: confirm.Yes}, nil)
		var refused *fail.Error
		switch {
		case ctx.Err() != nil:
			done.State, done.Reason = catalog.StateBroken, machinefile.Interrupted
			return append(out, done)
		case errors.As(err, &refused) && refused.Code == appscalls.NotInstalled:
			done.Detail = "already gone"
		case err != nil:
			done.State, done.Detail = catalog.StateBroken, err.Error()
		}
		for _, o := range outcomes {
			done.State, done.Detail, done.Log = o.Result, o.Detail, o.Log
		}
		if done.State == catalog.StateOK {
			if err := r.d.Ledger.Delete(ctx, e.Item); err != nil {
				done.State, done.Detail = catalog.StateBroken, err.Error()
			}
		}
		out = append(out, done)
	}
	return out
}
