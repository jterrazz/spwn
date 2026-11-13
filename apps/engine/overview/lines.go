package overview

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func counted(parts []Part) []Part {
	// invariant: a line names only the states it has; one with none says so
	// by its first count, zero.
	out := []Part{}
	for i, p := range parts {
		if !p.Counted || p.N > 0 || i == 0 {
			out = append(out, p)
		}
	}
	return out
}

func ActivityRead(r activity.Report) Read {
	// invariant: SPEC §5.1's one line: CPU, memory, disk free, network in,
	// each left out until it is known; a Meter that does not answer says so.
	if r.Silent {
		return Read{Line: Line{Section: section.Activity, Parts: []Part{{State: catalog.StateUnknown, Word: "no answer in 2 s"}}}, Findings: r.Findings}
	}
	s := r.Last.Sample
	var parts []Part
	if r.Window > 0 {
		parts = append(parts, Part{Word: "CPU " + activity.Percent(r.Rates.CPU)})
	}
	if s.MemTotal > 0 {
		parts = append(parts, Part{Word: "memory " + strings.TrimSuffix(activity.Memory(s.MemUsed), " GB") + " of " + activity.Memory(s.MemTotal)})
	}
	if s.DiskTotal > 0 {
		parts = append(parts, Part{Bytes: s.DiskFree, Word: "free"})
	}
	if r.Window > 0 {
		parts = append(parts, Part{Word: "in " + activity.Rate(r.Rates.NetIn)})
	}
	if run := r.Last.Run; run != nil {
		parts = []Part{{Word: "CPU " + activity.Percent(run.CPU) + " of its Mac"}, {Word: activity.Memory(run.Memory) + " resident"}}
	}
	if len(parts) == 0 {
		// invariant: a section that applies keeps its line, so a surface
		// can tell what applies from the lines alone (SPEC §3.2).
		parts = []Part{{State: catalog.StateUnknown, Word: "not measured"}}
	}
	return Read{Line: Line{Section: section.Activity, Parts: parts}, Findings: r.Findings}
}

// value: what the apps line says of the engine's nights: the apps the last
// one moved, when it is the night just past, and what the last one left
// the owner, as it read it: every version waiting past a range, each app
// left for the owner's run or whose update failed, why the shared source
// was not pulled, and why the night did not run to its end. Beside a night
// that did not, the last one that did still says what it moved and what
// waits.

type Nights struct {
	Moved       int
	Machinefile string
	Waiting     []machinefile.Waiting
	Left        []string
	Failed      []string
	Unpulled    string
	Error       string
	// value: why no night runs at all: the ledger an older station wrote,
	// refused until station init brings it to today.
	Held string
}

func NightOf(n machinefile.Night, ran bool, now time.Time) Nights {
	if !ran {
		return Nights{}
	}
	out := Nights{Machinefile: n.Machinefile, Waiting: n.Waiting, Left: n.Left, Failed: n.Failed, Unpulled: n.Unpulled, Error: n.Error}
	nights := []machinefile.Night{n}
	if n.Last != nil && !n.Complete() {
		out.Waiting = n.Last.Waiting
		nights = append(nights, *n.Last)
	}
	for _, night := range nights {
		if !night.At.Before(machinefile.LastNight(now)) {
			out.Moved += len(night.Moved)
		}
	}
	return out
}

func (n Nights) findings(m machine.Ref) []finding.Finding {
	// invariant: one Needs you line per thing the last night left the
	// owner, each fixed by the owner's own upgrade but a waiting version,
	// fixed by its line of the Machinefile.
	var out []finding.Finding
	upgrade := finding.Fix{Command: "station upgrade"}
	said := func(severity finding.Severity, subject, text string, fix finding.Fix) {
		out = append(out, finding.Finding{Machine: m, Section: section.Apps, Severity: severity, Subject: subject, Text: text, Fix: fix})
	}
	if n.Held != "" {
		said(finding.Broken, "ledger", "no night's upgrade runs: "+n.Held, finding.Fix{Command: "station init"})
	}
	if n.Error != "" {
		said(finding.Broken, n.Machinefile, "the night's upgrade of "+n.Machinefile+" failed: "+n.Error, upgrade)
	}
	if n.Unpulled != "" {
		said(finding.NeedsALook, "shared source", "the shared source was not pulled: "+n.Unpulled, upgrade)
	}
	for _, app := range n.Failed {
		said(finding.Broken, app, app+" failed its update last night", upgrade)
	}
	for _, app := range n.Left {
		said(finding.NeedsALook, app, app+" waits for your station upgrade: its update asks you", upgrade)
	}
	for _, w := range n.Waiting {
		file := w.File
		w.File = ""
		said(finding.NeedsALook, w.App, w.App+" "+w.Words(), finding.Fix{Path: file})
	}
	return out
}

