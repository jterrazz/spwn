package wiki

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

type Tier string

const (
	TierLead   Tier = "lead"
	TierNormal Tier = "normal"
	TierQuiet  Tier = "quiet"
)

type Weight struct {
	Score int  `json:"score"`
	Tier  Tier `json:"tier"`
	// value: the tier its author wrote, when one is written.
	By  Tier    `json:"by,omitempty"`
	Why []Cause `json:"why"`
}

type Cause struct {
	Name   string
	Points int
}

// invariant: a table finds its rows, never holds them: the blocks of kind
// `of` inside its scope `in` that match `where`, in reading order. Its kind
// is a kind (H19), each column a field of it or derived (H20), its condition
// on a field of it, `has` on a list, a state a value of the target's state
// enum (H36).
func (k *checker) table(p *Page, b *Block) {
	of := textOf(b.Props, "of")
	kind, ok := k.schema.kind(of)
	if !ok || kind.Structural {
		k.add("H19", p.ID, b.ID, b.Props.at("of", b.At), fmt.Sprintf("%q is no kind: a table shows the blocks of one kind", of))
		return
	}
	cols, _ := b.Props.Get("columns")
	for _, c := range cols.List {
		name := text(c)
		if _, field := kind.field(name); !field && !slices.Contains(derivedColumns(), name) {
			k.add("H20", p.ID, b.ID, whereOf(c, b.Props.at("columns", b.At)), fmt.Sprintf("column %q is neither a field of %s (%s) nor derived (%s)", name, of, strings.Join(kind.fieldNames(), ", "), strings.Join(derivedColumns(), ", ")))
		}
	}
	if len(cols.List) > tableCols {
		k.add("S17", p.ID, b.ID, b.At, fmt.Sprintf("table %q has %d columns", b.ID, len(cols.List)))
	}
	v, hasWhere := b.Props.Get("where")
	var q *Query
	if hasWhere && v.Kind == ValueQuery {
		q = v.Query
		if !k.condition(p, b, kind, q, b.Props.at("where", b.At)) {
			q = nil
		}
	}
	scope := textOf(b.Props, "in")
	var found, rows []string
	for _, h := range k.inScope(scope) {
		if h.block.Kind != of {
			continue
		}
		found = append(found, h.block.Address)
		if q == nil && hasWhere || q != nil && !k.matches(q, h.block) {
			continue
		}
		rows = append(rows, h.block.Address)
	}
	if _, resolves := k.targets[scope]; resolves && len(found) == 0 {
		k.add("S16", p.ID, b.ID, b.At, fmt.Sprintf("no %s under %s: the table is empty, not wrong", of, scope))
	}
	k.rows[b] = rows
}

func (k *checker) inScope(address string) []held {
	t, ok := k.targets[address]
	switch {
	case !ok:
		return nil
	case t.block != nil:
		return []held{{block: t.block}}
	case t.group != nil:
		var out []held
		for _, h := range k.blocks[t.page.ID] {
			if h.group == t.group {
				out = append(out, h)
			}
		}
		return out
	default:
		return k.blocks[t.page.ID]
	}
}

func (k *checker) condition(p *Page, b *Block, kind Kind, q *Query, at At) bool {
	f, ok := kind.field(q.Field)
	_, list := listOf(f.Type)
	switch {
	case !ok:
		k.add("H36", p.ID, b.ID, at, fmt.Sprintf("where: %q is no field of %s (%s)", q.Field, kind.ID, strings.Join(kind.fieldNames(), ", ")))
		return false
	case q.Op == OpHas && !list:
		k.add("H36", p.ID, b.ID, at, fmt.Sprintf("where: %s is no list, has tests a list's members: write is", q.Field))
		return false
	case q.Op == OpIs && list:
		k.add("H36", p.ID, b.ID, at, fmt.Sprintf("where: %s is a list, is compares one value: write has", q.Field))
		return false
	}
	if q.Operand.Kind != ValueState {
		return true
	}
	to, _ := k.schema.kind(f.To)
	state, hasState := to.field(to.State)
	switch {
	case f.To == "" || to.State == "" || !hasState:
		k.add("H36", p.ID, b.ID, at, fmt.Sprintf("where: state(%q) reads the state of %s's target, and %s points to no kind with a state", q.Operand.Text, q.Field, q.Field))
		return false
	case !slices.Contains(state.Values, q.Operand.Text):
		k.add("H36", p.ID, b.ID, at, fmt.Sprintf("%q is no state of %s: %s", q.Operand.Text, to.ID, orList(state.Values)))
		return false
	}
	return true
}

