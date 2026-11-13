package page

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

var strict bool

func Strict(on bool) { strict = on }

func Seal(p Page) Page {
	pending := false
	for i := range p.Sections {
		counts := map[Mark]int{}
		waits := false
		for _, it := range p.Sections[i].Items {
			waits = tally(it, counts) || waits
		}
		pending = pending || waits
		// value: an uncounted section keeps the counts it was given: none, or
		// those of the larger set it sums up (Home's Tools and Config).
		if p.Sections[i].Uncounted && !waits {
			continue
		}
		p.Sections[i].Counts = nil
		if p.Sections[i].Role == Summary {
			continue
		}
		for _, m := range Marks() {
			if counts[m] > 0 {
				p.Sections[i].Counts = append(p.Sections[i].Counts, Tally{Mark: m, N: counts[m]})
			}
		}
	}
	switch {
	case p.Error != nil:
		p.State = Failed
	case pending:
		p.State = Loading
	default:
		p.State = Ready
	}
	if err := Check(p); err != nil && strict {
		panic(err)
	}
	secret.TripwireJSON(p)
	return p
}

func Tallied(items []Item) []Tally {
	// value: the tally of a set of items: what Seal gives a section, for a
	// section that shows only some of the rows it sums up.
	counts := map[Mark]int{}
	for _, it := range items {
		tally(it, counts)
	}
	var out []Tally
	for _, m := range Marks() {
		if counts[m] > 0 {
			out = append(out, Tally{Mark: m, N: counts[m]})
		}
	}
	return out
}

func tally(it Item, counts map[Mark]int) (pending bool) {
	switch {
	case it.Row != nil:
		counts[it.Row.Mark]++
		return rowPending(*it.Row)
	case it.Group != nil:
		counts[it.Group.Head.Mark]++
		pending = rowPending(it.Group.Head)
		for _, child := range it.Group.Items {
			pending = tally(child, counts) || pending
		}
		return pending
	case it.Fact != nil:
		return it.Fact.Value.Kind == KindPending
	}
	return false
}

func rowPending(r Row) bool {
	if r.Mark == Awaiting || r.Detail.Kind == KindPending {
		return true
	}
	for _, c := range r.Cells {
		if c.Kind == KindPending {
			return true
		}
	}
	return false
}

// value: a banner that is a page's own first lines needs no title: the
// `On <machine>:` line a page of another machine starts with (SPEC §8.1), an
// Overview's section lines, Awake's Macs (vision 07).
func onLine(s Section) bool {
	return s.Role == Banner && len(s.Items) > 0
}

func Check(p Page) error {
	ids := map[string]bool{}
	for _, s := range p.Sections {
		if s.ID == "" || ids[s.ID] {
			return errors.New("section " + strconv.Quote(s.ID) + ": an ID unique in the page is required")
		}
		ids[s.ID] = true
		if s.Title == "" && !onLine(s) {
			return errors.New("section " + s.ID + ": a title is required")
		}
		judged := len(s.Columns) == 0 || s.Columns[0].Kind != KindListing
		rows := map[string]bool{}
		for _, it := range s.Items {
			if err := checkItem(s, it, judged, rows, 1); err != nil {
				return errors.New("section " + s.ID + ": " + err.Error())
			}
		}
	}
	for _, h := range p.Hints {
		if err := checkHint(&h); err != nil {
			return err
		}
	}
	return checkPanel(p)
}

func checkItem(s Section, it Item, judged bool, rows map[string]bool, depth int) error {
	set := 0
	for _, ok := range []bool{it.Row != nil, it.Fact != nil, it.Group != nil, it.Empty != nil, it.Error != nil, it.Note != nil, it.Link != nil,
		it.Tile != nil, it.Gauge != nil, it.Chart != nil, it.Message != nil, it.Steps != nil, it.Log != nil, it.Uptime != nil} {
		if ok {
			set++
		}
	}
	if set != 1 {
		return errors.New("an entry of a section holds exactly one node, not " + strconv.Itoa(set))
	}
	switch {
	case it.Row != nil:
		return checkRow(s, *it.Row, judged, rows)
	case it.Group != nil:
		if depth > 3 {
			return errors.New("a group nests 3 levels at most")
		}
		if err := checkRow(s, it.Group.Head, judged && !it.Group.Heading(), rows); err != nil {
			return err
		}
		for _, child := range it.Group.Items {
			if err := checkItem(s, child, judged, rows, depth+1); err != nil {
				return err
			}
		}
	case it.Empty != nil:
		return checkHint(it.Empty.Hint)
	case it.Error != nil:
		if it.Error.Hint == "" {
			return errors.New("an error carries a hint")
		}
	}
	return nil
}

