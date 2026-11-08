package menubar

import (
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	pagehome "github.com/jterrazz/studio-station/src/pages/home"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
	"strconv"
	"time"
)

func serverPage(m Machine, h model.MacView, vm model.VM) page.Page {
	p, head := vmHead(m, h, vm)
	kind := machine.Server
	p.Sections = []page.Section{head, upSection(m, vm), healthSection(vm, m.Now), activitySection(running(m, vm), vm.State == model.Running, "Activity", vm.Name), serverSection(m)}
	restart := page.Action{ID: ActionRestart, Label: "Restart", On: page.OnPage, Priority: 1, Mutates: true, Confirm: ask(m.Asks, "restart", kind, vm.Name, "Restart "+vm.Name+"?", "restart"),
		Run: &page.Run{Call: CallRestartVM, In: map[string]string{"Name": vm.Name}, Target: target(h)}}
	restart.Run.In = answered(restart.Confirm, restart.Run.In)
	stop := page.Action{ID: ActionStop, Label: "Stop", On: page.OnPage, Priority: 2, Mutates: true,
		Confirm: ask(m.Asks, "stop", kind, vm.Name, "Stop "+vm.Name+"?", "stop", "It stays stopped until you start it: the engine no longer restarts it."),
		Run:     &page.Run{Call: CallStopVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}}
	stop.Run.In = answered(stop.Confirm, stop.Run.In)
	p.Actions = []page.Action{restart, stop, openIn("--on", vm.Name)}
	p.Live = liveKeys(p)
	return page.Seal(p)
}

// value: a server's Up as vision panel f's figure: days and hours (`12 d
// 4 h`), since it booted; its restarts beside it.
func upSection(m Machine, vm model.VM) page.Section {
	up := page.Row{ID: "up", Label: "Up", Cells: []page.Value{page.None()}}
	if m.Summary != nil && m.Summary.Head.Up > 0 {
		up.Cells = []page.Value{page.List(page.Text(span(m.Summary.Head.Up)), page.Text("since boot"))}
	}
	words := vm.Restarts.Words(m.Now)
	restarts := page.Row{ID: "restarts", Label: "Restarts", Cells: []page.Value{page.List(page.Count(int64(vm.Restarts.Count), plural(vm.Restarts.Count, "restart")), page.Text(words[len(words)-1]))}}
	return page.Section{ID: SectionUp, Title: "Overview", Role: page.Summary, Columns: columns(), Items: items([]page.Row{up, restarts})}
}

func span(d time.Duration) string {
	day := 24 * time.Hour
	switch {
	case d >= day && d%day >= time.Hour:
		return strconv.Itoa(int(d/day)) + " d " + strconv.Itoa(int(d%day/time.Hour)) + " h"
	case d >= day:
		return strconv.Itoa(int(d/day)) + " d"
	case d >= time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + " h"
	}
	return strconv.Itoa(max(int(d/time.Minute), 1)) + " min"
}

// value: a server's health steps, run after every boot, how long ago its
// check last ran, and what a passing check answered (vision panel f).
func healthSection(vm model.VM, now time.Time) page.Section {
	caption := []page.Value{page.Text("after every boot")}
	if at := vm.CheckStatus.At; !at.IsZero() && !now.IsZero() {
		caption = append(caption, page.Age(at, now))
	}
	steps := pagemachines.Steps(vm)
	for i := range steps {
		if steps[i].ID == "check" && steps[i].Mark == page.OK && vm.CheckStatus.Answer != "" {
			steps[i].Cells = []page.Value{page.List(page.Text(vm.Check), page.Text(vm.CheckStatus.Answer))}
		}
	}
	return page.Section{ID: SectionHealth, Title: "Health", Caption: caption, Columns: columns(), Items: items(steps)}
}

// value: what a server's page adds, as its Overview reads it: its ports'
// counts and the command that opens a shell on it, for debugging; station
// manages nothing inside it.
func serverSection(m Machine) page.Section {
	s := page.Section{ID: SectionServer, Title: "Ports and shell", Role: page.Summary, Columns: columns()}
	if m.Summary == nil {
		s.Items = []page.Item{{Row: &page.Row{ID: "reading", Label: "reading", Cells: []page.Value{page.Pending()}}}}
		return s
	}
	for _, body := range pagehome.Body(*m.Summary, true) {
		if body.ID != "sections" {
			continue
		}
		for _, it := range body.Items {
			if it.Row == nil || it.Row.ID != string(SectionActivity) {
				s.Items = append(s.Items, it)
			}
		}
	}
	return s
}
