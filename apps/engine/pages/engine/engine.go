package engine

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
)

const (
	ID  = "engine"
	Row = "engine"
)

func columns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "detail", Kind: page.KindText, Keep: 60}}
}

func clock(t time.Time, zone *time.Location) string {
	if zone == nil {
		zone = time.UTC
	}
	return t.In(zone).Format("15:04")
}

// value: how long the engine has run, in its largest unit alone (`3 d`,
// `5 h`): a reading says it at a glance.
func uptime(d time.Duration) string {
	for _, u := range []struct {
		n    time.Duration
		name string
	}{{24 * time.Hour, "d"}, {time.Hour, "h"}, {time.Minute, "min"}} {
		if d >= u.n {
			return strconv.Itoa(int(d/u.n)) + " " + u.name
		}
	}
	return strconv.Itoa(int(d/time.Second)) + " s"
}

func count(n int, one string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + one + "s"
}

func Read(r engine.Reading, zone *time.Location, thisMac page.Context) page.Page {
	// value: an engine's reading as one row: its state, then what it runs.
	row := page.Row{ID: Row, Label: Row, Mark: page.OK}
	switch r.State {
	case engine.EngineRunning:
		parts := []string{"running", r.Version, "up " + uptime(r.Up), count(r.VMs, "VM")}
		if r.AwakeUntil != nil {
			parts = append(parts, "Awake until "+clock(*r.AwakeUntil, zone))
		}
		row.Cells = []page.Value{page.Texts(parts...)}
		if r.Skewed() {
			row.Mark, row.Reason = page.Attention, page.Mismatch
			row.Cells = append(row.Cells, page.Text("the engine runs "+r.Version+", station is "+r.Station))
			row.Hint = page.Cmd("station engine start")
		}
	case engine.EngineStopped:
		row.Mark, row.Reason = page.Attention, page.Stopped
		parts := []string{"stopped"}
		if !r.Stopped.At.IsZero() {
			parts = append(parts, "since "+clock(r.Stopped.At, zone)+" by "+r.Stopped.By)
		}
		row.Cells = []page.Value{page.Texts(parts...)}
		row.Hint = page.Cmd("station engine start")
	default:
		row.Mark, row.Reason = page.Attention, page.Missing
		row.Cells = []page.Value{page.Text("not installed")}
		row.Hint = page.Cmd("station engine start")
	}
	s := page.Section{ID: "engine", Title: "Engine", Role: page.Banner, Columns: columns(), Items: []page.Item{{Row: &row}}}
	return page.Seal(page.Page{ID: ID, Command: "engine", Context: thisMac, Sections: []page.Section{s}})
}

func Stop(r engine.StopReport, thisMac page.Context) page.Page {
	// value: a stop as one sentence: what it stopped, what it ended, and
	// whether the engine stopped with it.
	var stopped, failed []string
	for _, v := range r.VMs {
		if v.Result == engine.ResultFailed {
			failed = append(failed, v.Name)
			continue
		}
		stopped = append(stopped, v.Name)
	}
	var parts []string
	if len(stopped) > 0 {
		parts = append(parts, "stopped "+count(len(stopped), "VM")+" ("+strings.Join(stopped, ", ")+")")
	}
	switch r.Awake {
	case engine.AwakeEnded:
		parts = append(parts, "Awake off")
	case engine.AwakeReleased:
		parts = append(parts, "the lid released")
	}
	if r.Sharing {
		parts = append(parts, "sharing off")
	}
	for _, s := range r.Servers {
		parts = append(parts, s+" skipped (server)")
	}
	state := catalog.StateOK
	switch r.Engine {
	case engine.EngineHalted:
		parts = append(parts, "the engine stopped")
	case engine.EngineAlready:
		parts = append(parts, "the engine was already stopped")
	case engine.EngineKept:
		state = catalog.StateBroken
		parts = append(parts, strings.Join(failed, ", ")+" did not stop", "the engine keeps running")
	case engine.EngineServers:
		parts = append(parts, "the engine keeps running for "+strings.Join(r.Servers, ", "))
	}
	row := pages.Done(Row, state, "")
	row.Detail = page.Texts(parts...)
	p := pages.Outcome("engine stop", thisMac, nil, []page.Row{row}, "nothing to stop")
	if r.Engine == engine.EngineHalted || r.Engine == engine.EngineAlready {
		p.Hints = []page.Hint{*page.Cmd("station engine start")}
	}
	return p
}

func StopConfirm(w engine.Work, mac string) page.Confirm {
	lines := make([]page.Value, 0, len(w.VMs)+1)
	for _, v := range w.VMs {
		lines = append(lines, page.Text(v.Name+" will stop; its disk is kept"))
	}
	for _, s := range w.Servers {
		lines = append(lines, page.Text(s+" skipped (server)"))
	}
	if len(w.Servers) > 0 {
		lines = append(lines, page.Text("Awake ends, and the engine keeps running for "+strings.Join(w.Servers, ", ")))
		return page.Confirm{Question: "Stop everything on " + mac + "?", Lines: lines, Danger: true, Yes: "Stop"}
	}
	lines = append(lines, page.Text("Awake ends; the engine stays off until you log in again or run"), page.Command("station engine start"))
	return page.Confirm{Question: "Stop everything on " + mac + "?", Lines: lines, Danger: true, Yes: "Stop"}
}

func Start(r engine.StartReport, thisMac page.Context) page.Page {
	words := []string{"running", r.Version}
	switch {
	case r.Installed.From != "":
		words = []string{"restarted", r.Installed.From + " to " + r.Version}
	case r.Installed.Restarted:
		words[0] = "restarted"
	case r.Installed.Written:
		words[0] = "installed, running"
	case r.Already:
		words[0] = "already running"
	}
	if r.Installed.Moved {
		words = append(words, "its job moved to "+catalog.EngineLabel)
	}
	row := pages.Done(Row, catalog.StateOK, "running")
	row.Detail = page.Texts(words...)
	return pages.Outcome("engine start", thisMac, nil, []page.Row{row}, "engine to start")
}
