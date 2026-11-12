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
// each block the ledger holds of an app the plan carries no longer, which
// `apply --remove` takes back while it is still what apply wrote. A block
// the ledger does not hold is never station's to take, and a setting
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
	wanted := map[string]bool{}
	for _, l := range p.Lines {
		if l.Step.Kind != catalog.KindApp || len(l.Step.Spec.Shell) == 0 {
			continue
		}
		it, _ := cat.Item(l.Step.Item)
		for _, b := range l.Step.Spec.Shell {
			wanted[b.ID] = true
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
	for _, id := range settings.PackageBlockIDs(zshrc) {
		if w, ok := written[id]; ok && !wanted[id] {
			p.Blocks = append(p.Blocks, staleBlock(zshrc, w))
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

func staleBlock(zshrc string, w machinefile.Written) machinefile.Block {
	have, state := settings.PackageBlock(zshrc, w.ID)
	b := machinefile.Block{Item: w.Item, ID: w.ID, Condition: machinefile.Changed, Action: machinefile.Leave}
	if lastWritten(have, state, w, true) {
		b.Zsh, b.Condition, b.Action = have, machinefile.Extra, machinefile.Remove
		b.Diff = blockDiff(have, "", w.ID)
	}
	return b
}

// invariant: a block is written once its app reads there, and only while
// the file still holds what the plan judged; it is recorded as written,
// so a later edit by hand reads changed and a later template change reads
// outdated. A stale block is not taken here: it goes with `apply --remove`,
// after its app (removedBlocks).
func (r *run) blocks(ctx context.Context, p machinefile.Plan) []machinefile.BlockDone {
	out := make([]machinefile.BlockDone, 0, len(p.Blocks))
	for _, b := range p.Blocks {
		done := machinefile.BlockDone{Block: b, State: catalog.StateOK}
		switch {
		case b.Action == machinefile.Remove:
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

// invariant: `apply --remove` takes back a stale block with its app: while
// the app's removal is judged again next run (required, held, unread,
// failed), its block stays with it; once the app is taken, gone, or let go
// of as the owner's, the block station wrote goes.
func (r *run) removedBlocks(ctx context.Context, p machinefile.Plan) []machinefile.BlockDone {
	var out []machinefile.BlockDone
	for _, b := range p.Blocks {
		if b.Action != machinefile.Remove {
			continue
		}
		done := machinefile.BlockDone{Block: b, State: catalog.StateOK}
		switch {
		case r.stays[b.Item]:
			done.Detail = "kept with " + b.Item + ": judged again next run"
		case ctx.Err() != nil:
			done.State = catalog.StateBroken
		default:
			done = r.writeBlock(ctx, done)
		}
		out = append(out, done)
	}
	return out
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
		if w, ok := written[done.ID]; !ok || staleBlock(zshrc, w).Action != machinefile.Remove {
			done.State, done.Detail = catalog.StateDegraded, "edited by hand: left as it is"
			return done
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
