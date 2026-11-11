package images

import (
	"slices"
	"strconv"

	"github.com/jterrazz/studio-station/src/images"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
)

type ImageJSON struct {
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	SizeBytes int64    `json:"size_bytes"`
	Mac       string   `json:"mac"`
	VMs       []string `json:"vms"`
	Newest    bool     `json:"newest"`
}

type ListJSON struct {
	Schema string      `json:"schema"`
	Images []ImageJSON `json:"images"`
}

func ListContract(reports []images.Report) ListJSON {
	body := ListJSON{Schema: "station.images.list/v1", Images: make([]ImageJSON, 0, len(reports))}
	for _, r := range reports {
		body.Images = append(body.Images, ImageJSON{
			Name:      r.Record.Name,
			Version:   r.Record.Version,
			SizeBytes: int64(r.Record.SizeGB) * 1_000_000_000,
			Mac:       r.Mac,
			VMs:       append([]string{}, r.VMs...),
			Newest:    r.Newest,
		})
	}
	return body
}

func Columns() []page.Column {
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100},
		{Key: "version", Label: "version", Kind: page.KindVersion, Keep: 50},
		{Key: "size", Label: "size", Kind: page.KindSize, Keep: 60},
		{Key: "mac", Label: "on", Kind: page.KindText, Keep: 40, Show: page.EyeOnly},
		{Key: "on", Label: "on", Kind: page.KindText, Keep: 40, Show: page.PipeOnly},
	}
}

func Section(reports []images.Report, known []catalog.Image, thisMac page.Context) page.Section {
	// invariant: one row per image per Mac, then one row for each image of
	// the catalogue no Mac holds.
	items := make([]page.Item, 0, len(reports)+len(known))
	for _, r := range reports {
		row := page.Row{
			ID: r.Mac + "/" + r.Record.Name + "@" + r.Record.Version, Label: r.Record.Name, Mark: page.OK,
			Cells:  []page.Value{page.Version(r.Record.Version), page.Size(int64(r.Record.SizeGB) * 1_000_000_000), page.Text(page.MacName(r.Mac, thisMac.Machine)), page.Text(r.Mac)},
			Detail: madeFrom(r.VMs),
		}
		if !r.Newest {
			row.Mark, row.Reason = page.Absent, page.Outdated
			if len(r.VMs) == 0 {
				row.Detail = page.Text("older")
				row.Hint = page.Cmd("station images delete " + r.Record.Name + "@" + r.Record.Version)
			}
		}
		items = append(items, page.Item{Row: &row})
	}
	for _, img := range known {
		if slices.ContainsFunc(reports, func(r images.Report) bool { return r.Record.Name == img.Name }) {
			continue
		}
		row := page.Row{ID: img.Name, Label: img.Name, Mark: page.Absent, Reason: page.Missing, Detail: page.Text("not built yet")}
		items = append(items, page.Item{Row: &row})
	}
	return page.Section{ID: "images", Title: "Images", Role: page.Banner, Columns: Columns(), Items: items}
}

func madeFrom(vms []string) page.Value {
	if len(vms) == 0 {
		return page.Text("no VM made from it")
	}
	return page.Texts(vms...)
}

const (
	KeyBuild  = "b"
	KeyUpdate = "u"
	KeyDelete = "x"
)

func List(reports []images.Report, known []catalog.Image, thisMac page.Context) page.Page {
	return page.Seal(page.Page{
		ID: "images", Command: "images", Context: thisMac,
		Sections: []page.Section{firstLine(reports, known), Section(reports, known, thisMac)},
		Hints:    []page.Hint{*page.Cmd("station images update")},
		Actions: []page.Action{
			{Key: KeyBuild, Label: "build", On: page.OnRow, Priority: 1, Mutates: true},
			{Key: KeyUpdate, Label: "update", On: page.OnRow, Priority: 2, Mutates: true},
			{Key: KeyDelete, Label: "delete", On: page.OnRow, Priority: 3, Mutates: true, Confirm: &page.Confirm{Yes: "delete"}},
		},
	})
}

func Built(records []images.Record, thisMac page.Context) page.Page {
	return pages.Outcome("images build", thisMac, versioned(), rows(records, "built"), "images to build")
}

func Updated(updates []images.Update, thisMac page.Context) page.Page {
	out := make([]page.Row, 0, len(updates))
	for _, u := range updates {
		done := "fetched"
		if u.Newest {
			done = "is the newest"
		}
		row := pages.Done(u.Ref.String(), catalog.StateOK, done)
		row.Label = u.Ref.Name
		row.Cells = []page.Value{page.Version(u.Ref.Version)}
		out = append(out, row)
	}
	return pages.Outcome("images update", thisMac, versioned(), out, "images to update")
}

func Deleted(records []images.Record, thisMac page.Context) page.Page {
	return pages.Outcome("images delete", thisMac, versioned(), rows(records, "deleted"), "images to delete")
}

func versioned() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "version", Kind: page.KindVersion, Keep: 50}}
}

func rows(records []images.Record, done string) []page.Row {
	out := make([]page.Row, 0, len(records))
	for _, r := range records {
		row := pages.Done(r.Name+"@"+r.Version, catalog.StateOK, done)
		row.Label = r.Name
		row.Cells = []page.Value{page.Version(r.Version)}
		out = append(out, row)
	}
	return out
}

// value: the first line: how many images on how many Macs, the older ones
// and the catalogue's images no Mac holds yet.
func firstLine(reports []images.Report, known []catalog.Image) page.Section {
	macs := map[string]bool{}
	older := 0
	for _, r := range reports {
		macs[r.Mac] = true
		if !r.Newest {
			older++
		}
	}
	words := []string{}
	if older > 0 {
		words = append(words, count(older, "older", "older"))
	}
	for _, img := range known {
		if !slices.ContainsFunc(reports, func(r images.Report) bool { return r.Record.Name == img.Name }) {
			words = append(words, img.Name+" not built yet")
		}
	}
	label := count(len(reports), "image", "images") + " on " + count(len(macs), "Mac", "Macs")
	if len(reports) == 0 {
		label = "No image yet"
	}
	row := page.Row{ID: "summary/images", Label: label, Mark: page.OK, Cells: []page.Value{page.Texts(words...)}}
	return page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true, Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: []page.Item{{Row: &row}}}
}

func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
