package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/settings"
)

// value: this machine's ~/.zshrc, read whole and written whole.

type Zshrc interface {
	Read(ctx context.Context) (string, error)
	Write(ctx context.Context, text string) error
}

const zshrcName = "~/.zshrc"

func blockName(id string) string { return zshrcName + " [" + id + "]" }

// invariant: a block is judged against the file and against what the ledger
// says apply last wrote under its id: absent is set; as apply would write it
// now is kept; still exactly what apply last wrote, but written otherwise now
// (a template or an option moved), is outdated and set again; any other text
// (edited by hand, a start with no end, an id twice) is left as it is. The
// blocks of an app the plan carries come first, in the plan's order; then
// each block the ledger holds of an extra of the plan, which goes with it
// (extraBlock). A block of an app that stays, the owner's, another
// Machinefile's or one skipped here, stays with it and is not judged; a
// block the ledger does not hold is never station's to take, and a setting
// writes its own block through its own apply, so it is never judged here.
func withBlocks(ctx context.Context, d Deps, p machinefile.Plan, cat catalog.Catalog) (machinefile.Plan, error) {
	if d.Zshrc == nil {
		return p, nil
	}
	zshrc, err := d.Zshrc.Read(ctx)
	if err != nil {
		return p, err
	}
	written, err := writtenBlocks(ctx, d)
	if err != nil {
		return p, err
	}
	vars := catalog.ShellVars{Home: d.Home, OS: d.Machine.System.OS, Arch: d.Machine.System.Arch}
	judged := map[string]bool{}
	for _, l := range p.Lines {
		if l.Step.Kind != catalog.KindApp || len(l.Step.Spec.Shell) == 0 {
			continue
		}
		it, _ := cat.Item(l.Step.Item)
		for _, b := range l.Step.Spec.Shell {
			judged[b.ID] = true
			if l.Action == machinefile.Leave {
				continue
			}
			v := vars
			v.Options = l.Step.Values
			body, err := it.Item.RenderShell(b, v)
			if err != nil {
				return p, &machinefile.BadValueError{Item: l.Step.Item, Key: "shell", Problem: err.Error(), At: l.Step.Origin.File}
			}
			p.Blocks = append(p.Blocks, judgeBlock(zshrc, l.Step.Item, b.ID, body, written))
		}
	}
	extras := map[string]machinefile.Fate{}
	for _, e := range p.Extra {
		extras[e.Step.Item] = e.Fate()
	}
	for _, id := range settings.PackageBlockIDs(zshrc) {
		w, ok := written[id]
		fate, extra := extras[w.Item]
		if ok && extra && !judged[id] {
			judged[id] = true
			p.Blocks = append(p.Blocks, extraBlock(zshrc, w, fate))
		}
	}
	return p, nil
}

func writtenBlocks(ctx context.Context, d Deps) (map[string]machinefile.Written, error) {
	list, err := d.Ledger.Blocks(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]machinefile.Written, len(list))
	for _, w := range list {
		out[w.ID] = w
	}
	return out, nil
}

// value: a block's diff as apply would change the file: what the file holds
// first (`-`), then what apply writes (`+`).
func blockDiff(have, want, id string) string {
	return settings.UnifiedDiff(have, want, blockName(id), "station's "+blockName(id))
}

// value: whether the file holds exactly what apply last wrote under the id.
func lastWritten(have string, state settings.BlockState, w machinefile.Written, ok bool) bool {
	return ok && state == settings.BlockHere && machinefile.BlockDigest(have) == w.Digest
}

func judgeBlock(zshrc, item, id, body string, written map[string]machinefile.Written) machinefile.Block {
	have, state := settings.PackageBlock(zshrc, id)
	w, recorded := written[id]
	b := machinefile.Block{Item: item, ID: id, Zsh: body}
	switch {
	case state == settings.BlockAbsent:
		b.Condition, b.Action = machinefile.Absent, machinefile.Set
		b.Diff = blockDiff("", body, id)
	case state == settings.BlockHere && have == body:
		b.Condition, b.Action = machinefile.Present, machinefile.Keep
	case lastWritten(have, state, w, recorded):
		b.Condition, b.Action = machinefile.Outdated, machinefile.Set
		b.Diff = blockDiff(have, body, id)
	default:
		b.Condition, b.Action = machinefile.Changed, machinefile.Leave
		b.Diff = blockDiff(have, body, id)
	}
	return b
}

// invariant: a block station wrote for an extra goes with it: taken back
// by `apply --remove` with the app's own removal (taken, or already gone)
// while it is still what apply last wrote; kept with an app that stays,
// and the owner's with an app let go of as the owner's; and the owner's,
// kept, once edited by hand since station wrote it.
func extraBlock(zshrc string, w machinefile.Written, app machinefile.Fate) machinefile.Block {
	have, state := settings.PackageBlock(zshrc, w.ID)
	b := machinefile.Block{Item: w.Item, ID: w.ID, Zsh: have, Condition: machinefile.Extra, Action: machinefile.Leave, Fate: app}
	switch {
	case !lastWritten(have, state, w, true):
		b.Fate = machinefile.KeptEdited
	case app == machinefile.Taken || app == machinefile.Gone:
		b.Action, b.Diff = machinefile.Remove, blockDiff(have, "", w.ID)
	}
	return b
}

