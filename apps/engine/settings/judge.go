package settings

import (
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
)

type Group string

const (
	SetByStation Group = "set"
	// value: the opt-in group of Set by station for a Mac that hosts
	// servers: applied by name only, folded until one of it is applied.
	AlwaysOnMac Group = "always-on-mac"
	// value: read from macOS, never applied.
	Checked Group = "checked"
)

type State string

const (
	StateApplied           State = "applied"
	StateChangedSince      State = "changed-since"
	StateNotApplied        State = "not-applied"
	StateOffThisMac        State = "off-here"
	StateNotForThisMachine State = "not-for-this-machine"
	// value: the row is still being read.
	StateReading State = "reading"
)

type Observation struct {
	// value: what the setting reads now on this machine, as the overview
	// reads it: in place, missing, partly there, unknown.
	State  catalog.State
	Detail string
	// value: a fingerprint of what is there now of station's effect, read
	// from the machine, never from what station would write; empty for a
	// setting whose effect has no content to compare (a login, a
	// procedure's yes or no).
	Digest string
	// value: where it lives: "~/.config/starship.toml".
	Where string
	// value: the unified diff of what station would write against what is
	// there, for a setting that writes files; empty when they are the same.
	Diff string
	// value: a setting that writes no file: its values now, and what station
	// would set.
	Before, After []KeyValue
	// value: an effect station cannot read back once done (the Dock reset).
	Blind bool
}

func (o Observation) Present() bool { return o.State == catalog.StateOK }

type Row struct {
	// value: the setting's catalog name, and Item its whole entry as the
	// machine that judged it reads it; a checked row's Item is empty.
	Name  string
	Item  catalog.Setting
	Group Group
	State State
	// value: a row of the Checked group: its state there, State left empty.
	Checked CheckedState
	// value: ChangedSince and Applied: when station applied it.
	Since time.Time
	Where string
	// value: NotForThisMachine: the reason, "Mac only"; NotApplied: what it
	// waits for, "needs starship".
	Why string
	// value: the app it needs that is not installed here.
	Needs string
	// value: a checked setting: what macOS reads ("off", "locks after
	// 10 min"), what this machine should have instead, and where the owner
	// fixes it.
	Value, Expected string
	Fix             finding.Fix
}

func Known(rows []Row, local func(string) (catalog.Setting, bool)) []Row {
	// invariant: a row a far station judged carries its item as that station
	// reads it; one from an older station, whose wire has none, is shown with
	// this station's item of the same name, and one it does not know keeps
	// its name alone.
	out := slices.Clone(rows)
	for i, r := range out {
		if r.Group == Checked || r.Item.Name != "" {
			continue
		}
		if it, ok := local(r.Name); ok {
			out[i].Item = it
		}
	}
	return out
}

func GroupOf(it catalog.Setting) Group {
	if it.Category == catalog.CategoryHeadless {
		return AlwaysOnMac
	}
	return SetByStation
}

func LivesOnReason(livesOn []catalog.MachineKind) string {
	// value: "not for this machine" with the kinds it lives on, in the
	// owner's words.
	mac := slices.Contains(livesOn, catalog.Mac)
	vm := slices.Contains(livesOn, catalog.Linux) || slices.Contains(livesOn, catalog.MacOS)
	switch {
	case mac && !vm:
		return "Mac only"
	case !mac && vm:
		return "VMs only"
	}
	return "Macs and macOS VMs only"
}

func JudgeSet(it catalog.Setting, lives bool, applied *Applied, now Observation, off bool) Row {
	// invariant: one setting's state on one machine, from its place, the owner's
	// Off here, what station recorded when it applied it and what is there now.
	// Off here wins over everything but the place; a record whose fingerprint no
	// longer matches, or whose effect is gone, is Changed since; one that still
	// matches what is there, which station now writes otherwise, is Not applied,
	// station's own earlier text for apply to write anew; with no record, a
	// setting already exactly as station writes it is Applied, from before
	// station kept records.
	row := Row{Name: it.Name, Item: it, Group: GroupOf(it), Where: now.Where}
	switch {
	case !lives:
		row.State, row.Where, row.Why = StateNotForThisMachine, "", LivesOnReason(it.Kinds)
	case off:
		row.State = StateOffThisMac
	case now.State == catalog.StatePending:
		row.State = StateReading
	case applied != nil && now.Blind:
		row.State, row.Since = StateApplied, applied.At
	case applied != nil && !now.Present() && now.Digest != "" && applied.Digest == now.Digest:
		row.State = StateNotApplied
	case applied != nil && (!now.Present() || applied.Digest != now.Digest):
		row.State, row.Since = StateChangedSince, applied.At
	case applied != nil:
		row.State, row.Since = StateApplied, applied.At
	case now.Present() && now.Diff == "" && !now.Blind:
		row.State = StateApplied
	default:
		row.State = StateNotApplied
	}
	return row
}
