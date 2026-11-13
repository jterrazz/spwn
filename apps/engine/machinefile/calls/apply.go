package calls

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
)

const (
	// value: an apply off a terminal whose plan asks the owner something (a
	// password, a license, a sign-in, a question): it changes nothing.
	NeedsTerminal fail.Code = "machinefile.needs-terminal"
	// value: a Machinefile named by its path whose name the owner's folder
	// already holds with other bytes: it is never replaced.
	NameTaken fail.Code = "machinefile.name-taken"
	// value: a phase that is none of the Machinefile's, or --remove asked
	// of a phase that carries only a part of it.
	BadPhase fail.Code = "machinefile.bad-phase"
)

// value: how long the read past an interrupt waits for the interrupted
// package's own answer: the owner asked to stop, so it is short, and an
// overrun leaves the package unrecorded, which only keeps it from removal.
const rereadBudget = 15 * time.Second

type ApplyIn struct {
	// value: the Machinefile, by its name or its path, which this machine
	// is then bound to; empty, the one it is bound to.
	Machinefile string
	// value: also remove what station installed for this machine that its
	// Machinefile lists no longer, and only that.
	Remove bool
	// value: the phase carried out: empty, apply's; image, what the VM
	// image bakes (its build runs it); boot, a VM's first start.
	Phase       catalog.Phase
	Interactive bool
	Confirm     string
}