func AppsRead(reports []apps.Report, m machine.Ref, night Nights) Read {
	n := apps.Counts(reports)
	updates := "updates"
	if n[apps.UpdateAvailable] == 1 {
		updates = "update"
	}
	parts := counted([]Part{
		Count(catalog.StateOK, n[apps.OK], "ok", string(apps.OK)),
		Count(catalog.StateDegraded, n[apps.UpdateAvailable], updates, string(apps.UpdateAvailable)),
		Count(catalog.StateBroken, n[apps.Failed], "failed", string(apps.Failed)),
		Count(catalog.StateUnknown, n[apps.NotManaged], "not managed", string(apps.NotManaged)),
		Count(catalog.StateUnknown, n[apps.NotKnown], "not known", string(apps.NotKnown)),
		Count(catalog.StateAbsent, n[apps.NotInstalled], "not installed", string(apps.NotInstalled)),
	})
	if night.Moved > 0 {
		parts = append(parts, Part{Word: strconv.Itoa(night.Moved) + " moved last night"})
	}
	found := append(apps.Findings(reports, m), night.findings(m)...)
	return Read{Line: Line{Section: section.Apps, Pending: n[apps.Pending] > 0, Parts: parts}, Findings: found}
}

func SettingsRead(rows []settings.Row, m machine.Ref) Read {
	var total, applied, changed, off, checked, expected int
	pending := false
	for _, r := range rows {
		if r.Group == settings.Checked && r.Checked != settings.CheckedNotForThisMachine && r.Checked != settings.CheckedOffThisMac {
			checked++
			if r.Checked == settings.AsExpected {
				expected++
			}
		}
		if r.Group != settings.SetByStation || r.State == settings.StateNotForThisMachine {
			continue
		}
		total++
		switch r.State {
		case settings.StateApplied:
			applied++
		case settings.StateChangedSince:
			changed++
		case settings.StateOffThisMac:
			off++
		case settings.StateReading:
			pending = true
		case settings.StateNotApplied, settings.StateNotForThisMachine:
		}
	}
	appliedState := catalog.StateOK
	if applied == 0 && total > 0 {
		appliedState = catalog.StateAbsent
	}
	parts := append([]Part{{Word: "set by station"}, {State: appliedState, N: applied, Of: total, Key: string(settings.StateApplied), Counted: true}},
		counted([]Part{{}, Count(catalog.StateDegraded, changed, "changed since", string(settings.StateChangedSince)), Count(catalog.StateAbsent, off, "off here", string(settings.StateOffThisMac))})[1:]...)
	if checked > 0 {
		parts = append(parts, Part{Word: "checked"}, Part{State: catalog.StateOK, N: expected, Of: checked, Key: string(settings.AsExpected), Counted: true})
	}
	return Read{Line: Line{Section: section.Settings, Pending: pending, Parts: parts}, Findings: settings.Findings(rows, m)}
}

func DiskRead(r disk.Reading, m machine.Ref) Read {
	misplaced := len(r.Of(disk.Misplaced))
	state := catalog.StateOK
	if misplaced > 0 {
		state = catalog.StateDegraded
	}
	parts := []Part{{State: state, Bytes: r.Free, Word: "free"}, {Bytes: disk.Sum(r.Of(disk.CanRebuild)), Word: "can rebuild"}}
	if misplaced > 0 {
		parts = append(parts, Count(catalog.StateDegraded, misplaced, "misplaced", string(disk.Misplaced)))
	}
	return Read{Line: Line{Section: section.Disk, Pending: r.Pending, Parts: parts}, Findings: disk.Findings(r, m)}
}

