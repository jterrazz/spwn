package devices

import (
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
)

const command = "devices"

func columns() []page.Column {
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100},
		{Key: "fingerprint", Label: "key", Kind: page.KindText, Keep: 30},
		{Key: "channels", Label: "may open", Kind: page.KindText, Keep: 60},
		{Key: "added", Label: "added", Kind: page.KindText, Keep: 80, Whole: true},
	}
}

func channelsOf(r devices.Row) string {
	words := make([]string, 0, len(r.Channels))
	for _, c := range r.Channels {
		words = append(words, string(c))
	}
	return strings.Join(words, ", ")
}

func rowOf(r devices.Row, mark page.Mark) page.Row {
	return page.Row{ID: r.Name, Label: r.Name, Mark: mark,
		Cells: []page.Value{page.Text(r.Fingerprint), page.Text(channelsOf(r)), page.Text(r.Added.Format("2006-01-02"))}}
}

func again(r devices.Row) string {
	// value: the command that adds a broken entry again, with the owner
	// present at this Mac; its key is the entry's, in the read's document.
	return "station devices add " + r.Name + " <its key> --channels " + strings.ReplaceAll(channelsOf(r), " ", "")
}

func Page(l devices.Listing, thisMac page.Context) page.Page {
	// invariant: this Mac's own key first, the line another device adds;
	// then the devices it trusts; then the broken entries, each with the
	// command that adds it again.
	sections := []page.Section{}
	if l.Key != "" {
		own := page.Fact{Key: "key", Value: page.Text(devices.Fingerprint(l.Key))}
		sections = append(sections, page.Section{ID: "key", Title: l.Mac, Items: []page.Item{{Fact: &own}}})
	}
	trusted := page.Section{ID: "devices", Title: "May reach " + l.Mac, Columns: columns()}
	for _, r := range l.Devices {
		row := rowOf(r, page.OK)
		trusted.Items = append(trusted.Items, page.Item{Row: &row})
	}
	if len(l.Devices) == 0 {
		trusted.Items = []page.Item{{Empty: &page.Empty{Text: "No other device added",
			Hint: &page.Hint{Command: "station devices add <name> <its key>"}}}}
	}
	sections = append(sections, trusted)
	if len(l.Broken) > 0 {
		broken := page.Section{ID: "broken", Title: "Seal broken: reaches nothing", Columns: columns()}
		for _, r := range l.Broken {
			row := rowOf(r, page.Broken)
			row.Reason = page.Failing
			row.Hint = &page.Hint{Command: again(r)}
			broken.Items = append(broken.Items, page.Item{Row: &row})
		}
		sections = append(sections, broken)
	}
	hints := []page.Hint{{Command: "station devices --json", Text: "the key line another device adds is its key"}}
	return page.Seal(page.Page{ID: command, Command: command, Context: thisMac, Sections: sections, Hints: hints})
}

func Added(r devices.Row, mac string, thisMac page.Context) page.Page {
	row := page.Row{ID: r.Name, Label: r.Name + " may reach " + mac, Mark: page.OK, Detail: page.Text(r.Fingerprint + ", by " + channelsOf(r))}
	return pages.Outcome("devices add", thisMac, nil, []page.Row{row}, "device")
}

func Forgotten(name, mac string, thisMac page.Context) page.Page {
	row := page.Row{ID: name, Label: name + " no longer reaches " + mac, Mark: page.OK}
	return pages.Outcome("devices forget", thisMac, nil, []page.Row{row}, "device")
}