func checkRow(s Section, r Row, judged bool, rows map[string]bool) error {
	if r.ID == "" || rows[r.ID] {
		return errors.New("row " + strconv.Quote(r.ID) + ": an ID unique in its section is required")
	}
	rows[r.ID] = true
	if judged && s.Role != Summary && r.Mark == Unjudged {
		return errors.New("row " + r.ID + ": a judged section's row carries a mark")
	}
	if s.Role != Summary && (r.Mark == Attention || r.Mark == Broken) && (r.Reason == "" || !explained(r)) {
		return errors.New("row " + r.ID + ": an attention or broken row carries a reason and a detail or a hint")
	}
	if len(s.Columns) > 0 && len(r.Cells) > len(s.Columns)-1 {
		return errors.New("row " + r.ID + ": more cells than columns")
	}
	for i, c := range r.Cells {
		if c.Kind != KindLevels {
			continue
		}
		for _, l := range c.Items {
			if l.Kind != KindSize || !contains(s.Columns[i+1].Levels, l.Label) {
				return errors.New("row " + r.ID + ": a levels cell holds only labeled sizes of its column's levels")
			}
		}
	}
	return checkHint(r.Hint)
}

func explained(r Row) bool {
	if !r.Detail.IsNone() || r.Hint != nil {
		return true
	}
	for _, c := range r.Cells {
		if c.Kind == KindText && c.Text != "" {
			return true
		}
	}
	return false
}