func (k *checker) matches(q *Query, b *Block) bool {
	v, ok := b.Props.Get(q.Field)
	if !ok {
		return false
	}
	if q.Op == OpHas {
		return slices.ContainsFunc(v.List, func(e Value) bool { return k.equals(q.Operand, e) })
	}
	return k.equals(q.Operand, v)
}

func (k *checker) equals(operand, v Value) bool {
	switch operand.Kind {
	case ValueRef:
		return v.Kind == ValueRef && v.Text == operand.Text
	case ValueState:
		t, ok := k.targets[v.Text]
		if v.Kind != ValueRef || !ok || t.block == nil {
			return false
		}
		kind, _ := k.schema.kind(t.block.Kind)
		return kind.State != "" && textOf(t.block.Props, kind.State) == operand.Text
	default:
		return v.Kind != ValueRef && text(v) == operand.Text
	}
}

func (k *checker) schemaRules() {
	owned := map[string]bool{}
	for _, p := range k.pages {
		owned[k.schema.seatOf(p.ID)] = true
	}
	read := map[Intent]bool{}
	for _, seat := range k.schema.Seats {
		if _, ok := k.targets[seat.Home]; !ok {
			says := fmt.Sprintf("seat %q opens on %q, which is no page", seat.ID, seat.Home)
			if near := nearest(seat.Home, k.addresses); near != "" {
				says += fmt.Sprintf(" (did you mean %s?)", near)
			}
			k.addSchema("H4", seat.Line, says)
		}
		if !owned[seat.ID] {
			k.addSchema("H33", seat.Line, fmt.Sprintf("seat %q owns no page: a seat opens the day it has content", seat.ID))
		}
		if !k.schema.iconExists(seat.ID) {
			k.addSchema("S21", seat.Line, fmt.Sprintf("seat %q has no icon: the dot stands in", seat.ID))
		}
		for _, i := range seat.Reads {
			read[i] = true
		}
	}
	for _, i := range intents() {
		if !read[i] && len(k.schema.Seats) > 0 {
			k.addSchema("S22", k.schema.Seats[0].Line, fmt.Sprintf("intent %q is read by no seat", i))
		}
	}
	ids := make([]string, 0, len(k.schema.Kinds))
	for id := range k.schema.Kinds {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		kind := k.schema.Kinds[id]
		switch {
		case kind.Icon == "":
			k.addSchema("S21", kind.Line, fmt.Sprintf("kind %q has no icon: the dot stands in", id))
		case !k.schema.iconExists(kind.Icon):
			k.addSchema("H29", kind.Line, fmt.Sprintf("icon %q is in no icons: of wiki.yaml and is not built in", kind.Icon))
		}
	}
	k.grids()
}

// invariant: a grid is rows of # and ., 12 by 12 at most (H30), and two
// icons never share one (H31).
func (k *checker) grids() {
	seen := map[string]string{}
	for _, icon := range k.schema.Icons {
		width := 0
		for i, row := range icon.Rows {
			width = max(width, len([]rune(row)))
			if bad := strings.Trim(row, "#."); bad != "" {
				first, _ := utf8.DecodeRuneInString(bad)
				k.addSchema("H30", icon.Line, fmt.Sprintf("icon %q: row %d holds %q; only # and .", icon.Name, i+1, string(first)))
			}
		}
		if width > gridSide || len(icon.Rows) > gridSide {
			k.addSchema("H30", icon.Line, fmt.Sprintf("icon %q is %d by %d, %d by %d at most", icon.Name, width, len(icon.Rows), gridSide, gridSide))
		}
		grid := strings.Join(icon.Rows, "\n")
		if first, ok := seen[grid]; ok {
			k.addSchema("H31", icon.Line, fmt.Sprintf("icons %q and %q are the same grid", first, icon.Name))
			continue
		}
		seen[grid] = icon.Name
	}
}