func PortsRead(l ports.Listing) Read {
	// invariant: a port is counted in its class once, a shared one as its
	// share; what listens on its machine only is not counted, and every port
	// reachable outside the tailnet with no declaration needs you.
	public, private := 0, 0
	var heard []ports.Port
	for _, h := range l.AllMachines.Macs {
		shared := map[string]bool{}
		for _, s := range h.Shared {
			shared[s.VM+":"+strconv.Itoa(s.GuestPort)] = true
		}
		private += len(shared)
		for _, p := range h.Ports {
			heard = append(heard, p)
			switch {
			case p.Class() == ports.Public:
				public++
			case shared[p.Key()] || p.Reach == ports.ThisMachine:
			default:
				private++
			}
		}
	}
	return Read{Line: Line{Section: section.Ports, Parts: []Part{{Word: "Public", N: public, Key: "public", Counted: true}, {Word: "Private", N: private, Key: "private", Counted: true}}},
		Findings: ports.Exposed(heard)}
}

func AccountsRead(r accounts.Report, m machine.Ref) Read {
	ok := 0
	var broken []Part
	for _, mm := range r.Machines {
		if !mm.ThisMac {
			continue
		}
		for _, a := range mm.Accounts {
			switch a.State {
			case catalog.StateOK:
				ok++
			case catalog.StateBroken:
				broken = append(broken, Part{State: catalog.StateBroken, Word: a.Provider})
			case catalog.StatePending:
				return Pending(section.Accounts)
			case catalog.StateDegraded, catalog.StateAbsent, catalog.StateUnknown:
			}
		}
	}
	return Read{Line: Line{Section: section.Accounts, Parts: append([]Part{Count(catalog.StateOK, ok, "", "signed-in")}, broken...)}, Findings: accounts.Findings(r, m)}
}

func BackgroundRead(tasks []background.Task, m machine.Ref) Read {
	var failing, orphans int
	for _, t := range tasks {
		switch {
		case t.Run == background.Failing:
			failing++
		case t.Group == background.GroupOrphan:
			orphans++
		}
	}
	return Read{Line: Line{Section: section.Background, Parts: counted([]Part{
		Count(catalog.StateOK, len(tasks)-failing-orphans, "", "ok"),
		Count(catalog.StateBroken, failing, "failing", string(background.Failing)),
		Count(catalog.StateDegraded, orphans, "orphans", string(background.GroupOrphan)),
	})}, Findings: background.Findings(tasks, m)}
}

func AwakeRead(r awake.Report, m machine.Ref) Read {
	// invariant: Awake is the engine's: with the engine stopped its line
	// says so rather than read off.
	if r.Engine != "" && r.Engine != awake.EngineRunning && r.Engine != awake.EngineReading {
		return Unread(section.Awake, "needs the engine")
	}
	var parts []Part
	switch r.Mode() {
	case awake.NeverSleeps:
		parts = []Part{{State: catalog.StateOK, Word: "never sleeps"}}
	case awake.AwakeUntil:
		s := r.Session
		zone := r.Zone
		if zone == nil {
			zone = time.UTC
		}
		parts = []Part{{State: catalog.StateOK, Word: "until " + s.Until.In(zone).Format("15:04")}}
		if s.Lid {
			parts = append(parts, Part{Word: "lid closed"})
		}
	case awake.StoppedByGuard:
		parts = []Part{{State: catalog.StateDegraded, Word: "stopped"}, {Word: r.LastEnd.Detail}}
	case awake.SleepsNormally:
		parts = []Part{{State: catalog.StateAbsent, Word: "sleeps normally"}}
	}
	return Read{Line: Line{Section: section.Awake, Parts: parts}, Findings: awake.Findings(r, m)}
}

func SSHRead(m machine.Ref) Read {
	// invariant: every Mac's ssh config holds a block for each VM by its
	// name, hopping through its Mac: the one door, from any Mac, is its name.
	return Read{Line: Line{Section: section.SSH, Parts: []Part{{Word: m.Name}}}}
}
