package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/catalog"

	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ui/token"
)

type Fixture struct {
	Name string
	Page page.Page
}

func All() []Fixture {
	return append(append(Apps(), Kitchen()...), Estate()...)
}

func row(id string, mark page.Mark, cells ...page.Value) page.Row {
	return page.Row{ID: id, Mark: mark, Label: id, Cells: cells}
}

func it(r page.Row) page.Item { return page.Item{Row: &r} }

func Kitchen() []Fixture {
	return []Fixture{
		{"kitchen-sink.loading", page.Seal(kitchen(func(p *page.Page) {
			for si := range p.Sections {
				pendAll(p.Sections[si].Items)
			}
			p.Strip = nil
		}))},
		{"kitchen-sink.empty", page.Seal(kitchen(func(p *page.Page) {
			for si := range p.Sections {
				p.Sections[si].Items = []page.Item{{Empty: &page.Empty{Noun: "VMs", Hint: page.KeyHint("station machines create", "n")}}}
			}
		}))},
		{"kitchen-sink.error", page.Seal(page.Page{ID: "home", Command: "status", Context: thisMacContext(), Error: &page.Error{Message: "the settings file is not valid JSON", Hint: "repair the JSON of ~/.config/station/settings.json"}})},
		{"kitchen-sink.full", page.Seal(kitchen(func(*page.Page) {}))},
	}
}

func pendAll(items []page.Item) {
	for i := range items {
		switch {
		case items[i].Row != nil:
			pend(items[i].Row)
		case items[i].Group != nil:
			if !items[i].Group.Heading() {
				pend(&items[i].Group.Head)
			}
			pendAll(items[i].Group.Items)
		}
	}
}

func pend(r *page.Row) {
	r.Mark, r.Reason, r.Hint, r.Detail = page.Awaiting, "", nil, page.None()
	for i := range r.Cells {
		if r.Cells[i].Kind == page.KindSize {
			r.Cells[i] = page.Stale(r.Cells[i])
			continue
		}
		r.Cells[i] = page.Pending()
	}
}