// value: a block's weight and what makes it, SPEC-v1 §7; nil for a kind the
// engine does not weigh.
func (k *checker) weightOf(b *Block, leans, illustrated map[string]bool, linkers map[string][]string, broken map[string]bool) *Weight {
	kind, _ := k.schema.kind(b.Kind)
	if kind.Structural {
		return nil
	}
	w := &Weight{Why: []Cause{}}
	cause := func(name string, points int) {
		if points > 0 {
			w.Why = append(w.Why, Cause{Name: name, Points: points})
			w.Score += points
		}
	}
	if leans[b.ID] {
		cause("prose-leans", 2)
	}
	if illustrated[b.Address] {
		cause("illustrated", 3)
	}
	cause("linked", min(len(linkers[b.Address]), 2))
	switch b.Kind {
	case "rule":
		if force := textOf(b.Props, "force"); force == "soft" {
			cause("guard-soft", 1)
		} else if force != "" {
			cause("guard-hard", 2)
		}
	case "claim":
		if textOf(b.Props, "certainty") == "decided" {
			cause("decided-claim", 2)
		}
	case "decision":
		cause("decision", 1)
	case "scenario":
		cause("scenario", 1)
	default:
		cause("kind", kind.Weighs)
	}
	if slices.ContainsFunc(anchoringFields(), func(f string) bool { _, ok := b.Props.Get(f); return ok }) {
		cause("anchored", 1)
	}
	if k.needsYou[b.Address] {
		cause("needs-you", 3)
	}
	if broken[b.ID] {
		cause("broken", 3)
	}
	if by, ok := named([]Tier{TierLead, TierQuiet}, textOf(b.Props, "weight")); ok {
		w.By = by
	}
	return w
}

// value: the weighed blocks of one set: a group's, or a section's on one floor.
type weighed struct {
	label  string
	group  *Group
	blocks []*Block
}

func (k *checker) weigh(p *Page) Reading {
	broken := map[string]bool{}
	for _, f := range k.findings {
		if f.Hard && f.Page == p.ID && f.Block != "" {
			broken[f.Block] = true
		}
	}
	leans := map[string]bool{}
	forProse(p, func(pr *Prose) {
		for _, m := range fragmentLink.FindAllStringSubmatch(pr.Markdown, -1) {
			leans[m[1]] = true
		}
	})
	illustrated, by := map[string]bool{}, map[*Block]string{}
	for _, h := range k.blocks[p.ID] {
		if h.block.Kind == "figure" {
			shows := textOf(h.block.Props, "shows")
			illustrated[shows], by[h.block] = true, shows
		}
	}
	linkers := map[string][]string{}
	for _, h := range k.blocks[p.ID] {
		for _, prop := range h.block.Props {
			for _, r := range refsIn(prop.Value) {
				if r.Text != h.block.Address && by[h.block] != r.Text && !slices.Contains(linkers[r.Text], h.block.Address) {
					linkers[r.Text] = append(linkers[r.Text], h.block.Address)
				}
			}
		}
	}
	weights := map[*Block]*Weight{}
	var sets []*weighed
	index := map[string]*weighed{}
	for _, h := range k.blocks[p.ID] {
		w := k.weightOf(h.block, leans, illustrated, linkers, broken)
		if w == nil {
			continue
		}
		weights[h.block] = w
		key, label := fmt.Sprintf("%d/%d", h.section, k.blockFloor(p, h.block)), "section"
		if h.group != nil {
			key, label = "#"+h.group.ID, "group"
		}
		set, ok := index[key]
		if !ok {
			set = &weighed{label: label, group: h.group}
			if h.group == nil {
				set.label = sectionName(p, h.section)
			}
			index[key] = set
			sets = append(sets, set)
		}
		set.blocks = append(set.blocks, h.block)
	}
	leads := map[*Group]string{}
	for _, set := range sets {
		if lead := k.tiers(p, set, weights); lead != nil && set.group != nil {
			leads[set.group] = lead.Address
		}
	}
	k.budget(p, weights)
	return k.reading(p, weights, leads)
}

func sectionName(p *Page, section int) string {
	if h := p.Sections[section].Heading; h != nil {
		return fmt.Sprintf("section %q", h.Text)
	}
	return "the page's opening"
}

