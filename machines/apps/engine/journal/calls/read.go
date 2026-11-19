package calls

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type ReadIn struct {
	// value: that Mac and its VMs alone.
	Mac string
	// value: that machine alone.
	Machine string
	// value: the lines of that app, machine, setting, skill or image alone.
	Subject *journal.Subject
	// value: only the newest night of the Macs asked.
	Night bool
	// value: the day the read ends before, excluded, "YYYY-MM-DD"; empty
	// reads up to today.
	Before string
	// value: the days before it the read covers: 7 when none, 30 at most.
	Days int
}

var Read = call.Call[Deps, ReadIn, journal.Reading]{
	Name: "journal.read", Where: call.InProcess, Applies: Applies,
	Errors: []fail.Code{UnknownMac, UnknownMachine, BadDay, Unreadable},
	Run: func(ctx context.Context, d Deps, in ReadIn, _ call.Progress) (journal.Reading, error) {
		// invariant: every Mac asked answers its own lines and its VMs'; a
		// Mac that does not, or whose station keeps none, is an Unheard row,
		// never a refusal; This Mac's journal that does not read is one.
		if in.Before != "" {
			if _, err := time.Parse(journal.DayForm, in.Before); err != nil {
				return journal.Reading{}, fail.New(fail.Refused, BadDay, in.Before, nil)
			}
		}
		all, err := d.Macs.Machines(ctx)
		if err != nil {
			return journal.Reading{}, err
		}
		asked, err := askedOf(all, in)
		if err != nil {
			return journal.Reading{}, err
		}
		now := d.Now()
		since := now.Add(-journal.Horizon)
		var lines []journal.Line
		var unheard []journal.Unheard
		var others []string
		for _, mac := range asked {
			if !mac.ThisMac {
				others = append(others, mac.Name)
				continue
			}
			own, err := d.Journal.Lines(ctx, since, time.Time{})
			if err != nil {
				return journal.Reading{}, coded(err)
			}
			lines = append(lines, own...)
		}
		answers := d.Macs.Kept(ctx, others, KeptIn{Since: since})
		if err := ctx.Err(); err != nil {
			return journal.Reading{}, err
		}
		for _, mac := range others {
			a := answers[mac]
			if a.Lines == nil && a.Why != "" {
				unheard = append(unheard, journal.Unheard{Mac: mac, Why: a.Why})
				continue
			}
			lines = append(lines, a.Lines...)
		}
		gone(lines, all, d.Known)
		days := in.Days
		if days <= 0 {
			days = journal.Days
		}
		days = min(days, journal.MostDays)
		groups, next := journal.Read(lines, now, journal.Query{
			Before: in.Before, Days: days, Night: in.Night,
			Narrowed: in.Mac != "" || in.Machine != "" || in.Subject != nil,
			Keep:     keep(in),
		})
		out := journal.Reading{Groups: groups, Unheard: unheard, Next: next}
		if len(groups) == 0 && len(unheard) == 0 {
			out.Empty = emptyOf(in, days, now)
		}
		return out, nil
	},
}

func askedOf(all machines.AllMachines, in ReadIn) ([]machines.MacView, error) {
	switch {
	case in.Mac != "":
		for _, mac := range all.Macs {
			if mac.Name == in.Mac {
				return []machines.MacView{mac}, nil
			}
		}
		return nil, fail.New(fail.NotFound, UnknownMac, in.Mac, nil)
	case in.Machine != "":
		for _, mac := range all.Macs {
			if mac.Name == in.Machine || slices.ContainsFunc(mac.VMs, func(vm machines.VM) bool { return vm.Name == in.Machine }) {
				return []machines.MacView{mac}, nil
			}
		}
		return nil, fail.New(fail.NotFound, UnknownMachine, in.Machine, nil)
	}
	return all.Macs, nil
}

func keep(in ReadIn) func(journal.Line) bool {
	return func(l journal.Line) bool {
		if in.Machine != "" && l.Machine != in.Machine {
			return false
		}
		if s := in.Subject; s != nil && (l.Subject == nil || l.Subject.Kind != s.Kind || l.Subject.Name != s.Name) {
			return false
		}
		return true
	}
}

func gone(lines []journal.Line, all machines.AllMachines, known func(journal.Subject) bool) {
	// invariant: a subject whose thing no longer exists at the read keeps
	// its name and says it is gone: a machine no Mac lists, a package or an
	// image the catalog no longer knows.
	names := map[string]bool{}
	for _, mac := range all.Macs {
		names[mac.Name] = true
		for _, vm := range mac.VMs {
			names[vm.Name] = true
		}
	}
	for i, l := range lines {
		if l.Subject == nil {
			continue
		}
		s := *l.Subject
		switch s.Kind {
		case journal.Machine:
			s.Gone = !names[s.Name]
		case journal.App, journal.Setting, journal.Skill, journal.Image:
			s.Gone = known != nil && !known(s)
		}
		lines[i].Subject = &s
	}
}

func emptyOf(in ReadIn, days int, now time.Time) string {
	// value: the engine's sentence for a read that has no line to show.
	if in.Night {
		return "No night upgrade ran in the last 30 days."
	}
	if in.Before == "" {
		return "Nothing was done in the last " + strconv.Itoa(days) + " days."
	}
	return "Nothing was done in the " + strconv.Itoa(days) + " days before " + journal.Words(in.Before, now) + "."
}
