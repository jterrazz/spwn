package calls

import (
	"context"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type ApplyIn struct {
	// value: the settings named; none is every Not applied and Changed
	// since setting of Set by station that asks no value.
	Names []string
	// value: the values given for a setting, by its name; a setting with
	// values and none given asks them, at a terminal.
	Values      map[string]map[string]string
	Interactive bool
	Confirm     string
}

var Apply = call.Call[Deps, ApplyIn, []settings.Change]{
	Name: "settings.apply", Section: section.Settings, Where: call.AtTarget, Applies: settings.Applies, Confirm: confirm.Overwrite,
	Errors: []fail.Code{Unknown, IsChecked, NotForThisMachine, NeedsAdmin, NeedsTerminal, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in ApplyIn, _ call.Progress) ([]settings.Change, error) {
		// invariant: every setting is judged before any question and any
		// effect; one over a change of the owner's asks first, as it is lost.
		off := offThisMac(d)
		targets, err := targetsOf(ctx, d, in, off)
		if err != nil {
			return nil, err
		}
		var changed []string
		for _, t := range targets {
			if t.row.State == settings.StateChangedSince {
				changed = append(changed, t.it.Name)
			}
		}
		if len(changed) > 0 {
			if refused := confirm.Require(confirm.For(confirm.Overwrite, d.Kind), strings.Join(changed, ", "), in.Confirm); refused != nil {
				return nil, refused
			}
		}
		changes := make([]settings.Change, 0, len(targets))
		for _, t := range targets {
			values, given := in.Values[t.it.Name]
			if !given && asks(t.it) && in.Interactive && d.Ask != nil && len(t.it.Inputs) > 0 {
				if values, err = d.Ask.Values(t.it); err != nil {
					return nil, err
				}
			}
			change, err := d.Settings.Apply(ctx, t.it.Name, values, in.Interactive || answered(t.it, in))
			change.Item, change.Verb = t.it.Name, "apply"
			if err != nil {
				change.Result = catalog.StateBroken
				if len(change.Steps) == 0 {
					change.Steps = []settings.ChangeStep{{Label: "failed", State: catalog.StateBroken, Detail: err.Error()}}
				}
				changes = append(changes, change)
				continue
			}
			change.Result = catalog.StateOK
			if off[t.it.Name] {
				delete(off, t.it.Name)
				if err := setOff(d, off); err != nil {
					return changes, err
				}
			}
			if err := keep(ctx, d, t.it); err != nil {
				return changes, err
			}
			// invariant: SPEC 5.6.3: only an always-on Mac's Remote login
			// is meant for its network; a laptop's goes where it goes.
			if t.it.Declares != nil && d.Public != nil && !d.Caps.Battery {
				if err := d.Public.Declare(ctx, *t.it.Declares, t.it.Name); err != nil {
					return changes, err
				}
			}
			changes = append(changes, change)
		}
		return changes, nil
	},
}

// invariant: values given stand for the terminal only for a setting whose
// one question is its form: one that runs its own dialog still needs one.
func answered(it catalog.Setting, in ApplyIn) bool {
	return in.Values[it.Name] != nil && !it.Interactive
}

type target struct {
	it  catalog.Setting
	row settings.Row
}

func targetsOf(ctx context.Context, d Deps, in ApplyIn, off map[string]bool) ([]target, error) {
	var out []target
	if len(in.Names) == 0 {
		for _, it := range d.Settings.Items() {
			if settings.GroupOf(it) != settings.SetByStation || asks(it) {
				continue
			}
			row, _ := judge(ctx, d, it, off)
			if (row.State == settings.StateNotApplied && row.Needs == "") || row.State == settings.StateChangedSince {
				out = append(out, target{it: it, row: row})
			}
		}
		return out, nil
	}
	seen := map[string]bool{}
	for _, name := range in.Names {
		if seen[name] {
			continue
		}
		seen[name] = true
		if c, ok := settings.CheckedNamed(name); ok {
			return nil, fail.New(fail.Refused, IsChecked, name, map[string]string{"fix": c.Fix})
		}
		it, err := find(d, name, true)
		if err != nil {
			return nil, err
		}
		if asks(it) && !in.Interactive && !answered(it, in) {
			return nil, fail.New(fail.NeedsTerminal, NeedsTerminal, name, nil)
		}
		if err := admitted(d, it, "apply"); err != nil {
			return nil, err
		}
		if err := d.Settings.Admit(name, in.Interactive || answered(it, in)); err != nil {
			return nil, err
		}
		row, _ := judge(ctx, d, it, map[string]bool{})
		out = append(out, target{it: it, row: row})
	}
	return out, nil
}

// invariant: what station applied is recorded as it reads once written, so
// a later edit by hand shows as Changed since.
func keep(ctx context.Context, d Deps, it catalog.Setting) error {
	a, ok := d.Appliers.For(it)
	if !ok {
		return nil
	}
	now, err := a.Observe(ctx, it)
	if err != nil {
		return err
	}
	return d.Applied.Put(ctx, settings.Applied{Setting: it.Name, Digest: now.Digest, At: d.Now().UTC()})
}

func setOff(d Deps, off map[string]bool) error {
	names := make([]string, 0, len(off))
	for _, it := range d.Settings.Items() {
		if off[it.Name] {
			names = append(names, it.Name)
		}
	}
	for _, it := range catalog.CheckedSettings {
		if off[it.Name] {
			names = append(names, it.Name)
		}
	}
	return d.OffThisMac.Set(names)
}
