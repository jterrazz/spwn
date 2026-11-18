package page_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/page"
)

var update = flag.Bool("update", false, "rewrite testdata/every-field.json from everyField")

const everyFieldPath = "testdata/every-field.json"

func panel(p page.Page) page.Page {
	p.Live = append(p.Live, "cpu:linux-1")
	p.Actions = append(p.Actions, page.Action{ID: "copy", Label: "Copy", Run: &page.Run{Copy: "linux-1.local"}})
	return p
}

func panelSection(items ...page.Item) page.Section {
	return page.Section{ID: "s", Title: "S", Columns: cols(), Items: items}
}

func TestCheckRefusesWhatThePanelCannotDraw(t *testing.T) {
	marked := func(r page.Row) page.Item {
		r.ID, r.Mark = "a", page.OK
		return row(r)
	}
	form := &page.Form{Fields: []page.Field{{Key: "name"}}}
	cases := map[string]page.Page{
		"its verb \"stop\" is no action":               panel(page.Page{Sections: []page.Section{panelSection(marked(page.Row{Verb: "stop"}))}}),
		"its quick slot \"stop\" is no action":         panel(page.Page{Sections: []page.Section{panelSection(marked(page.Row{Quick: "stop"}))}}),
		"its menu's \"stop\" is no action":             panel(page.Page{Sections: []page.Section{panelSection(marked(page.Row{Menu: []string{"copy", "-", "stop"}}))}}),
		"row g: its verb \"stop\" is no action":        panel(page.Page{Sections: []page.Section{panelSection(page.Item{Group: &page.Group{Head: page.Row{ID: "g", Mark: page.OK, Verb: "stop"}, Items: []page.Item{marked(page.Row{})}}})}}),
		"its trail \"cpu\" is no column":               panel(page.Page{Sections: []page.Section{panelSection(marked(page.Row{Trail: "cpu"}))}}),
		"live key \"cpu:infra\" is no live key":        panel(page.Page{Sections: []page.Section{panelSection(marked(page.Row{Cells: []page.Value{{Kind: page.KindPercent, Live: "cpu:infra"}}}))}}),
		"ring \"cpu:mac-mini\" is no live key":         panel(page.Page{Header: &page.Header{Glyph: page.Glyph{Kind: "desktop", Ring: "cpu:mac-mini"}}}),
		"a short wording carries its priority":         panel(page.Page{Sections: []page.Section{{ID: "s", Title: "S", Role: page.Summary, Caption: []page.Value{{Kind: page.KindText, Text: "on mac-mini", Short: "mini"}}}}}),
		"a tile carries its number":                    panel(page.Page{Sections: []page.Section{{ID: "s", Title: "S", Card: page.CardTiles, Items: []page.Item{{Tile: &page.Tile{Caption: "VMs"}}}}}}),
		"a sheet action carries its form":              panel(page.Page{Actions: []page.Action{{ID: "new-vm", Sheet: page.SheetNewVM}}}),
		"a destructive action carries a confirm":       panel(page.Page{Actions: []page.Action{{ID: "delete", Destructive: true}}}),
		"button \"attach\" is no action":               panel(page.Page{Footer: &page.Footer{Kind: page.FooterSSH, Buttons: []string{"attach"}}}),
		"the header's button \"menu\" is no action":    panel(page.Page{Header: &page.Header{Buttons: []string{"menu"}}}),
		"submenu item \"share\" is no action":          panel(page.Page{Actions: []page.Action{{ID: "more", Submenu: []string{"copy", "share"}}}}),
		"split item \"create\" is no action":           panel(page.Page{Actions: []page.Action{{ID: "new", Sheet: page.SheetNewVM, Form: &page.Form{Fields: form.Fields, Split: []string{"create"}}}}}),
		"its segment \"ports\" is no segment":          panel(page.Page{Segments: []page.Segment{{ID: "overview"}}, Sections: []page.Section{{ID: "s", Title: "S", Segment: "ports"}}}),
		"the empty card's action \"new\" is no action": panel(page.Page{Sections: []page.Section{{ID: "s", Title: "S", Items: []page.Item{{Empty: &page.Empty{Text: "No VMs yet", Actions: []string{"new"}}}}}}}),
		"a log's cause is one of its lines":            panel(page.Page{Sections: []page.Section{{ID: "s", Title: "S", Items: []page.Item{{Log: &page.Log{Lines: []string{"x"}, Cause: 2}}}}}}),
		"exactly one node":                             panel(page.Page{Sections: []page.Section{{ID: "s", Title: "S", Items: []page.Item{{Gauge: &page.Gauge{}, Chart: &page.Chart{}}}}}}),
	}
	for want, p := range cases {
		err := page.Check(p)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("want a refusal saying %q, got %v", want, err)
		}
	}
	if err := page.Check(panel(page.Page{Actions: []page.Action{{ID: "why", Sheet: page.SheetHint + "full-disk-access"}}})); err != nil {
		t.Errorf("a hint sheet shows its hint and asks nothing, yet Check refused it: %v", err)
	}
	picked := page.Section{ID: "w", Title: "Window", Picker: []page.Action{{ID: "window-6m", Label: "6 min", Run: &page.Run{Call: "activity.window"}}}}
	if err := page.Check(panel(page.Page{Sections: []page.Section{picked, panelSection(marked(page.Row{Quick: "window-6m"}))}})); err != nil {
		t.Errorf("a row may name a section picker's action, yet Check refused it: %v", err)
	}
}

