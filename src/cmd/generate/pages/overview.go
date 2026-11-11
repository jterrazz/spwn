package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/overview"
	pagehome "github.com/jterrazz/studio-station/src/pages/home"
)

// value: the Overview of This Mac as vision 07 draws it: five things need
// you, broken first and each in the one order, a line per section, the VMs
// it runs.
func overviewOf(needs bool) overview.Summary {
	s := overview.Summary{
		Head:   overview.Head{Name: "jterrazz-macbook-pro", Kind: machine.Computer, ThisMac: true, Up: 3 * 24 * time.Hour},
		Engine: overview.EngineRunning,
		Lines: []overview.Line{
			{Section: "activity", Parts: []overview.Part{{Word: "CPU 34%"}, {Word: "memory 12.4 of 16 GB"}, {Bytes: 388_000_000_000, Word: "free"}, {Word: "in 2.1 MB/s"}}},
			{Section: "apps", Parts: []overview.Part{
				overview.Count(catalog.StateOK, 39, "ok", "ok"), overview.Count(catalog.StateDegraded, 122, "updates", "update-available"),
				overview.Count(catalog.StateUnknown, 35, "not managed", "not-managed"), overview.Count(catalog.StateAbsent, 34, "not installed", "not-installed"),
			}},
			{Section: "settings", Parts: []overview.Part{{Word: "set by station"}, {State: catalog.StateOK, N: 12, Of: 22, Key: "applied", Counted: true}, {Word: "checked"}, {State: catalog.StateOK, N: 8, Of: 9, Key: "as-expected", Counted: true}}},
			{Section: "disk", Parts: []overview.Part{{State: catalog.StateDegraded, Bytes: 388_000_000_000, Word: "free"}, {Bytes: 34_800_000_000, Word: "can rebuild"}, overview.Count(catalog.StateDegraded, 1, "misplaced", "misplaced")}},
			{Section: "ports", Parts: []overview.Part{{Word: "Public", Key: "public", Counted: true}, {Word: "Private", N: 2, Key: "private", Counted: true}}},
			{Section: "accounts", Parts: []overview.Part{overview.Count(catalog.StateOK, 6, "", "signed-in"), {State: catalog.StateBroken, Word: "npm"}}},
			{Section: "background", Parts: []overview.Part{overview.Count(catalog.StateOK, 10, "", "ok"), overview.Count(catalog.StateBroken, 1, "failing", "failing")}},
			{Section: "awake", Parts: []overview.Part{{State: catalog.StateOK, Word: "until 18:00"}}},
		},
		VMs: []overview.VM{{Name: "linux-1", Running: true}, {Name: "api"}},
	}
	if needs {
		misplaced := "~/." + catalog.HypervisorApp
		s.Needs = []overview.Need{
			{Section: "accounts", Subject: "npm", Broken: true, Text: "npm sign-in expired", Command: "npm login"},
			{Section: "background", Subject: "com.jterrazz.brain.x-snapshot", Broken: true, Text: "1 background task failing", Command: "station background"},
			{Section: "apps", Subject: "brew", Text: "122 brew updates waiting", Call: "apps.update", Args: map[string]string{"manager": "brew"}},
			{Section: "settings", Subject: "firewall", Text: "Firewall off", Path: "System Settings > Network > Firewall"},
			{Section: "disk", Subject: misplaced, Text: misplaced + " is misplaced", Call: "disk.move", Args: map[string]string{"path": misplaced}},
		}
	}
	return s
}

func overviewPages() []Fixture {
	loading := overviewOf(false)
	loading.Pending = true
	for i := range loading.Lines {
		loading.Lines[i] = overview.Line{Section: loading.Lines[i].Section, Pending: true}
	}
	return states("overview",
		pagehome.Summary(loading, thisMacContext(), false),
		pagehome.Summary(overviewOf(false), thisMacContext(), false),
		failure("overview", "overview", "the engine is stopped on jterrazz-macbook-pro", "station engine start"),
		pagehome.Summary(overviewOf(true), thisMacContext(), false),
	)
}
