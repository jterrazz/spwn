package calls

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

type UpgradeIn struct {
	// value: the Machinefile, by its name; empty, the one this machine is
	// bound to, else the one named after it.
	Machinefile string
	Interactive bool
	// value: the engine's night: what asks the owner something waits for
	// the owner's run instead of refusing the upgrade, and the night is
	// recorded.
	Unattended bool
}

var Upgrade = call.Call[Deps, UpgradeIn, machinefile.Upgraded]{
	// value: every app the Machinefile lists that is there moved as far as
	// its range lets it, by the method that holds it, the managers first, a
	// version a method holds no range around kept; then the owner's
	// run.upgrade lines; then the lock written anew, its moves shown, and
	// each newer version past a range said waiting.
	Name: "machinefile.upgrade", Document: 2, Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, NeedsTerminal, Busy},
	Run: func(ctx context.Context, d Deps, in UpgradeIn, p call.Progress) (machinefile.Upgraded, error) {
		// invariant: every night is recorded, one that failed with its
		// error and the last night that ran to its end beside it, so the
		// owner reads it and the next try comes; the owner's
		// own upgrade run to its end refreshes what the last night of that
		// Machinefile says waits.
		out, err := upgrade(ctx, d, in, p)
		out.Interrupted = out.Interrupted || ctx.Err() != nil
		switch {
		case in.Unattended:
			last, ran, rerr := d.Ledger.Night(context.WithoutCancel(ctx))
			night := machinefile.NightOf(out, err, in.Machinefile, d.Now()).After(last, ran && rerr == nil)
			if werr := d.Ledger.Write(context.WithoutCancel(ctx), night); werr != nil && err == nil {
				err = werr
			}
		case err == nil && !out.Interrupted:
			if n, ran, rerr := d.Ledger.Night(ctx); rerr == nil && ran && n.Machinefile == out.Plan.Name {
				_ = d.Ledger.Write(context.WithoutCancel(ctx), n.Refreshed(out))
			}
		}
		return out, err
	},
}

func upgrade(ctx context.Context, d Deps, in UpgradeIn, p call.Progress) (machinefile.Upgraded, error) {
	// invariant: one run of the machine at a time, judged at the shared
	// source's newest commit it may take (atNewest) before any effect.
	release, err := take(ctx, d)
	if err != nil {
		return machinefile.Upgraded{}, err
	}
	defer release()
	offTerminal := !in.Unattended && !in.Interactive
	plan, unpulled, err := atNewest(
		func() (machinefile.Plan, error) { return judged(ctx, d, in.Machinefile, "") },
		func() (func(), bool, string) { return pull(ctx, d) },
		func(p machinefile.Plan) error {
			if asked := p.UpgradeNeeds(); offTerminal && len(asked) > 0 {
				return refusedOffTerminal(asked, "upgrade")
			}
			return nil
		})
	if err != nil {
		return machinefile.Upgraded{Plan: plan}, err
	}
	out := machinefile.Upgraded{Plan: plan, Unpulled: unpulled}
	if bound, err := d.Prefs.Bound(); err == nil && bound == "" {
		out.Unbound = true
	}
	var waits []string
	if in.Unattended {
		waits = plan.Waits()
	}
	lines := plan.Upgrades()
	for i, l := range lines {
		if ctx.Err() != nil {
			for _, rest := range lines[i:] {
				out.Done = append(out.Done, machinefile.Done{Line: rest, State: rest.Seen.State, Version: rest.Seen.Version, Reason: machinefile.NotRun})
			}
			break
		}
		if slices.Contains(waits, l.Step.Package) {
			out.Done = append(out.Done, machinefile.Done{Line: l, State: l.Seen.State, Method: l.Seen.Method, Version: l.Seen.Version, Reason: machinefile.WaitsForYou})
			continue
		}
		if l.Seen.Condition == machinefile.Absent {
			r := &run{d: d, p: p, in: ApplyIn{Interactive: in.Interactive}, name: plan.Name}
			out.Done = append(out.Done, r.reinstall(ctx, l))
			continue
		}
		out.Done = append(out.Done, update(ctx, d, l, p))
	}
	if s, ok := plan.ScriptOf(machinefile.RunUpgrade); ok && ctx.Err() == nil {
		out.Ran = ran(ctx, d, s, p)
	}
	out.Interrupted = ctx.Err() != nil
	out.Waiting = machinefile.WaitingOf(out.Done)
	now := machinefile.Relocked(plan, out.Done, plan.Source)
	out.Lock = machinefile.Diff(plan.Lock.LockFile, now)
	if !out.Lock.Empty() {
		if err := d.Locks.Write(context.WithoutCancel(ctx), plan.Lock.LockFile, now); err != nil {
			return out, err
		}
	}
	return out, ctx.Err()
}

func atNewest(judge func() (machinefile.Plan, error), pull func() (back func(), moved bool, unpulled string), refuse func(machinefile.Plan) error) (machinefile.Plan, string, error) {
	// invariant: everything judged before any effect: the plan where the
	// shared source's clone stands, and the refusal of what asks the owner
	// something off a terminal; the source then moves forward to its
	// branch's newest commit and the plan is judged again there. A source
	// that cannot move, or whose newest commit does not resolve, is put
	// back where it stood and the run goes on from it, saying why; one
	// whose newest commit asks something off a terminal is put back and the
	// upgrade refused. A clone whose own commit does not resolve has no
	// plan to refuse: the source moves first, its newest commit is judged
	// as the only one, and the run fails only when that one does not
	// resolve either, the clone left on it.
	plan, stood := judge()
	if stood == nil {
		if err := refuse(plan); err != nil {
			return plan, "", err
		}
	}
	back, moved, unpulled := pull()
	if !moved {
		return plan, unpulled, stood
	}
	fresh, err := judge()
	switch {
	case err != nil && stood != nil:
		return machinefile.Plan{}, unpulled, err
	case err != nil:
		return keptBefore(plan, back, err)
	}
	if err := refuse(fresh); err != nil {
		back()
		return fresh, "", err
	}
	return fresh, unpulled, nil
}