func TestAPageThatSetsEveryFieldSealsAndChecks(t *testing.T) {
	page.Strict(true)
	defer page.Strict(false)
	p := page.Seal(everyField())
	if missing := unset(reflect.ValueOf(p)); len(missing) > 0 {
		t.Errorf("the page leaves fields unset, so it proves nothing of them: %v", missing)
	}
	body, err := json.MarshalIndent(page.Document{Schema: page.Schema, Page: p}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	if *update {
		if werr := os.WriteFile(everyFieldPath, body, 0o600); werr != nil {
			t.Fatal(werr)
		}
	}
	golden, err := os.ReadFile(everyFieldPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(golden, body) {
		t.Errorf("%s is not everyField's page: run go test ./apps/engine/page -update", everyFieldPath)
	}
	back, err := page.Decode(golden)
	if err != nil || !reflect.DeepEqual(back, p) {
		t.Errorf("the page does not survive its own JSON: %v", err)
	}
}

func unset(root reflect.Value) []string {
	seen := map[reflect.Type]map[int]bool{}
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		k := v.Kind()
		if k == reflect.Pointer && !v.IsNil() {
			walk(v.Elem())
		}
		if k == reflect.Slice || k == reflect.Array {
			for i := range v.Len() {
				walk(v.Index(i))
			}
		}
		if k != reflect.Struct {
			return
		}
		if seen[v.Type()] == nil {
			seen[v.Type()] = map[int]bool{}
		}
		for i := range v.NumField() {
			if !v.Field(i).IsZero() {
				seen[v.Type()][i] = true
			}
			walk(v.Field(i))
		}
	}
	walk(root)
	reached := map[reflect.Type]bool{}
	var reach func(t reflect.Type)
	reach = func(t reflect.Type) {
		k := t.Kind()
		if k == reflect.Pointer || k == reflect.Slice || k == reflect.Array || k == reflect.Map {
			reach(t.Elem())
		}
		if k != reflect.Struct || reached[t] {
			return
		}
		reached[t] = true
		if seen[t] == nil {
			seen[t] = map[int]bool{}
		}
		for i := range t.NumField() {
			reach(t.Field(i).Type)
		}
	}
	reach(root.Type())
	var missing []string
	for typ, fields := range seen {
		for i := range typ.NumField() {
			if !fields[i] {
				missing = append(missing, typ.Name()+"."+typ.Field(i).Name)
			}
		}
	}
	slices.Sort(missing)
	return missing
}