// invariant: one set's tiers: the written lead, else the highest score at 3
// or more, the first in reading order on a tie; quiet when written so or
// scored 1 or less; normal the rest. A second written lead is refused (H16).
func (k *checker) tiers(p *Page, set *weighed, weights map[*Block]*Weight) *Block {
	var lead *Block
	top := 0
	for _, b := range set.blocks {
		w := weights[b]
		top = max(top, w.Score)
		if w.By != TierLead {
			continue
		}
		if lead != nil {
			name := set.label
			if set.group != nil {
				name = fmt.Sprintf("group %q", set.group.ID)
			}
			k.add("H16", p.ID, b.ID, b.Props.at("weight", b.At), fmt.Sprintf("%s already has a written lead, %q: one per group; write quiet, or nothing", name, lead.ID))
			continue
		}
		lead = b
	}
	if lead == nil && top >= 3 {
		for _, b := range set.blocks {
			if w := weights[b]; w.Score == top && w.By != TierQuiet {
				lead = b
				break
			}
		}
	}
	for _, b := range set.blocks {
		w := weights[b]
		switch {
		case b == lead:
			w.Tier = TierLead
		case w.By == TierQuiet || w.Score <= 1:
			w.Tier = TierQuiet
		default:
			w.Tier = TierNormal
		}
		switch {
		case w.By == TierLead && top-w.Score >= 3:
			k.add("S12", p.ID, b.ID, b.Props.at("weight", b.At), fmt.Sprintf("%q is written lead, its score is %d and its %s's top is %d", b.ID, w.Score, set.labelWord(), top))
		case w.By == TierQuiet && w.Score >= 4:
			k.add("S12", p.ID, b.ID, b.Props.at("weight", b.At), fmt.Sprintf("%q is written quiet, its score is %d", b.ID, w.Score))
		}
	}
	return lead
}

func (s *weighed) labelWord() string {
	if s.group != nil {
		return "group"
	}
	return "section"
}

// invariant: at the corpus budget, a page shows at most seven line Faces
// (S11), counted by SPEC-v1 §15's table for the page's seat: at 10 min a
// quiet block is one line, at 2 min a normal one, at 30 s none; a group the
// seat does not read is one line and hides its blocks, and at 30 s every
// group is one line. A page of more than seven weighed blocks has a lead (S13).
func (k *checker) budget(p *Page, weights map[*Block]*Weight) {
	var reads []Intent
	seat := k.schema.seatOf(p.ID)
	for _, s := range k.schema.Seats {
		if s.ID == seat {
			reads = s.Reads
		}
	}
	unread := func(g *Group) bool { return g != nil && seat != "" && !slices.Contains(reads, g.Intent) }
	lines := 0
	for _, g := range k.groups[p.ID] {
		if unread(g.group) || k.schema.Budget == Budget30s {
			lines++
		}
	}
	line := map[Budget]Tier{Budget10min: TierQuiet, Budget2min: TierNormal}[k.schema.Budget]
	led := false
	for _, h := range k.blocks[p.ID] {
		w := weights[h.block]
		if w == nil {
			continue
		}
		led = led || w.Tier == TierLead
		if !unread(h.group) && line != "" && w.Tier == line {
			lines++
		}
	}
	if lines > pageLines {
		k.add("S11", p.ID, "", pageAt(p), fmt.Sprintf("page %q does not fit %s: %d lines", p.ID, k.schema.Budget, lines))
	}
	if n := len(weights); n > pageBlocks && !led {
		k.add("S13", p.ID, "", pageAt(p), fmt.Sprintf("page %q: %d blocks, none leads", p.ID, n))
	}
}

func (k *checker) reading(p *Page, weights map[*Block]*Weight, leads map[*Group]string) Reading {
	r := Reading{Page: p.ID, Seat: k.schema.seatOf(p.ID), Floor: k.floors[p.ID], Blocks: []BlockReading{}}
	entry := func(b *Block) BlockReading {
		return BlockReading{Address: b.Address, Kind: b.Kind, Floor: k.blockFloor(p, b), Weight: weights[b], Rows: k.rows[b]}
	}
	for _, sec := range p.Sections {
		for _, it := range sec.Items {
			switch it := it.(type) {
			case *Block:
				r.Blocks = append(r.Blocks, entry(it))
			case *Group:
				g := BlockReading{Address: p.ID + "#" + it.ID, Kind: "group", Floor: k.floors[p.ID], Lead: leads[it]}
				for _, s := range k.schema.Seats {
					if slices.Contains(s.Reads, it.Intent) {
						g.ReadBy = append(g.ReadBy, s.ID)
					}
				}
				r.Blocks = append(r.Blocks, g)
				for _, inner := range it.Items {
					if b, ok := inner.(*Block); ok {
						r.Blocks = append(r.Blocks, entry(b))
					}
				}
			}
		}
	}
	return r
}