// value: the plan where the clone stood, the clone put back there, and
// why its newest commit was not taken, as the run says it.
func keptBefore(plan machinefile.Plan, back func(), unresolved error) (machinefile.Plan, string, error) {
	back()
	return plan, "its newest commit does not resolve, so station keeps the one before: " + unresolved.Error(), nil
}

// value: the machine's turn, taken for one apply or upgrade; a turn
// another run holds is the owner's to wait for, refused in words.
func take(ctx context.Context, d Deps) (func(), error) {
	release, err := d.Turn.Take(ctx)
	if errors.Is(err, contract.ErrBusy) {
		return nil, fail.New(fail.Refused, Busy, "this machine", nil)
	}
	return release, err
}

// value: each source that follows a branch moved forward: how to put
// every one that moved back, whether one moved, and why one did not
// move, in the owner's words (empty when each did).
func pull(ctx context.Context, d Deps) (back func(), moved bool, unpulled string) {
	var undo []func()
	var said []string
	for _, s := range d.Sources() {
		f, ok := s.(contract.Follows)
		if !ok {
			continue
		}
		pulled, err := f.Pull(ctx)
		var refused *contract.NotForwardError
		switch {
		case errors.As(err, &refused):
			said = append(said, refused.Error()+": delete "+refused.Clone+" to take the branch as it is now")
		case err != nil:
			said = append(said, err.Error())
		case pulled.Moved():
			moved = true
			undo = append(undo, func() { _ = f.Back(context.WithoutCancel(ctx), pulled.Was) })
		}
	}
	return func() {
		for _, u := range undo {
			u()
		}
	}, moved, strings.Join(said, "; ")
}

// invariant: an app the lock holds that is not there is installed by the
// method the lock names, at the newest version the Machinefile allows (the
// lock's own version is what could no longer install), and the lock takes
// what that method reads once it ran.
func (r *run) reinstall(ctx context.Context, l machinefile.Line) machinefile.Done {
	held := cmp.Or(l.Step.Locked, l.Step.Recorded)
	fresh := l
	fresh.Step.Locked = ""
	done := r.install(ctx, fresh, machinefile.Done{Line: l, State: l.Seen.State, Version: l.Seen.Version})
	done.Line = l
	if done.Installed {
		done.Updated = true
		done.Detail = strings.TrimSpace("installed anew, the lock held " + held)
	}
	return done
}

// invariant: an update interrupted hands back nothing of what it did, so
// the app is read again past the interrupt, as apply's install is: one
// that reads there at another version than before finished just before
// it, and the lock takes that version; else the line stays interrupted
// and the lock keeps the one it held.
func finishedUpdate(ctx context.Context, d Deps, done machinefile.Done) machinefile.Done {
	done.State, done.Reason = catalog.StateBroken, machinefile.Interrupted
	past, cancel := context.WithTimeout(context.WithoutCancel(ctx), rereadBudget)
	defer cancel()
	now, ok := reread(past, d, done.Line.Step.Package)
	if !ok || (now.Condition != machinefile.Present && now.Condition != machinefile.Outdated) || now.Version == done.Line.Seen.Version {
		return done
	}
	done.State, done.Method, done.Version, done.Reason, done.Updated = catalog.StateOK, now.Method, now.Version, "", true
	return done
}

func update(ctx context.Context, d Deps, l machinefile.Line, p call.Progress) machinefile.Done {
	// invariant: an app whose version the Machinefile pins is kept at it;
	// every other is updated by the method that holds it, and the lock
	// takes the version that method lists once it ran.
	done := machinefile.Done{Line: l, State: l.Seen.State, Method: l.Seen.Method, Version: l.Seen.Version}
	if l.Step.UpdatesSelf {
		done.Reason = machinefile.SelfUpdating
		return done
	}
	if l.Step.Kept() {
		done.Reason = machinefile.Pinned
		return done
	}
	outcomes, err := call.Invoke(ctx, d.Calls, machine.Ref{}, appscalls.Update, appscalls.UpdateIn{Names: []string{l.Step.Package}, Within: l.Step.Within()}, p)
	switch {
	case ctx.Err() != nil:
		return finishedUpdate(ctx, d, done)
	case err != nil:
		var refused *fail.Error
		done.State, done.Detail = catalog.StateBroken, err.Error()
		if errors.As(err, &refused) && refused.Code == appscalls.NotInstalled {
			done.State, done.Detail = catalog.StateAbsent, "not installed: station apply installs it"
		}
		return done
	}
	for _, o := range outcomes {
		if o.Name != l.Step.Package {
			continue
		}
		done.State, done.Detail, done.Log, done.Updated, done.Newest = o.Result, o.Detail, o.Log, true, o.Newest
		if o.Method != "" {
			done.Method, done.Version = string(o.Method), o.Version
		}
	}
	return done
}
