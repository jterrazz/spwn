package page_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/page"
)

func cols() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText}, {Key: "levels", Kind: page.KindLevels, Levels: []string{"data"}}}
}

func section(items ...page.Item) page.Section {
	return page.Section{ID: "s", Title: "S", Columns: cols(), Items: items}
}

func row(r page.Row) page.Item { return page.Item{Row: &r} }

func TestSealRefusesWhatNoPageMayHold(t *testing.T) {
	ok := page.Row{ID: "a", Mark: page.OK, Label: "a"}
	cases := map[string]page.Page{
		"an ID unique in the page":       {Sections: []page.Section{section(), section()}},
		"a title is required":            {Sections: []page.Section{{ID: "s"}}},
		"carries a mark":                 {Sections: []page.Section{section(row(page.Row{ID: "a"}))}},
		"carries a reason":               {Sections: []page.Section{section(row(page.Row{ID: "a", Mark: page.Attention, Detail: page.Text("x")}))}},
		"unique in its section":          {Sections: []page.Section{section(row(ok), row(ok))}},
		"names its command":              {Sections: []page.Section{section(row(page.Row{ID: "a", Mark: page.Absent, Hint: &page.Hint{}}))}},
		"is External":                    {Hints: []page.Hint{{Command: "sudo reboot"}}},
		"exactly one node":               {Sections: []page.Section{section(page.Item{Row: &ok, Note: &page.Note{}})}},
		"3 levels at most":               {Sections: []page.Section{section(nest(4))}},
		"more cells than columns":        {Sections: []page.Section{section(row(page.Row{ID: "a", Mark: page.OK, Cells: []page.Value{page.None(), page.None()}}))}},
		"labeled sizes of its column's":  {Sections: []page.Section{section(row(page.Row{ID: "a", Mark: page.OK, Cells: []page.Value{page.Levels(page.Labeled("kept", page.Size(1)))}}))}},
		"an error carries a hint":        {Sections: []page.Section{section(page.Item{Error: &page.Error{Message: "x"}})}},
		"a judged section's row carries": {Sections: []page.Section{section(row(page.Row{ID: "b"}))}},
	}
	for want, p := range cases {
		err := page.Check(p)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("want a refusal saying %q, got %v", want, err)
		}
	}
	page.Strict(true)
	defer page.Strict(false)
	defer func() {
		if recover() == nil {
			t.Error("a strict Seal kept a page it refuses")
		}
	}()
	page.Seal(cases["carries a mark"])
}

func nest(depth int) page.Item {
	g := &page.Group{Head: page.Row{ID: "g" + strconv.Itoa(depth), Mark: page.OK}}
	if depth > 1 {
		g.Items = []page.Item{nest(depth - 1)}
	}
	return page.Item{Group: g}
}

func TestSealCountsTheMarksAndSetsTheState(t *testing.T) {
	p := page.Seal(page.Page{Sections: []page.Section{section(
		row(page.Row{ID: "a", Mark: page.OK}),
		page.Item{Group: &page.Group{Head: page.Row{ID: "g", Mark: page.Absent}, Items: []page.Item{row(page.Row{ID: "b", Mark: page.Awaiting})}}},
		page.Item{Fact: &page.Fact{Key: "k", Value: page.Text("v")}},
	)}})
	got := p.Sections[0].Counts
	want := []page.Tally{{Mark: page.OK, N: 1}, {Mark: page.Absent, N: 1}, {Mark: page.Awaiting, N: 1}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("counts %v, want %v", got, want)
	}
	if p.State != page.Loading {
		t.Errorf("a page with a pending row is %s", p.State)
	}
	if p := page.Seal(page.Page{Error: &page.Error{Message: "x", Hint: "y"}}); p.State != page.Failed {
		t.Errorf("a page with an error is %s", p.State)
	}
	if p := page.Seal(page.Page{}); p.State != page.Ready {
		t.Errorf("an empty page is %s", p.State)
	}
}

func TestTheOnLineOfAnotherMachineNeedsNoTitle(t *testing.T) {
	on := page.Section{ID: "on", Role: page.Banner, Items: []page.Item{{Note: &page.Note{Text: "On infra:"}}}}
	if err := page.Check(page.Page{Sections: []page.Section{on}}); err != nil {
		t.Errorf("Check(the On line) = %v; want it kept untitled", err)
	}
	on.Role = ""
	if err := page.Check(page.Page{Sections: []page.Section{on}}); err == nil {
		t.Error("Check kept an untitled section that is no banner")
	}
}