func checkHint(h *Hint) error {
	if h == nil {
		return nil
	}
	if h.Command == "" {
		return errors.New("a hint names its command")
	}
	if !isStation(h.Command) && !h.External {
		return errors.New("hint " + strconv.Quote(h.Command) + " is not station's: it is External")
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

type names struct {
	actions  map[string]bool
	live     map[string]bool
	segments map[string]bool
}

func checkPanel(p Page) error {
	n := names{actions: map[string]bool{}, live: map[string]bool{}, segments: map[string]bool{}}
	for _, k := range p.Live {
		n.live[k] = true
	}
	for _, s := range p.Segments {
		n.segments[s.ID] = true
	}
	for _, a := range p.Actions {
		n.actions[a.ID] = a.ID != ""
	}
	for _, s := range p.Sections {
		for _, a := range s.Picker {
			n.actions[a.ID] = a.ID != ""
		}
	}
	if err := walkValues(reflect.ValueOf(p), n); err != nil {
		return err
	}
	if p.Header != nil {
		if err := n.named("the header's button", p.Header.Buttons...); err != nil {
			return err
		}
	}
	if p.Footer != nil {
		if err := n.named("the footer's button", p.Footer.Buttons...); err != nil {
			return err
		}
	}
	for _, a := range p.Actions {
		if err := checkAction(a, n); err != nil {
			return err
		}
	}
	for _, s := range p.Sections {
		if s.Segment != "" && !n.segments[s.Segment] {
			return errors.New("section " + s.ID + ": its segment " + strconv.Quote(s.Segment) + " is no segment of the page")
		}
		for _, a := range s.Picker {
			if err := checkAction(a, n); err != nil {
				return errors.New("section " + s.ID + ": " + err.Error())
			}
		}
		for _, it := range s.Items {
			if err := checkPanelItem(s, it, n); err != nil {
				return errors.New("section " + s.ID + ": " + err.Error())
			}
		}
	}
	return nil
}

func checkPanelItem(s Section, it Item, n names) error {
	switch {
	case it.Row != nil:
		return checkPanelRow(s, *it.Row, n)
	case it.Group != nil:
		if err := checkPanelRow(s, it.Group.Head, n); err != nil {
			return err
		}
		for _, child := range it.Group.Items {
			if err := checkPanelItem(s, child, n); err != nil {
				return err
			}
		}
	case it.Empty != nil:
		return n.named("the empty card's action", it.Empty.Actions...)
	case it.Tile != nil:
		if it.Tile.Number.IsNone() {
			return errors.New("a tile carries its number")
		}
		if it.Tile.Toggle != "" {
			return n.named("the tile's toggle", it.Tile.Toggle)
		}
	case it.Log != nil:
		if it.Log.Cause < 0 || it.Log.Cause > len(it.Log.Lines) {
			return errors.New("a log's cause is one of its lines")
		}
		if it.Log.More != "" {
			return n.named("the log's more", it.Log.More)
		}
	}
	return nil
}

func checkPanelRow(s Section, r Row, n names) error {
	for _, ref := range []struct{ what, id string }{{"verb", r.Verb}, {"quick slot", r.Quick}} {
		if ref.id != "" && !n.actions[ref.id] {
			return errors.New("row " + r.ID + ": its " + ref.what + " " + strconv.Quote(ref.id) + " is no action of the page")
		}
	}
	for _, id := range r.Menu {
		if id != "-" && !n.actions[id] {
			return errors.New("row " + r.ID + ": its menu's " + strconv.Quote(id) + " is no action of the page")
		}
	}
	if r.Trail != "" && !slices.ContainsFunc(s.Columns, func(c Column) bool { return c.Key == r.Trail }) {
		return errors.New("row " + r.ID + ": its trail " + strconv.Quote(r.Trail) + " is no column of the section")
	}
	return nil
}

func checkAction(a Action, n names) error {
	if a.Sheet != "" && !strings.HasPrefix(a.Sheet, SheetHint) && a.Form == nil {
		return errors.New("action " + a.ID + ": a sheet action carries its form")
	}
	if a.Destructive && a.Confirm == nil {
		return errors.New("action " + a.ID + ": a destructive action carries a confirm")
	}
	if err := n.named("action "+a.ID+"'s submenu item", slices.DeleteFunc(slices.Clone(a.Submenu), func(id string) bool { return id == "-" })...); err != nil {
		return err
	}
	if a.Form != nil {
		if err := checkFields(a.Form.Fields); err != nil {
			return errors.New("action " + a.ID + ": " + err.Error())
		}
		return n.named("action "+a.ID+"'s split item", a.Form.Split...)
	}
	return nil
}

func checkFields(fields []Field) error {
	for _, f := range fields {
		if err := checkHint(f.Hint); err != nil {
			return err
		}
		if f.Kind == FieldCommand && f.Hint == nil {
			return errors.New("field " + f.Key + ": a command field carries its hint")
		}
		if err := checkFields(f.Fields); err != nil {
			return err
		}
	}
	return nil
}

func (n names) named(what string, ids ...string) error {
	for _, id := range ids {
		if !n.actions[id] {
			return errors.New(what + " " + strconv.Quote(id) + " is no action of the page")
		}
	}
	return nil
}

func walkValues(v reflect.Value, n names) error {
	k := v.Kind()
	if k == reflect.Pointer || k == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		return walkValues(v.Elem(), n)
	}
	if k == reflect.Slice || k == reflect.Array {
		for i := range v.Len() {
			if err := walkValues(v.Index(i), n); err != nil {
				return err
			}
		}
		return nil
	}
	if k != reflect.Struct {
		return nil
	}
	switch x := v.Interface().(type) {
	case Value:
		if err := checkValue(x, n); err != nil {
			return err
		}
	case Glyph:
		if x.Ring != "" && !n.live[x.Ring] {
			return errors.New("the glyph's ring " + strconv.Quote(x.Ring) + " is no live key of the page")
		}
	}
	for i := range v.NumField() {
		if err := walkValues(v.Field(i), n); err != nil {
			return err
		}
	}
	return nil
}

func checkValue(v Value, n names) error {
	if v.Live != "" && !n.live[v.Live] {
		return errors.New("the value's live key " + strconv.Quote(v.Live) + " is no live key of the page")
	}
	if v.Short != "" && v.Pri == 0 {
		return errors.New("a value with a short wording carries its priority")
	}
	return nil
}
