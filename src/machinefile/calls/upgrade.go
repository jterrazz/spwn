package calls

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"

	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
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
	// version a method holds no range around kept; then the lock written
	// anew, its moves shown, and each newer version past a range said
	// waiting.
	Name: "machinefile.upgrade", Where: call.AtTarget, Applies: machine.Anywhere,
	Errors: []fail.Code{NotFound, NeedsTerminal, Busy},
	Run: func(ctx context.Context, d Deps, in UpgradeIn, p call.Progress) (machinefile.Upgraded, error) {
		// invariant: an upgrade is the one run that moves the shared source
		// to its branch's newest commit, before anything is read: the lock
		// it leaves holds the commit read then.
		release, err := take(ctx, d)
		if err != nil {
			return machinefile.Upgraded{}, err
		}
		defer release()
		err = pull(ctx, d)
		if err != nil {
			return machinefile.Upgraded{}, err
		}
		plan, err := judged(ctx, d, in.Machinefile)
		if err != nil {
			return machinefile.Upgraded{}, err
		}
		out := machinefile.Upgraded{Plan: plan}
		var waits []string
		if asked := plan.UpgradeNeeds(); in.Unattended {
			waits = plan.Waits()
		} else if !in.Interactive && len(asked) > 0 {
			return out, refusedOffTerminal(asked, "upgrade")
		}
		lines := plan.Upgrades()
		for i, l := range lines {
			if ctx.Err() != nil {
				for _, rest := range lines[i:] {
					out.Done = append(out.Done, machinefile.Done{Line: rest, State: rest.Seen.State, Version: rest.Seen.Version, Reason: machinefile.NotRun})
				}
				break
			}
			if slices.Contains(waits, l.Step.Item) {
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
		out.Interrupted = ctx.Err() != nil
		out.Waiting = machinefile.WaitingOf(out.Done)
		now := machinefile.Relocked(plan, out.Done, plan.Source)
		out.Lock = machinefile.Diff(plan.Lock.LockFile, now)
		if !out.Lock.Empty() {
			if err := d.Locks.Write(context.WithoutCancel(ctx), plan.Lock.LockFile, now); err != nil {
				return out, err
			}
		}
		if in.Unattended {
			if err := d.Ledger.Write(context.WithoutCancel(ctx), machinefile.NightOf(out, d.Now())); err != nil {
				return out, err
			}
		}
		return out, ctx.Err()
	},
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

func pull(ctx context.Context, d Deps) error {
	for _, s := range d.Sources() {
		if f, ok := s.(contract.Follows); ok {
			if err := f.Pull(ctx); err != nil {
				return Worded(err)
			}
		}
	}
	return nil
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
	now, ok := reread(past, d, done.Line.Step.Item)
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
	outcomes, err := call.Invoke(ctx, d.Calls, machine.Ref{}, appscalls.Update, appscalls.UpdateIn{Names: []string{l.Step.Item}, Within: l.Step.Within()}, p)
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
		if o.Name != l.Step.Item {
			continue
		}
		done.State, done.Detail, done.Log, done.Updated, done.Newest = o.Result, o.Detail, o.Log, true, o.Newest
		if o.Method != "" {
			done.Method, done.Version = string(o.Method), o.Version
		}
	}
	return done
}
