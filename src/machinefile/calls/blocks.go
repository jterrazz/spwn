package calls

import (
	"context"
	"strings"

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

// invariant: a block is judged against the file alone, as apply would write
// it now: absent is set, as written is kept, any other text (edited by hand)
// is left as it is. The blocks of an app the plan carries come first, in
// the plan's order; then the blocks of an app it carries no longer, each
// removed while it is still exactly what a method of the app writes, and
// left once it is not. A setting writes its own block through its own
// apply, so it is never judged here.
func withBlocks(ctx context.Context, d Deps, p machinefile.Plan, cat catalog.Catalog) (machinefile.Plan, error) {
	if d.Zshrc == nil {
		return p, nil
	}
	zshrc, err := d.Zshrc.Read(ctx)
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
			p.Blocks = append(p.Blocks, judgeBlock(zshrc, l.Step.Item, b.ID, body))
		}
	}
	for _, id := range settings.PackageBlockIDs(zshrc) {
		if wanted[id] {
			continue
		}
		if b, ok := staleBlock(zshrc, cat, vars, id); ok {
			p.Blocks = append(p.Blocks, b)
		}
	}
	return p, nil
}

// value: a block's diff as apply would change the file: what the file holds
// first (`-`), then what apply writes (`+`).
func blockDiff(want, have, id string) string {
	return settings.UnifiedDiff(have, want, blockName(id), "station's "+blockName(id))
}

func judgeBlock(zshrc, item, id, body string) machinefile.Block {
	have, state := settings.PackageBlock(zshrc, id)
	b := machinefile.Block{Item: item, ID: id, Zsh: body}
	switch {
	case state == settings.BlockAbsent:
		b.Condition, b.Action = machinefile.Absent, machinefile.Set
		b.Diff = blockDiff(body, "", id)
	case state == settings.BlockHere && have == body:
		b.Condition, b.Action = machinefile.Present, machinefile.Keep
	default:
		b.Condition, b.Action = machinefile.Changed, machinefile.Leave
		b.Diff = blockDiff(body, have, id)
	}
	return b
}

// invariant: only the block of a known app is judged, by what a method of
// that app writes under the id with the options' defaults; an id no method
// writes any more is left, never guessed at.
func staleBlock(zshrc string, cat catalog.Catalog, vars catalog.ShellVars, id string) (machinefile.Block, bool) {
	name, _, _ := strings.Cut(strings.TrimPrefix(id, "station:"), "/")
	it, ok := cat.Item(name)
	if !ok || it.Item.Kind != catalog.KindApp {
		return machinefile.Block{}, false
	}
	have, state := settings.PackageBlock(zshrc, id)
	b := machinefile.Block{Item: name, ID: id, Condition: machinefile.Changed, Action: machinefile.Leave}
	for _, m := range it.Item.Methods {
		for _, written := range m.Shell {
			if written.ID != id {
				continue
			}
			body, err := it.Item.RenderShell(written, vars)
			if err == nil && state == settings.BlockHere && body == have {
				b.Zsh, b.Condition, b.Action = body, machinefile.Extra, machinefile.Remove
				b.Diff = blockDiff("", have, id)
				return b, true
			}
		}
	}
	return b, true
}

func (r *run) blocks(ctx context.Context, p machinefile.Plan) []machinefile.BlockDone {
	out := make([]machinefile.BlockDone, 0, len(p.Blocks))
	for _, b := range p.Blocks {
		done := machinefile.BlockDone{Block: b, State: catalog.StateOK}
		switch {
		case b.Action == machinefile.Keep:
		case b.Action == machinefile.Leave:
			done.State, done.Detail = catalog.StateDegraded, "edited by hand: left as it is"
		case b.Action == machinefile.Set && !r.there[b.Item]:
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

func (r *run) writeBlock(ctx context.Context, done machinefile.BlockDone) machinefile.BlockDone {
	zshrc, err := r.d.Zshrc.Read(ctx)
	if err != nil {
		done.State, done.Detail = catalog.StateBroken, err.Error()
		return done
	}
	var (
		text    string
		changed bool
	)
	switch done.Action {
	case machinefile.Set:
		text, changed = settings.UpsertPackageBlock(zshrc, done.ID, done.Zsh)
		done.Detail = "written"
	default:
		if have, state := settings.PackageBlock(zshrc, done.ID); state != settings.BlockHere || have != done.Zsh {
			done.State, done.Detail = catalog.StateDegraded, "edited by hand: left as it is"
			return done
		}
		text, changed = settings.RemovePackageBlock(zshrc, done.ID)
		done.Detail = "removed"
	}
	if changed {
		if err := r.d.Zshrc.Write(ctx, text); err != nil {
			done.State, done.Detail = catalog.StateBroken, err.Error()
		}
	} else {
		done.Detail = ""
	}
	return done
}
