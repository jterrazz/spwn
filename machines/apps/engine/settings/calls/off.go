package calls

import (
	"context"
	"errors"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type OffIn struct {
	Names   []string
	Confirm string
}

var Off = call.Call[Deps, OffIn, []settings.Change]{
	Name: "settings.off", Section: section.Settings, Where: call.AtTarget, Applies: settings.Applies, Confirm: confirm.OffHere,
	Errors: []fail.Code{Unknown, NotForThisMachine, NeedsAdmin, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in OffIn, _ call.Progress) (answer []settings.Change, err error) {
		defer func() { answer = settings.Answered(answer) }()
		// invariant: off never destroys a change of the owner's: only a
		// setting still exactly as station applied it is undone, or one whose
		// only drift is in what holds it (Loose); any other is left as it is,
		// and every one is Off here after. A checked
		// setting Off here is only no longer checked here.
		var pkgs []catalog.Setting
		var checked []string
		for _, name := range in.Names {
			if c, ok := settings.CheckedNamed(name); ok {
				if !d.Settings.Lives(catalog.Setting{Name: c.Name, Kinds: c.Kinds}) {
					return nil, fail.New(fail.NotForThisMachine, NotForThisMachine, name, map[string]string{"why": settings.LivesOnReason(c.Kinds)})
				}
				checked = append(checked, name)
				continue
			}
			it, err := find(d, name, true)
			if err != nil {
				return nil, err
			}
			if err := admitted(d, it, "off"); err != nil {
				return nil, err
			}
			pkgs = append(pkgs, it)
		}
		if err := confirm.Require(confirm.For(confirm.OffHere, d.Kind), strings.Join(in.Names, ", "), in.Confirm); err != nil {
			return nil, err
		}
		off := offThisMac(d)
		changes := make([]settings.Change, 0, len(pkgs)+len(checked))
		for _, name := range checked {
			off[name] = true
			changes = append(changes, settings.Change{Package: name, Verb: "off", Result: catalog.StateOK,
				Steps: []settings.ChangeStep{{Label: "off here", State: catalog.StateOK, Detail: "not checked on this machine"}}})
		}
		for _, it := range pkgs {
			row, now := judge(ctx, d, it, map[string]bool{})
			change := settings.Change{Package: it.Name, Verb: "off", Result: catalog.StateOK}
			if row.State == settings.StateApplied || row.State == settings.StateChangedSince && now.Loose {
				if a, ok := d.Appliers.For(it); ok {
					undone, err := a.Undo(ctx, it)
					switch {
					case errors.Is(err, contract.ErrKept):
						change.Result = catalog.StateBroken
						change.Steps = append(undone.Steps, settings.ChangeStep{Label: "needs you", State: catalog.StateBroken, Detail: err.Error()})
						changes = append(changes, change)
						continue
					case errors.Is(err, contract.ErrNoUndo):
						change.Steps = []settings.ChangeStep{{Label: it.Name, State: catalog.StateOK, Detail: "kept: " + err.Error()}}
					case err != nil:
						change.Result = catalog.StateBroken
						change.Steps = []settings.ChangeStep{{Label: "failed", State: catalog.StateBroken, Detail: err.Error()}}
						changes = append(changes, change)
						continue
					default:
						change.Steps = undone.Steps
					}
				}
			} else if row.State == settings.StateChangedSince || now.State == catalog.StateBroken {
				if k, ok := keeper(d, it); ok {
					if why := k.Kept(ctx, it); why != nil {
						change.Result = catalog.StateBroken
						change.Steps = []settings.ChangeStep{{Label: "needs you", State: catalog.StateBroken, Detail: why.Error()}}
						changes = append(changes, change)
						continue
					}
				}
				if row.State == settings.StateChangedSince {
					label := row.Where
					if label == "" {
						label = it.Name
					}
					change.Steps = []settings.ChangeStep{{Label: label, State: catalog.StateOK, Detail: "yours, kept as it is"}}
				}
			}
			if err := d.Applied.Delete(ctx, it.Name); err != nil {
				return changes, err
			}
			if it.Declares != nil && d.Public != nil {
				if err := d.Public.Undeclare(ctx, *it.Declares); err != nil {
					return changes, err
				}
			}
			off[it.Name] = true
			changes = append(changes, change)
		}
		return changes, setOff(d, off)
	},
}

func keeper(d Deps, it catalog.Setting) (contract.Keeper, bool) {
	a, ok := d.Appliers.For(it)
	if !ok {
		return nil, false
	}
	k, ok := a.(contract.Keeper)
	return k, ok
}
