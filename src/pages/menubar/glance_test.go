package menubar

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
)

// invariant: Needs you's caption says its lines in the vision's short
// words, one part per kind of finding (vision panel a: 122 updates, npm
// sign-in, a folder, firewall off, 1 failing task), never a launchd label.
func TestNeedsYouSaysItsLinesInShortWords(t *testing.T) {
	home := overview.Summary{Needs: []overview.Need{
		{Section: "accounts", Broken: true, Subject: "registry", Text: "registry sign-in expired"},
		{Section: "background", Broken: true, Subject: "com.example.x-snapshot", Text: "com.example.x-snapshot is failing, last exit 2"},
		{Section: "apps", Subject: "tool-a", Text: "1 tool-a update waiting", Call: CallUpdate},
		{Section: "apps", Subject: "tool-b", Text: "121 tool-b updates waiting", Call: CallUpdate},
		{Section: "settings", Subject: "firewall", Text: "Firewall: off"},
		{Section: "disk", Subject: "~/.vm-disks", Text: "~/.vm-disks is misplaced"},
		{Section: "background", Subject: "com.example.socket", Text: "com.example.socket is left over"},
	}}
	s := needsYou(Reading{Home: &home})
	want := []string{"122 updates", "registry sign-in", "~/.vm-disks", "firewall off", "1 failing task", "1 left-over task"}
	if got := texts(s.Glance); !slices.Equal(got, want) {
		t.Fatalf("short words %q, want %q", got, want)
	}
	two := overview.Summary{Needs: []overview.Need{
		{Section: "disk", Subject: "~/.vm-disks", Text: "~/.vm-disks is misplaced"},
		{Section: "disk", Subject: "~/.cache", Text: "~/.cache is misplaced"},
		{Section: "settings", Subject: "firewall", Text: "Firewall: off"},
		{Section: "settings", Subject: "filevault", Text: "FileVault: off"},
	}}
	if got := texts(needsYou(Reading{Home: &two}).Glance); !slices.Equal(got, []string{"2 folders to move", "2 settings"}) {
		t.Fatalf("short words %q", got)
	}
}

// invariant: a workspace that does not run has no activity: its page
// shows none, never the last measures it had while it ran.
func TestAStoppedWorkspaceShowsNoActivity(t *testing.T) {
	at := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	samples := []activity.Reading{
		{Sample: activity.Sample{At: at, CPUBusy: 100, CPUTotal: 1000, MemUsed: 1 << 30, MemTotal: 4 << 30, DiskFree: 10, DiskTotal: 20}},
		{Sample: activity.Sample{At: at.Add(2 * time.Second), CPUBusy: 180, CPUTotal: 2000, MemUsed: 1 << 30, MemTotal: 4 << 30, DiskFree: 10, DiskTotal: 20}},
	}
	read := func(state model.VMState) (page.Section, bool) {
		fleet := &model.AllMachines{Macs: []model.MacView{{Name: "jterrazz-macbook-pro", ThisMac: true, Reachable: true, VMs: []model.VM{
			{Name: "linux-1", Lifetime: model.Ephemeral, State: state},
		}}}}
		p := MachinePage(Machine{Name: "linux-1", All: fleet, Samples: samples, Asks: func(string, machine.Kind, ...string) string { return LevelYesNo }})
		for _, s := range p.Sections {
			if s.Segment == SegmentActivity {
				return s, true
			}
		}
		return page.Section{}, false
	}
	if _, ok := read(model.Running); !ok {
		t.Fatal("no activity on a running workspace")
	}
	if s, ok := read(model.Stopped); ok {
		t.Fatalf("a stopped workspace shows its activity %+v", s.Items)
	}
}

// invariant: Stop everything's last line sets the command that starts the
// engine again off as a command, never inside the sentence.
func TestStopEverythingSetsItsCommandOffAsACommand(t *testing.T) {
	fleet := &model.AllMachines{Macs: []model.MacView{{Name: "jterrazz-macbook-pro", ThisMac: true, VMs: []model.VM{
		{Name: "linux-1", Lifetime: model.Ephemeral, State: model.Running},
	}}}}
	lines := stopAll(Reading{All: fleet}).Confirm.Lines
	last := lines[len(lines)-1]
	if last.Kind != page.KindCommand || last.Text != "station engine start" {
		t.Fatalf("last line %+v, want the command alone", last)
	}
	for _, l := range lines[:len(lines)-1] {
		if l.Kind == page.KindText && strings.Contains(l.Text, "station engine start") {
			t.Fatalf("a sentence holds the command: %q", l.Text)
		}
	}
}

func texts(vs []page.Value) []string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.Text)
	}
	return out
}

// invariant: an unfolded Needs you line says a background task by what
// it is, the app it belongs to or its own words, never its launchd
// label, which stays the line's detail.
func TestNeedsYouSaysABackgroundTaskInPlainWords(t *testing.T) {
	home := overview.Summary{Needs: []overview.Need{
		{Section: "background", Broken: true, Subject: "com.jterrazz.brain.x-snapshot", Text: "com.jterrazz.brain.x-snapshot is failing, last exit 2"},
		{Section: "background", Subject: "com.docker.vmnetd", Text: "com.docker.vmnetd is left over"},
		{Section: "background", Subject: "com.bjango.istatmenus.installer", Text: "com.bjango.istatmenus.installer is left over"},
		{Section: "background", Subject: "org.example.helper", Text: "org.example.helper is left over"},
		{Section: "settings", Subject: "firewall", Text: "Firewall: off"},
	}}
	s := needsYou(Reading{Home: &home})
	want := []string{
		"A background task is failing: brain snapshot",
		"Docker left a helper behind",
		"iStat Menus left a helper behind",
		"A background task is left over: helper",
		"Firewall: off",
	}
	details := []string{"com.jterrazz.brain.x-snapshot", "com.docker.vmnetd", "com.bjango.istatmenus.installer", "org.example.helper", ""}
	for i, it := range s.Items {
		if it.Row.Label != want[i] || it.Row.Detail.Text != details[i] {
			t.Fatalf("line %d %q (detail %q), want %q (detail %q)", i, it.Row.Label, it.Row.Detail.Text, want[i], details[i])
		}
	}
}