func kitchen(edit func(*page.Page)) page.Page {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	values := page.Section{
		ID: "values", Title: "Values", Caption: []page.Value{page.Range(40000, 40999)},
		Columns: []page.Column{
			{Key: "name", Kind: page.KindText, Keep: 100},
			{Key: "a", Label: "value", Kind: page.KindText, Keep: 40},
			{Key: "b", Label: "number", Kind: page.KindCount, Keep: 60},
		},
		Items: []page.Item{
			it(row("text", page.OK, page.Text("plain words"), page.Count(120, "outdated"))),
			it(row("size", page.OK, page.Size(275), page.Size(22600))),
			it(row("ratio", page.OK, page.Ratio(12400000000, 17200000000), page.Percent(345))),
			it(row("percent", page.OK, page.Percent(35), page.Duration(1951200))),
			it(row("version", page.OK, page.Version("1.22209.3,babe11577dfefe3e209c06bd674628d862f0dbae"), page.Port(5173))),
			it(row("age", page.OK, page.Age(now.Add(-22*time.Minute), now), page.PortMap(5173, 40001))),
			it(row("old", page.OK, page.Age(now.AddDate(0, -2, 0), now), page.Duration(45))),
			it(row("until", page.OK, page.Until(now.Add(11*time.Hour+40*time.Minute+20*time.Second), now), page.Duration(42))),
			it(row("soon", page.OK, page.Until(now.Add(40*time.Second), now))),
			it(row("expired", page.OK, page.Until(now.Add(-time.Minute), now))),
			it(row("url", page.OK, page.URL("https://jterrazz-macbook-pro.tail1234.ts.net:40001"))),
			it(row("identity", page.OK, page.Identity("admin@jterrazz.com"))),
			it(row("file", page.OK, page.Path("/Applications/Tailscale.app/Contents/MacOS/Tailscale"))),
			it(row("command", page.OK, page.Command("station apps update --method brew"))),
			it(row("list", page.OK, page.Texts("brew", "system"))),
			it(row("switch", page.OK, page.On(), page.Count(3, ""))),
			it(row("sip", page.OK, page.Off())),
			it(row("unknown", page.Attention, page.Unknown())),
		},
	}
	values.Items[len(values.Items)-1].Row.Reason, values.Items[len(values.Items)-1].Row.Detail = page.Late, page.Text("no answer in 2s")
	marks := page.Section{ID: "marks", Title: "Marks", Fold: page.Auto, Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "state", Kind: page.KindText, Keep: 50}}}
	reasons := []struct {
		reason page.Reason
		mark   page.Mark
		detail string
		hint   string
	}{
		{page.Outdated, page.Attention, "120 outdated", "station apps update --method brew"},
		{page.Misplaced, page.Attention, "misplaced", "station disk move ~/.bun"},
		{page.Unreachable, page.Attention, "unreachable", ""},
		{page.Mismatch, page.Attention, "station v2.0.0-240, This Mac v2.0.0-242", ""},
		{page.Unmanaged, page.Attention, "on PATH, not managed by cask", ""},
		{page.Missing, page.Attention, "installed, mise not installed", "station apps install mise"},
		{page.Partial, page.Attention, "policy partly applied", ""},
		{page.Differs, page.Attention, "differs from station's", "station settings apply " + catalog.HypervisorLauncher},
		{page.Unset, page.Attention, "name not set", "station init"},
		{page.Stopped, page.Attention, "stopped", "station engine start"},
		{page.Late, page.Attention, "no answer in 2s", ""},
		{page.Degraded, page.Attention, "Serve is off on Tailscale", ""},
		{page.Orphan, page.Attention, "orphan: Docker.app is gone", "sudo launchctl bootout system/com.docker.vmnetd"},
		{page.Waiting, page.Absent, "waiting", ""},
		{page.Failing, page.Broken, "failing" + token.Sep + "last exit 2", ""},
		{page.Refused, page.Broken, "refused", ""},
		{page.Down, page.Broken, "not running", ""},
		{page.Outlived, page.Broken, "stale", "station machines delete linux-2"},
		{page.Expiring, page.Attention, "42m left", "station accounts share claude --with api"},
		{page.Guard, page.Attention, "28%" + token.Sep + "on battery", "station awake"},
	}
	for _, r := range reasons {
		x := row(string(r.reason), r.mark, page.Text(string(r.mark)))
		x.Reason, x.Detail = r.reason, page.Text(r.detail)
		if r.reason == page.Misplaced {
			x.Label = "misplaced"
		}
		if r.hint != "" {
			x.Hint = page.Cmd(r.hint)
		}
		marks.Items = append(marks.Items, it(x))
	}
	marks.Items = append(marks.Items,
		it(page.Row{ID: "absent", Mark: page.Absent, Label: "absent", Hint: page.KeyHint("station apps install mas", "i")}),
		it(row("off-here", page.OffHere, page.Text("turned off on this machine"))),
		it(row("not-for-this-machine", page.NotForThisMachine, page.Text("Macs only"))),
		it(row("not-known", page.NotKnown, page.Text("does not answer"))),
	)
	allMachines := page.Section{
		ID: "all-machines", Title: "All machines",
		Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "kind", Kind: page.KindText, Keep: 40}, {Key: "mac", Kind: page.KindText, Keep: 10, Show: page.PipeOnly}},
		Items: []page.Item{{Group: &page.Group{
			Head: row("jterrazz-mac-mini", page.OK, page.Text("Mac"), page.Text("jterrazz-mac-mini")),
			Items: []page.Item{
				{Group: &page.Group{
					Head: row("linux-1", page.OK, page.Text("persistent"), page.Text("jterrazz-mac-mini")),
					Items: []page.Item{
						{Link: &page.Link{Label: "8081", Lead: page.PortMap(8081, 40002), Value: page.URL("https://jterrazz-mac-mini.ts.net:40002"), Detail: page.Text("vite"), Page: "ports", Row: "api"}},
						it(row("web", page.OK, page.Text("server"), page.Text("jterrazz-mac-mini"))),
					},
				}},
				it(row("linux-2", page.Absent, page.Text("persistent"), page.Text("jterrazz-mac-mini"))),
			},
		}}},
	}
	disk := page.Section{
		ID: "disk", Title: "Disk",
		Columns: []page.Column{{Key: "name", Label: "app", Kind: page.KindText, Keep: 100}, {Key: "levels", Kind: page.KindLevels, Keep: 80, Levels: []string{"data", "kept"}}},
		Items: []page.Item{
			it(row("station", page.OK, page.Levels(page.Labeled("data", page.Size(275)), page.Labeled("kept", page.Size(546))))),
			it(row(catalog.HypervisorApp, page.OK, page.Levels(page.Labeled("data", page.Size(73300000000))))),
		},
	}
	prose := page.Section{
		ID: "prose", Title: "Prose",
		Items: []page.Item{
			{Fact: &page.Fact{Key: "machine", Value: page.Text("jterrazz-macbook-pro")}},
			{Fact: &page.Fact{Key: "uptime", Value: page.Duration(1951200)}},
			{Note: &page.Note{Text: "Serve is off on Tailscale, so no shared port is reachable from another machine until it is turned on in the admin console."}},
			{Error: &page.Error{Message: "defaults is not installed", Hint: "run station init"}},
			{Empty: &page.Empty{Noun: "shared accounts", Hint: page.Cmd("station accounts share <account> --with <vm>")}},
		},
	}
	peers := page.Section{
		ID: "peers", Title: "Peers", Role: page.Banner,
		Columns: []page.Column{{Key: "name", Kind: page.KindListing, Keep: 100}, {Key: "state", Kind: page.KindText, Keep: 50}},
		Items:   []page.Item{it(page.Row{ID: "iphone", Label: "iphone", Cells: []page.Value{page.Text("offline")}})},
	}
	todo := page.Section{
		ID: "actions", Title: "To do", Role: page.Actions,
		Items: []page.Item{it(page.Row{ID: "marks/outdated", Mark: page.Attention, Label: "brew", Reason: page.Outdated, Detail: page.Count(120, "outdated"), Hint: page.Cmd("station apps update --method brew")})},
	}
	summary := page.Section{
		ID: "summary", Title: "Summary", Role: page.Summary,
		Items: []page.Item{it(page.Row{ID: "marks", Mark: page.Broken, Label: "Marks", Reason: page.Failing, Detail: page.Count(4, "broken")})},
	}
	p := page.Page{
		ID: "home", Command: "status", Context: page.Context{Machine: "jterrazz-macbook-pro", Kind: "computer", Registered: true},
		Strip: &page.Strip{
			CPU: page.Percent(340), Spark: []int{3, 10, 25, 40, 60, 80, 100, 50, 34, 20, 10, 5, 0, 15, 30, 34},
			Memory: page.Ratio(12400000000, 17200000000), Battery: page.Percent(1000), Awake: page.Texts("awake", "3h 12m"),
			Net: []page.Value{page.Text("28.2 KB/s"), page.Text("14.5 KB/s")},
		},
		Sections: []page.Section{todo, summary, values, marks, allMachines, disk, prose, peers},
		Hints:    []page.Hint{*page.Cmd("station overview --all"), {Command: "station apps", Text: "for the catalogue"}},
		Actions: []page.Action{
			{Key: "enter", Label: "fix", On: page.OnRow, Priority: 1, Mutates: true, Confirm: &page.Confirm{Question: "Delete linux-1?", Lines: []page.Value{page.Text("its disk is removed"), page.Command("station machines delete linux-1")}, Danger: true, Yes: "delete"}},
			{Key: "ctrl+r", Label: "restart", On: page.OnRow, Rows: []string{"linux-1"}, Priority: 2, Mutates: true, Confirm: &page.Confirm{Question: "Restart api?", Type: "api", Yes: "restart"}},
			{Key: "n", Label: "create", On: page.OnPage, Priority: 3, Mutates: true},
		},
	}
	edit(&p)
	return p
}
