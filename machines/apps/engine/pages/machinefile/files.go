package machinefile

import (
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func shelves() []struct {
	shelf machinefile.Shelf
	title string
} {
	return []struct {
		shelf machinefile.Shelf
		title string
	}{{machinefile.ShelfYours, "Yours"}, {machinefile.ShelfMixin, "Mixins"}, {machinefile.ShelfShipped, "Shipped"}}
}

func makes(h *machinefile.Head) (kind, os string) {
	if h == nil {
		return "mixin", ""
	}
	return string(h.Kind), string(h.System.OS)
}

func markOfTone(tone string) (page.Mark, page.Reason) {
	switch tone {
	case "warn":
		return page.Attention, page.Differs
	case "ok":
		return page.OK, ""
	case "bad":
		return page.Broken, page.Failing
	}
	return page.Unjudged, ""
}

func Files(files []machinefile.FileView, sentence string, thisMac page.Context) page.Page {
	// value: one section per shelf, yours then the mixins then the shipped
	// files, one row per Machinefile: what it makes, where it is read, and
	// how its machines stand with it; the page's verdict under the first.
	cols := []page.Column{{Key: "name", Kind: page.KindListing, Keep: 100}, {Key: "makes", Label: "makes", Kind: page.KindText, Keep: 50}, {Key: "origin", Label: "from", Kind: page.KindText, Keep: 40}}
	var sections []page.Section
	for _, sh := range shelves() {
		s := page.Section{ID: string(sh.shelf), Title: sh.title, Columns: cols}
		for _, f := range files {
			if f.Shelf != sh.shelf {
				continue
			}
			mark, reason := markOfTone(string(f.Verdict.Tone))
			row := page.Row{ID: f.Name, Label: f.Name, Mark: mark, Reason: reason, Detail: page.Text(f.Verdict.Sentence), Cells: []page.Value{page.Texts(makes(f.Head)), page.Text(string(f.Origin))}}
			s.Items = append(s.Items, page.Item{Row: &row})
		}
		if len(s.Items) > 0 {
			sections = append(sections, s)
		}
	}
	if len(sections) == 0 {
		sections = append(sections, page.Section{ID: "machinefiles", Title: "Machinefiles", Items: []page.Item{{Empty: &page.Empty{Text: "no Machinefile is read here"}}}})
	}
	sections[0].Caption = []page.Value{page.Text(sentence)}
	return page.Seal(page.Page{ID: ID, Command: "machinefile", Context: thisMac, Sections: sections})
}

type FollowerJSON struct {
	Machine string `json:"machine"`
	Mac     string `json:"mac"`
	State   string `json:"state"`
}

type FileRowJSON struct {
	Name      string         `json:"name"`
	Shelf     string         `json:"shelf"`
	Origin    string         `json:"origin"`
	Path      string         `json:"path"`
	Kind      string         `json:"kind"`
	OS        string         `json:"os"`
	Followers []FollowerJSON `json:"followers"`
	Verdict   string         `json:"verdict"`
	Tone      string         `json:"tone"`
}

type FilesJSON struct {
	Schema  string        `json:"schema"`
	Files   []FileRowJSON `json:"files"`
	Verdict string        `json:"verdict"`
	Tone    string        `json:"tone"`
}

func FilesContract(schema string, files []machinefile.FileView, sentence, tone string) FilesJSON {
	body := FilesJSON{Schema: schema, Files: []FileRowJSON{}, Verdict: sentence, Tone: tone}
	for _, f := range files {
		kind, os := makes(f.Head)
		row := FileRowJSON{Name: f.Name, Shelf: string(f.Shelf), Origin: string(f.Origin), Path: f.Path, Kind: kind, OS: os, Followers: []FollowerJSON{}, Verdict: f.Verdict.Sentence, Tone: string(f.Verdict.Tone)}
		for _, w := range f.Followers {
			row.Followers = append(row.Followers, FollowerJSON{Machine: w.Machine, Mac: w.Mac, State: string(w.State)})
		}
		body.Files = append(body.Files, row)
	}
	return body
}
