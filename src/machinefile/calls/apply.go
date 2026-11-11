package calls

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
)

const (
	// value: an apply off a terminal whose plan asks the owner something (a
	// password, a license, a sign-in, a question): it changes nothing.
	NeedsTerminal fail.Code = "machinefile.needs-terminal"
	// value: a Machinefile named by its path whose name the owner's folder
	// already holds with other bytes: it is never replaced.
	NameTaken fail.Code = "machinefile.name-taken"
)

// value: how long the read past an interrupt waits for the interrupted
// item's own answer: the owner asked to stop, so it is short, and an
// overrun leaves the item unrecorded, which only keeps it from removal.
const rereadBudget = 15 * time.Second

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
	Errors: []fail.Code{NotFound, NeedsTerminal, NameTaken, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in ApplyIn, p call.Progress) (machinefile.Applied, error) {
		// invariant: everything is judged before any effect: the
		// Machinefile, the needs a run off a terminal cannot answer (with
		// --remove, those of each extra it uninstalls), the removal's
		// confirmation, a copy that would replace the owner's file. Then
		// the machine is bound, so an interrupted run is resumed by a bare
		// `station apply`.
		plan, err := judged(ctx, d, in.Machinefile)
		if err != nil {
			return machinefile.Applied{}, err
		}
		out := machinefile.Applied{Plan: plan}
		if asked := plan.Interactive(in.Remove); !in.Interactive && len(asked) > 0 {
			return out, refusedOffTerminal(asked)
		}
		var extra []string
		for _, e := range plan.Extra {
			if e.Takes() {
				extra = append(extra, e.Step.Item)
			}
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
			if done.Reason == machinefile.NotRun {
				r.ended[l.Step.Item] = catalog.StateBroken
			}
			out.Done = append(out.Done, done)
		}
		if err := ctx.Err(); err != nil {
			out.Interrupted = true
			return out, err
		}
		if in.Remove {
			out.Removed = r.remove(ctx, plan.Extra)
		}
		out.Interrupted = ctx.Err() != nil
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
		var kept *contract.KeptError
		switch err := d.Owned.Keep(ctx, name, tree.Files[0].Body); {
		case errors.As(err, &kept):
			return fail.New(fail.Refused, NameTaken, asked, map[string]string{"name": kept.Name, "path": kept.Path})
		case err != nil:
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
	// invariant: a line whose step waits on one that failed is not run,
	// naming it, and what waits on it is not run either; every other
	// branch goes on.
	done := machinefile.Done{Line: l, State: l.Seen.State, Version: l.Seen.Version, Detail: l.Seen.Detail}
	if !l.Acts() {
		return done
	}
	for _, w := range l.Step.Waits {
		if r.ended[w] == catalog.StateBroken {
			done.Reason, done.Detail = machinefile.NotRun, "requires "+w+", which did not install"
			return done
		}
	}
	if l.Action == machinefile.Set {
		return r.set(ctx, l, done)
	}
	return r.install(ctx, l, done)
}

func (r *run) install(ctx context.Context, l machinefile.Line, done machinefile.Done) machinefile.Done {
	// invariant: what finished is in the ledger before the interrupt is
	// read, so an item installed just before Ctrl-C is station's to take
	// back like any other.
	outcomes, err := call.Invoke(ctx, r.d.Calls, machine.Ref{}, appscalls.Install, appscalls.InstallIn{Names: []string{l.Step.Item}, Interactive: r.in.Interactive}, r.p)
	if err != nil && ctx.Err() == nil {
		done.State, done.Detail = catalog.StateBroken, err.Error()
		return done
	}
	answered := false
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
		answered = o.Why != apps.WhyInterrupted
		done.State, done.Version, done.Detail, done.Log, done.Installed = o.Result, o.Version, o.Detail, o.Log, fresh
		if !answered {
			done.Reason, done.Detail = machinefile.Interrupted, ""
		}
	}
	if ctx.Err() != nil && !answered {
		done = r.finished(ctx, l, done)
	}
	return done
}

// invariant: an install interrupted hands back nothing of what it did, so
// when no outcome of the item came back, or only its interrupt, the apps
// are read again past the interrupt, until the item's own row answers:
// one that reads there by its method (the one apps.install chose here,
// whatever Machinefile key chose the item) finished just before it, and
// is recorded by that method like any other.
func (r *run) finished(ctx context.Context, l machinefile.Line, done machinefile.Done) machinefile.Done {
	done.State, done.Reason, done.Detail = catalog.StateBroken, machinefile.Interrupted, ""
	past, cancel := context.WithTimeout(context.WithoutCancel(ctx), rereadBudget)
	defer cancel()
	now, ok := reread(past, r.d, l.Step.Item)
	if !ok || (now.Condition != machinefile.Present && now.Condition != machinefile.Outdated) {
		return done
	}
	if r.d.Ledger.Put(past, machinefile.Installed{Item: l.Step.Item, Method: now.Method, Machinefile: r.name, At: r.d.Now().UTC()}) != nil {
		return done
	}
	done.State, done.Version, done.Reason, done.Installed = catalog.StateOK, now.Version, "", true
	return done
}