// invariant: a block is written once its app reads there, and only while
// the file still holds what the plan judged; it is recorded as written,
// so a later edit by hand reads changed and a later template change reads
// outdated. An extra's block is not judged here: it follows its app in
// `apply --remove` (followApp).
func (r *run) blocks(ctx context.Context, p machinefile.Plan) []machinefile.BlockDone {
	out := make([]machinefile.BlockDone, 0, len(p.Blocks))
	for _, b := range p.Blocks {
		done := machinefile.BlockDone{Block: b, State: catalog.StateOK}
		switch {
		case b.Condition == machinefile.Extra:
			continue
		case b.Action == machinefile.Keep:
		case b.Action == machinefile.Leave:
			done.State, done.Detail = catalog.StateDegraded, "edited by hand: left as it is"
		case !r.there[b.Item]:
			done.State, done.Detail = catalog.StateAbsent, "waits on "+b.Item
		case ctx.Err() != nil:
			done.State = catalog.StateBroken
		default:
			done = r.writeBlock(ctx, done)
		}
		out = append(out, done)
	}
	return out
}

// value: what became of an extra's app in `apply --remove`, which its
// blocks follow: kept for the next run to judge again, taken back (or
// already gone), or let go of as the owner's.

type appEnd int

const (
	appStays appEnd = iota
	appGoes
	appLetGo
)

// invariant: an extra's blocks follow what became of its app in this run:
// with the app taken back or already gone, each block still as apply last
// wrote it is taken; with the app let go of as the owner's, its blocks are
// let go of too, kept in the file; with the app kept for the next run, they
// stay with it, records and all. A block edited by hand since station wrote
// it is let go of as the owner's, whatever became of its app. Once the app
// goes or is let go of, no record of its blocks is left; broke is true when
// one could not be taken or let go of, so the app's own record is kept and
// the next run tries again.
func (r *run) followApp(ctx context.Context, p machinefile.Plan, item string, end appEnd) (out []machinefile.BlockDone, broke bool) {
	for _, b := range p.Blocks {
		if b.Condition != machinefile.Extra || b.Item != item {
			continue
		}
		done := machinefile.BlockDone{Block: b, State: catalog.StateOK}
		switch {
		case b.Fate == machinefile.KeptEdited:
			done = r.letGo(ctx, done, editedKept)
		case end == appLetGo:
			done = r.letGo(ctx, done, "kept with "+item+", and yours")
		case end == appStays || b.Action != machinefile.Remove:
			done.Detail = "kept with " + item + ": judged again next run"
		case ctx.Err() != nil:
			done.State = catalog.StateBroken
		default:
			done = r.writeBlock(ctx, done)
		}
		broke = broke || done.State == catalog.StateBroken
		out = append(out, done)
	}
	if end == appStays || broke {
		return out, broke
	}
	written, err := writtenBlocks(ctx, r.d)
	for _, w := range written {
		if err == nil && w.Item == item {
			err = r.d.Ledger.DeleteBlock(ctx, w.ID)
		}
	}
	if err != nil {
		out = append(out, machinefile.BlockDone{Block: machinefile.Block{Item: item, Condition: machinefile.Extra}, State: catalog.StateBroken, Detail: err.Error()})
	}
	return out, err != nil
}

// value: the words of a block edited by hand since station wrote it, once
// its app is listed no longer: the owner's from then on.
const editedKept = "edited by hand: kept, and yours"

func (r *run) letGo(ctx context.Context, done machinefile.BlockDone, detail string) machinefile.BlockDone {
	done.Detail = detail
	if err := r.d.Ledger.DeleteBlock(ctx, done.ID); err != nil {
		done.State, done.Detail = catalog.StateBroken, err.Error()
	}
	return done
}

func (r *run) writeBlock(ctx context.Context, done machinefile.BlockDone) machinefile.BlockDone {
	zshrc, err := r.d.Zshrc.Read(ctx)
	if err != nil {
		done.State, done.Detail = catalog.StateBroken, err.Error()
		return done
	}
	written, err := writtenBlocks(ctx, r.d)
	if err != nil {
		done.State, done.Detail = catalog.StateBroken, err.Error()
		return done
	}
	var text string
	switch done.Action {
	case machinefile.Set:
		now := judgeBlock(zshrc, done.Item, done.ID, done.Zsh, written).Action
		if now == machinefile.Keep {
			done.Detail = ""
			return done
		}
		if now != machinefile.Set {
			done.State, done.Detail = catalog.StateDegraded, "edited by hand: left as it is"
			return done
		}
		text, _ = settings.UpsertPackageBlock(zshrc, done.ID, done.Zsh)
		done.Detail = "written"
	default:
		have, state := settings.PackageBlock(zshrc, done.ID)
		if w, ok := written[done.ID]; !lastWritten(have, state, w, ok) {
			return r.letGo(ctx, done, editedKept)
		}
		text, _ = settings.RemovePackageBlock(zshrc, done.ID)
		done.Detail = "removed"
	}
	if err = r.d.Zshrc.Write(ctx, text); err != nil {
		done.State, done.Detail = catalog.StateBroken, err.Error()
		return done
	}
	if done.Action == machinefile.Set {
		err = r.d.Ledger.PutBlock(ctx, machinefile.Written{ID: done.ID, Item: done.Item, Digest: machinefile.BlockDigest(done.Zsh), At: r.d.Now().UTC()})
	} else {
		err = r.d.Ledger.DeleteBlock(ctx, done.ID)
	}
	if err != nil {
		done.State, done.Detail = catalog.StateBroken, err.Error()
	}
	return done
}