var Apply = call.Call[Deps, ApplyIn, machinefile.Applied]{
	// value: the plan carried out: each package installed by apps.install,
	// by the method and at the version its step chose, or set by
	// settings.apply, one at a time in the plan's order; then the owner's
	// lines of its phase.
	Name: "machinefile.apply", Document: 2, Where: call.AtTarget, Applies: machine.Anywhere, Confirm: confirm.Remove,
	Errors: []fail.Code{NotFound, NeedsTerminal, NameTaken, BadPhase, confirm.YesNeeded, Busy},
	Run: func(ctx context.Context, d Deps, in ApplyIn, p call.Progress) (answer machinefile.Applied, err error) {
		defer func() { answer = answer.Answered() }()
		if bad := badPhase(in.Phase, in.Remove); bad != nil {
			return machinefile.Applied{}, bad
		}
		// invariant: everything is judged before any effect: the
		// Machinefile, the needs a run off a terminal cannot answer (with
		// --remove, those of each extra it uninstalls), the removal's
		// confirmation, a copy that would replace the owner's file. Then
		// the machine is bound, so an interrupted run is resumed by a bare
		// `station apply`. One run of the machine at a time: an upgrade or
		// the night's holding it is waited for, a while. An apply run to its
		// end reads the last night's waiting versions against the ranges it
		// applied, so a range moved past one clears it at once.
		release, err := take(ctx, d)
		if err != nil {
			return machinefile.Applied{}, err
		}
		defer release()
		plan, err := judged(ctx, d, in.Machinefile, in.Phase)
		if err != nil {
			return machinefile.Applied{}, err
		}
		out := machinefile.Applied{Plan: plan}
		if asked := plan.Interactive(in.Remove); !in.Interactive && len(asked) > 0 {
			return out, refusedOffTerminal(asked, "apply")
		}
		var extra []string
		for _, e := range plan.Extra {
			if e.Takes() {
				extra = append(extra, e.Step.Package)
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
		r := run{d: d, p: p, in: in, name: plan.Name, ended: map[string]catalog.State{}, there: map[string]bool{}}
		out.Done = r.carry(ctx, plan.Lines)
		if err := ctx.Err(); err != nil {
			out.Interrupted = true
			return out, errors.Join(err, relock(ctx, d, plan, out.Done))
		}
		out.Blocks = r.blocks(ctx, plan)
		if in.Remove {
			removed, blocks := r.remove(ctx, plan)
			out.Removed, out.Blocks = removed, append(out.Blocks, blocks...)
		}
		if s, ok := plan.Script(); ok && ctx.Err() == nil {
			out.Ran = ran(ctx, d, s, p)
		}
		out.Interrupted = ctx.Err() != nil
		if err := relock(ctx, d, plan, out.Done); err != nil {
			return out, err
		}
		if n, ran, err := d.Ledger.Night(ctx); err == nil && ran && n.Machinefile == plan.Name {
			_ = d.Ledger.Write(context.WithoutCancel(ctx), n.Reread(plan))
		}
		return out, ctx.Err()
	},
}

// invariant: the owner's lines of a phase run once its packages are carried
// out, as this machine's user from its home, and stop at the first line
// that fails: their run ends broken, with the last words they said
// (saidLines), and what was applied stays applied.
func ran(ctx context.Context, d Deps, s machinefile.Script, p call.Progress) *machinefile.Ran {
	task := "run." + string(s.Phase)
	p.Step(task, screened(strings.Join(machinefile.ScriptLines(s.Lines), "\n")))
	out := &machinefile.Ran{Script: s, State: catalog.StateOK}
	if d.Shell == nil {
		out.State, out.Detail = catalog.StateBroken, "this machine runs no shell"
		p.Done(task, errors.New(out.Detail))
		return out
	}
	got, err := d.Shell.Run(ctx, s.Lines)
	switch {
	case err != nil:
		out.State, out.Detail = catalog.StateBroken, err.Error()
	case got.ExitCode != 0:
		out.State, out.Detail = catalog.StateBroken, "exited "+strconv.Itoa(got.ExitCode)
		if said := lastLines(got.Stdout+got.Stderr, saidLines); said != "" {
			out.Detail += ": " + said
		}
	}
	if out.State == catalog.StateBroken {
		p.Done(task, errors.New(out.Detail))
	} else {
		p.Done(task, nil)
	}
	return out
}

// value: how many of the last lines a failed run said its step shows.
const saidLines = 3

func lastLines(text string, n int) string {
	lines := machinefile.ScriptLines(text)
	lines = lines[max(0, len(lines)-n):]
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return strings.Join(lines, "; ")
}

func badPhase(phase catalog.Phase, remove bool) error {
	switch {
	case phase != "" && !slices.Contains(phase.Enum(), string(phase)):
		return fail.New(fail.Refused, BadPhase, string(phase), map[string]string{"phases": strings.Join(phase.Enum(), ", ")})
	case remove && phase != "" && phase != catalog.PhaseApply:
		return fail.New(fail.Refused, BadPhase, string(phase), map[string]string{"remove": "true"})
	}
	return nil
}

func refusedOffTerminal(asked []machinefile.Asked, verb string) error {
	pkgs, needs := []string{}, make([]string, 0, len(asked))
	for _, a := range asked {
		needs = append(needs, string(a.Need))
		for _, it := range a.Packages {
			if !slices.Contains(pkgs, it) {
				pkgs = append(pkgs, it)
			}
		}
	}
	detail := map[string]string{"needs": strings.Join(needs, ", ")}
	if verb != "apply" {
		detail["verb"] = verb
	}
	return fail.New(fail.NeedsTerminal, NeedsTerminal, strings.Join(pkgs, ", "), detail)
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
	return d.Prefs.Bind(name, d.Now())
}

type run struct {
	d     Deps
	p     call.Progress
	in    ApplyIn
	name  string
	ended map[string]catalog.State
	// value: the apps that read there once their line ended, whose blocks
	// may be written.
	there map[string]bool
}

// invariant: the packages are carried out one at a time in the plan's
// order; a phase run inside a VM (image, boot) gives them their own share
// of its bound (catalog.ItemsBound), so the owner's lines after them keep
// theirs: a package still running at the end of the share ends there, and
// those after it are not run.
func (r *run) carry(ctx context.Context, lines []machinefile.Line) []machinefile.Done {
	if r.in.Phase == catalog.PhaseImage || r.in.Phase == catalog.PhaseBoot {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, catalog.PackagesBound)
		defer cancel()
	}
	out := make([]machinefile.Done, 0, len(lines))
	for i, l := range lines {
		if ctx.Err() != nil {
			for _, rest := range lines[i:] {
				out = append(out, machinefile.Done{Line: rest, State: catalog.StateAbsent, Reason: machinefile.NotRun})
			}
			break
		}
		done := r.line(ctx, l)
		r.ended[l.Step.Package] = done.State
		r.there[l.Step.Package] = l.Step.Kind == catalog.KindApp && done.Reason == "" && (done.State == catalog.StateOK || (l.Action == machinefile.Adopt && done.State == catalog.StateDegraded))
		if done.Reason == machinefile.NotRun {
			r.ended[l.Step.Package] = catalog.StateBroken
		}
		out = append(out, done)
	}
	return out
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
	return r.wayOut(l, r.install(ctx, l, done))
}

// invariant: an install at a locked version that failed names the way
// out, which the lock itself cannot: `station upgrade <app>` installs the
// newest the Machinefile allows and relocks it, and the lock's line may be
// edited by hand.
func (r *run) wayOut(l machinefile.Line, done machinefile.Done) machinefile.Done {
	if l.Step.Locked == "" || done.State != catalog.StateBroken || done.Reason != "" {
		return done
	}
	hint := "locked at " + l.Step.Locked + " in " + l.Step.Lock + ": station upgrade installs the newest the Machinefile allows, or edit that line"
	if done.Detail != "" {
		hint = done.Detail + "; " + hint
	}
	done.Detail = hint
	return done
}

func (r *run) install(ctx context.Context, l machinefile.Line, done machinefile.Done) machinefile.Done {
	// invariant: what finished is in the ledger before the interrupt is
	// read, so a package installed just before Ctrl-C is station's to take
	// back like any other.
	outcomes, err := call.Invoke(ctx, r.d.Calls, machine.Ref{}, appscalls.Install, appscalls.InstallIn{Names: []string{l.Step.Package}, Interactive: r.in.Interactive, Method: l.Step.Method, Version: l.Step.InstallVersion()}, r.p)
	if err != nil && ctx.Err() == nil {
		done.State, done.Detail = catalog.StateBroken, err.Error()
		return done
	}
	answered := false
	for _, o := range outcomes {
		fresh := o.Result == catalog.StateOK && o.Detail != apps.AlreadyInstalled
		if fresh {
			if err := r.d.Ledger.Put(ctx, machinefile.Installed{Package: o.Name, Method: string(o.Method), Machinefile: r.name, At: r.d.Now().UTC()}); err != nil {
				done.State, done.Detail = catalog.StateBroken, err.Error()
				return done
			}
		}
		if o.Name != l.Step.Package {
			continue
		}
		answered = o.Why != apps.WhyInterrupted
		done.State, done.Method, done.Version, done.Detail, done.Log, done.Installed = o.Result, string(o.Method), o.Version, o.Detail, o.Log, fresh
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
// when no outcome of the package came back, or only its interrupt, the apps
// are read again past the interrupt, until the package's own row answers:
// one that reads there finished just before it, and is recorded by the
// method that holds it, the one the plan chose and apps.install ran.
func (r *run) finished(ctx context.Context, l machinefile.Line, done machinefile.Done) machinefile.Done {
	done.State, done.Reason, done.Detail = catalog.StateBroken, machinefile.Interrupted, ""
	past, cancel := context.WithTimeout(context.WithoutCancel(ctx), rereadBudget)
	defer cancel()
	now, ok := reread(past, r.d, l.Step.Package)
	if !ok || (now.Condition != machinefile.Present && now.Condition != machinefile.Outdated) {
		return done
	}
	if r.d.Ledger.Put(past, machinefile.Installed{Package: l.Step.Package, Method: now.Method, Machinefile: r.name, At: r.d.Now().UTC()}) != nil {
		return done
	}
	done.State, done.Method, done.Version, done.Reason, done.Installed = catalog.StateOK, now.Method, now.Version, "", true
	return done
}

// value: what one app reads in a reading started afresh (the shared one
// dropped, every method listed again), taken at that app's first answer
// without waiting for the others; false when the read failed or did not
// answer.
func reread(ctx context.Context, d Deps, pkg string) (machinefile.Seen, bool) {
	updates, err := call.Follow(ctx, d.Calls, machine.Ref{}, appscalls.List, appscalls.ListIn{Fresh: true}, nil)
	if err != nil {
		return machinefile.Seen{}, false
	}
	for u := range updates {
		if u.Err != nil {
			return machinefile.Seen{}, false
		}
		for _, report := range u.Value {
			if report.App.Name == pkg && !report.Pending {
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
		r.p.Step("Setting "+l.Step.Package, "")
	}
	in := settingscalls.ApplyIn{Names: []string{l.Step.Package}, Interactive: r.in.Interactive}
	if values != nil {
		in.Values = map[string]map[string]string{l.Step.Package: values}
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
		plain, err := r.d.Vault.Get(ctx, ref)
		if err != nil {
			return nil, nil, errors.New(string(v) + " is not in this machine's vault")
		}
		values[k], secrets[string(plain)] = string(plain), string(v)
	}
	return values, secrets, nil
}

func screened(text string) string {
	kept, _ := secret.Screen(text)
	return kept
}

func masked(words string, secrets map[string]string) string {
	for plain, ref := range secrets {
		if plain != "" {
			words = strings.ReplaceAll(words, plain, ref)
		}
	}
	return words
}

// invariant: only what the ledger says station installed is removed, by
// the method that installed it, and only while it reads there by that
// method; an extra already gone leaves the ledger alone; one held by
// another method since, or that station cannot uninstall (a script's
// install, the App Store's, an apt app it recorded adding nothing of), is
// said and kept, and the ledger lets it go, so no later run fails on it
// again. One another app still needs, a
// manager that holds packages of the owner's, one whose manager is
// missing here, and a read that did not answer keep their record for the
// next run. An extra the run did not take back (its
// uninstall failed or was refused) is still there, so what it leans on
// is kept with its record, and so on down. Each extra's blocks follow it
// (followApp), and an extra whose block could not follow keeps its record.
func (r *run) remove(ctx context.Context, p machinefile.Plan) ([]machinefile.Done, []machinefile.BlockDone) {
	var out []machinefile.Done
	var blocks []machinefile.BlockDone
	var stayed []machinefile.ExtraLine
	follow := func(e machinefile.ExtraLine, end appEnd) bool {
		done, broke := r.followApp(ctx, p, e.Step.Package, end)
		blocks = append(blocks, done...)
		return broke
	}
	for _, e := range p.Extra {
		done := machinefile.Done{Line: e.Line, State: e.Seen.State, Version: e.Seen.Version}
		end := appGoes
		switch e.Fate() {
		case machinefile.KeptRequired:
			done.Detail = "required by " + strings.Join(e.RequiredBy, ", ") + ": kept, judged again next run"
			out = append(out, done)
			follow(e, appStays)
			continue
		case machinefile.KeptManagerMissing:
			done.Detail = "its manager " + e.Seen.Lacks + " is missing: kept, judged again next run"
			out = append(out, done)
			follow(e, appStays)
			continue
		case machinefile.KeptHolding:
			done.Detail = e.Holding() + ": kept, judged again next run"
			out = append(out, done)
			follow(e, appStays)
			continue
		case machinefile.KeptUnread:
			done.Detail = "reads " + strings.ReplaceAll(string(e.Seen.Condition), "-", " ") + ": kept, judged again next run"
			out = append(out, done)
			follow(e, appStays)
			continue
		case machinefile.KeptIrreversible:
			done.Detail, end = machinefile.CannotUninstall, appLetGo
		case machinefile.KeptAnotherWay, machinefile.KeptEdited:
			done.Detail, end = machinefile.AnotherWay, appLetGo
		case machinefile.Gone:
			done.State, done.Detail = catalog.StateAbsent, "already gone"
		case machinefile.Taken:
			if needers := leaningOn(stayed, e.Step.Package); len(needers) > 0 {
				done.Detail = "required by " + strings.Join(needers, ", ") + ": kept, judged again next run"
				stayed = append(stayed, e)
				out = append(out, done)
				follow(e, appStays)
				continue
			}
			if ctx.Err() != nil {
				done.State, done.Reason = catalog.StateBroken, machinefile.Interrupted
				return append(out, done), blocks
			}
			done = r.uninstall(ctx, e, done)
			if done.State == catalog.StateBroken || done.Line.Action == machinefile.Leave {
				stayed = append(stayed, e)
			}
			if done.Reason == machinefile.Interrupted || done.State == catalog.StateBroken {
				out = append(out, done)
				if done.Reason == machinefile.Interrupted {
					return out, blocks
				}
				follow(e, appStays)
				continue
			}
			if done.Line.Action == machinefile.Leave {
				end = appLetGo
			}
		}
		if follow(e, end) {
			out = append(out, done)
			continue
		}
		if err := r.d.Ledger.Delete(ctx, e.Step.Package); err != nil {
			done.State, done.Detail = catalog.StateBroken, err.Error()
		}
		out = append(out, done)
	}
	return out, blocks
}

func leaningOn(stayed []machinefile.ExtraLine, pkg string) []string {
	var out []string
	for _, s := range stayed {
		if slices.Contains(s.Step.Waits, pkg) {
			out = append(out, s.Step.Package)
		}
	}
	return out
}

func (r *run) uninstall(ctx context.Context, e machinefile.ExtraLine, done machinefile.Done) machinefile.Done {
	outcomes, err := call.Invoke(ctx, r.d.Calls, machine.Ref{}, appscalls.Remove, appscalls.RemoveIn{Names: []string{e.Step.Package}, Confirm: confirm.Yes}, nil)
	var (
		refused      *fail.Error
		irreversible *apps.NotReversibleError
		fromAppStore *apps.AppStoreError
		unrecorded   *apps.UnrecordedError
	)
	switch {
	case ctx.Err() != nil:
		done.State, done.Reason = catalog.StateBroken, machinefile.Interrupted
		return done
	case errors.As(err, &irreversible), errors.As(err, &fromAppStore), errors.As(err, &unrecorded):
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

// invariant: the lock is written once the run ends, interrupted or not,
// and only when it changed, so a run that changed nothing leaves the
// file as it was.
func relock(ctx context.Context, d Deps, p machinefile.Plan, done []machinefile.Done) error {
	now := machinefile.Relocked(p, done, p.Source)
	if machinefile.Diff(p.Lock.LockFile, now).Empty() {
		return nil
	}
	return d.Locks.Write(context.WithoutCancel(ctx), p.Lock.LockFile, now)
}