// value: what one app reads in a reading started afresh (the shared one
// dropped, every method listed again), taken at that app's first answer
// without waiting for the others; false when the read failed or did not
// answer.
func reread(ctx context.Context, d Deps, item string) (machinefile.Seen, bool) {
	updates, err := call.Follow(ctx, d.Calls, machine.Ref{}, appscalls.List, appscalls.ListIn{Fresh: true}, nil)
	if err != nil {
		return machinefile.Seen{}, false
	}
	for u := range updates {
		if u.Err != nil {
			return machinefile.Seen{}, false
		}
		for _, report := range u.Value {
			if report.App.Name == item && !report.Pending {
				return appSeen(report), true
			}
		}
	}
	return machinefile.Seen{}, false
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
// the method that installed it, and only while it reads there by that
// method; an extra already gone leaves the ledger alone; one held by
// another method since, or that station cannot uninstall (a script's
// install, the App Store's), is said and kept, and the ledger lets it go,
// so no later run fails on it again. One another app still needs, a
// manager that holds packages of the owner's, one whose manager is
// missing here, and a read that did not answer keep their record for the
// next run. An extra the run did not take back (its
// uninstall failed or was refused) is still there, so what it leans on
// is kept with its record, and so on down.
func (r *run) remove(ctx context.Context, extra []machinefile.ExtraLine) []machinefile.Done {
	var out []machinefile.Done
	var stayed []machinefile.ExtraLine
	for _, e := range extra {
		done := machinefile.Done{Line: e.Line, State: e.Seen.State, Version: e.Seen.Version}
		switch e.Fate() {
		case machinefile.KeptRequired:
			done.Detail = "required by " + strings.Join(e.RequiredBy, ", ") + ": kept, judged again next run"
			out = append(out, done)
			continue
		case machinefile.KeptManagerMissing:
			done.Detail = "its manager " + e.Seen.Lacks + " is missing: kept, judged again next run"
			out = append(out, done)
			continue
		case machinefile.KeptHolding:
			done.Detail = e.Holding() + ": kept, judged again next run"
			out = append(out, done)
			continue
		case machinefile.KeptUnread:
			done.Detail = "reads " + strings.ReplaceAll(string(e.Seen.Condition), "-", " ") + ": kept, judged again next run"
			out = append(out, done)
			continue
		case machinefile.KeptIrreversible:
			done.Detail = machinefile.CannotUninstall
		case machinefile.KeptAnotherWay:
			done.Detail = machinefile.AnotherWay
		case machinefile.Gone:
			done.State, done.Detail = catalog.StateAbsent, "already gone"
		case machinefile.Taken:
			if needers := leaningOn(stayed, e.Step.Item); len(needers) > 0 {
				done.Detail = "required by " + strings.Join(needers, ", ") + ": kept, judged again next run"
				stayed = append(stayed, e)
				out = append(out, done)
				continue
			}
			if ctx.Err() != nil {
				done.State, done.Reason = catalog.StateBroken, machinefile.Interrupted
				return append(out, done)
			}
			done = r.uninstall(ctx, e, done)
			if done.State == catalog.StateBroken || done.Line.Action == machinefile.Leave {
				stayed = append(stayed, e)
			}
			if done.Reason == machinefile.Interrupted || done.State == catalog.StateBroken {
				out = append(out, done)
				if done.Reason == machinefile.Interrupted {
					return out
				}
				continue
			}
		}
		if err := r.d.Ledger.Delete(ctx, e.Step.Item); err != nil {
			done.State, done.Detail = catalog.StateBroken, err.Error()
		}
		out = append(out, done)
	}
	return out
}

func leaningOn(stayed []machinefile.ExtraLine, item string) []string {
	var out []string
	for _, s := range stayed {
		if slices.Contains(s.Step.Waits, item) {
			out = append(out, s.Step.Item)
		}
	}
	return out
}

func (r *run) uninstall(ctx context.Context, e machinefile.ExtraLine, done machinefile.Done) machinefile.Done {
	outcomes, err := call.Invoke(ctx, r.d.Calls, machine.Ref{}, appscalls.Remove, appscalls.RemoveIn{Names: []string{e.Step.Item}, Confirm: confirm.Yes}, nil)
	var (
		refused      *fail.Error
		irreversible *apps.NotReversibleError
		fromAppStore *apps.AppStoreError
	)
	switch {
	case ctx.Err() != nil:
		done.State, done.Reason = catalog.StateBroken, machinefile.Interrupted
		return done
	case errors.As(err, &irreversible), errors.As(err, &fromAppStore):
		done.Line.Action, done.Detail = machinefile.Leave, machinefile.CannotUninstall
		return done
	case errors.As(err, &refused) && refused.Code == appscalls.NotInstalled:
		done.State, done.Detail = catalog.StateAbsent, "already gone"
		return done
	case err != nil:
		done.State, done.Detail = catalog.StateBroken, err.Error()
		return done
	}
	for _, o := range outcomes {
		done.State, done.Version, done.Detail, done.Log = o.Result, "", o.Detail, o.Log
	}
	return done
}
