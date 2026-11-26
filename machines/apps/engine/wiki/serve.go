package wiki

import (
	"cmp"
	"errors"
	"path"
	"slices"
	"strconv"
	"strings"
)

type Served struct {
	// value: a page of a corpus as a reader is served it, at its four levels
	// of detail at once: each block, group and prose with the face it takes
	// at each level and the most it opens, its blocks' reading (tier, the
	// findings that sit on it), its groups and the tables that mirror them,
	// its tables' rows found, and how long each level takes to read. Every judgement in it is
	// the engine's; the window draws it.
	Page      PageHead
	Paragraph []Paragraph
	Lede      ServedLede
	Floor     FloorHead
	Sections  []ServedSection
	Findings  []ServedFinding
	// value: what the page's head counts of it: the findings that need the
	// owner, on the page or its blocks, and its blocks, a group's included.
	NeedsYou, Blocks int
	// value: every finding served on the page and its blocks, in the order
	// a reader is asked to look at them, what needs the owner first.
	Attention []ServedFinding
	// value: the reading time of each level, word first.
	Reading []ServedReading
	Icons   map[string][]string
	Verdict Verdict
}

type ServedLede struct {
	// value: what stands under the page's title at each level: its
	// paragraph's first sentence at word, the paragraph clipped to its line
	// at line and whole from card; on a decision record its decision, a line
	// up to line and its full face from card.
	Ranks []Detail
	First []Run
	More  bool
}

type PageHead struct {
	ID    string
	Seat  string
	Floor int
	Word  string
	Line  string
	Icon  string
	// value: the hard and the soft findings that sit on the page or its blocks.
	Hard, Soft int
	// value: on a decision record, its number and its title read from its
	// head, and the decision as its record and the block that names it say
	// it; empty on any other page.
	Number   string
	Title    string
	Decision *ServedDecision
}

type FloorHead struct {
	Index    int
	Name     string
	Question string
}

type Verdict struct{ Hard, Soft int }

type ServedSection struct {
	Heading string
	Anchor  string
	// value: the page's own "Rests on" section, the one S6 reads: what the
	// page stands on, served where it is written.
	Rests bool
	// value: how a reader names the section, its heading or the opening's
	// word, and its icon: home for the opening, rests for Rests on, guard
	// when a group of it guards, else define.
	Label string
	Icon  string
	Items []ServedItem
}

type ServedItem struct {
	// invariant: one of Prose, Group or Block is set. Ranks holds the face
	// it takes at each level, word first, never less at a level than
	// at the one before it nor more than Max, the most it opens.
	Ranks []Detail
	Max   Detail
	Prose *ServedProse
	Group *ServedGroup
	Block *ServedBlock
}

type ServedProse struct {
	// value: prose as paragraphs, list items, quotes and table lines, its
	// id p<section>.<k>, the k-th prose of its section, and its first
	// sentence, which its word and line faces show.
	ID         string
	Paragraphs []Paragraph
	First      []Run
	More       bool
}

type ServedGroup struct {
	ID      string
	Address string
	// value: a block it holds raises what needs the owner.
	NeedsYou bool
	Title    string
	Intent   Intent
	// value: why it opens folded: its author wrote fold, or the seat of the
	// page reads no group of its intent; empty when it opens.
	Folded Fold
	Blocks int
	// value: the table of the page whose rows are its blocks, and the first
	// level at which that table shows them as rows, from which the group
	// shows its blocks' words alone.
	MirroredBy   string
	MirroredFrom Detail
	// value: the floor its blocks sit on, served only when it is not its
	// page's.
	OwnFloor *FloorHead
	// value: the line Face of its lead, its one line when folded.
	Lead   string
	ReadBy []string
	Items  []ServedItem
}

type Fold string

const (
	FoldNone   Fold = ""
	FoldAuthor Fold = "author"
	FoldSeat   Fold = "seat"
)

type Shown string

const (
	// value: how a budget of S11 shows a block: a wide tile, a card, one
	// line, counted beside its set, or not at all; the levels are read off it.
	ShownTile    Shown = "tile"
	ShownCard    Shown = "card"
	ShownLine    Shown = "line"
	ShownCounted Shown = "counted"
	ShownNone    Shown = "none"
)

type ServedBlock struct {
	ID      string
	Address string
	Kind    string
	Title   string
	Icon    string
	Word    string
	Line    string
	Floor   int
	// value: the floor the block writes, served only when it is not its
	// page's, so a block set deeper than its page says where it sits.
	OwnFloor *FloorHead
	Tier     Tier
	Score    int
	Why      []Cause
	Fields   []ServedField
	Table    *ServedTable
	Sub      *PageHead
	Figure   *ServedFigure
	Decision *ServedDecision
	Finding  []ServedFinding
	// value: one of its findings needs the owner.
	NeedsYou bool
}

type ServedField struct {
	Name string
	Text string
	Refs []ServedRef
}

type ServedRef struct {
	Address string
	Word    string
	// value: the address names a page or a block of the corpus.
	Resolves bool
}

type ServedTable struct {
	Of      string
	Title   string
	In      ServedRef
	Where   string
	Columns []string
	Rows    []ServedRow
	// value: how many rows it found, said with the noun row; and the group
	// of the page whose blocks are its rows, drawn above it.
	Count   int
	Mirrors string
}

type ServedRow struct {
	Address string
	Word    string
	Icon    string
	Cells   []ServedField
}

type ServedFigure struct {
	Form    string
	Caption string
	Shows   ServedRef
	Parts   []FigurePart
	X, Y    string
}

type ServedFinding struct {
	Rule  string
	Hard  bool
	Level Level
	Line  int
	Says  string
	// value: where the reader finds what it is about: the address of its
	// block, or of the group served folded that holds it, empty for the
	// page's own head; and the word of what it is about.
	At   string
	Word string
	// value: the first level that says it: word for one that needs the
	// owner or a hard one, line for a remark, card for an ambient one.
	From Detail
}

var (
	ErrNoSchema    = errors.New("the corpus has no wiki.yaml at its root")
	ErrUnknownPage = errors.New("no page of the corpus has this id")
)

const schemaFile = "wiki.yaml"

type Corpus struct {
	// value: a corpus read from its files by slash path under its root: its
	// schema, every page compiled in id order, and the check of them all.
	Schema Schema
	Pages  []Page
	Report Report
	// value: the decision records among the pages, by page id, each read
	// from its file's head.
	Records map[string]Record
}

func Read(files map[string][]byte) (Corpus, error) {
	body, ok := files[schemaFile]
	if !ok {
		return Corpus{}, ErrNoSchema
	}
	schema, err := ReadSchema(schemaFile, body)
	if err != nil {
		return Corpus{}, err
	}
	var ids []string
	for p := range files {
		if strings.HasSuffix(p, ".md") && path.Base(p) != "README.md" {
			ids = append(ids, strings.TrimSuffix(p, ".md"))
		}
	}
	slices.Sort(ids)
	pages := make([]Page, 0, len(ids))
	records := map[string]Record{}
	for _, id := range ids {
		body := string(files[id+".md"])
		pages = append(pages, Compile(id, body, schema))
		if r, ok := readRecord(id, body); ok {
			records[id] = r
		}
	}
	return Corpus{Schema: schema, Pages: pages, Report: Check(schema, pages), Records: records}, nil
}

func (c Corpus) Home() string {
	// value: the page a reader opens when none is named: the home of the
	// first seat.
	if len(c.Schema.Seats) == 0 {
		return ""
	}
	return c.Schema.Seats[0].Home
}

func (c Corpus) Serve(id string) (Served, error) {
	i := slices.IndexFunc(c.Pages, func(p Page) bool { return p.ID == id })
	if i < 0 {
		return Served{}, ErrUnknownPage
	}
	s := server{c: c, readings: map[string]BlockReading{}, pages: map[string]*Page{}, blocks: map[string]*Block{}, groups: map[string]*Group{}}
	s.index()
	p := &c.Pages[i]
	reading := s.reading(p.ID)
	out := Served{
		Page: s.head(p), Paragraph: Paragraphs(p.Faces.Paragraph),
		Floor: s.floor(reading.Floor), Icons: map[string][]string{},
	}
	out.Lede = lede(out.Page, out.Paragraph)
	for _, icon := range c.Schema.Icons {
		out.Icons[icon.Name] = icon.Rows
	}
	for _, f := range c.Report.Findings {
		if f.Hard {
			out.Verdict.Hard++
		} else {
			out.Verdict.Soft++
		}
		if f.Page == p.ID && f.Block == "" {
			out.Findings = append(out.Findings, served(f, "", p.Faces.Word))
		}
	}
	for si, sec := range p.Sections {
		ss := ServedSection{Label: openingLabel, Icon: "home"}
		if sec.Heading != nil {
			ss.Heading, ss.Anchor, ss.Rests = sec.Heading.Text, sec.Heading.Anchor, restsOn(sec.Heading.Text)
			ss.Label, ss.Icon = ss.Heading, sectionIcon(ss.Rests, sec.Items)
		}
		if si == 0 && sec.Heading == nil {
			sec.Items = s.afterParagraph(sec.Items, p.Faces.Paragraph)
		}
		k := 0
		for _, it := range sec.Items {
			item := s.item(p, reading, it, false)
			if item.Prose != nil {
				item.Prose.ID = "p" + strconv.Itoa(si) + "." + strconv.Itoa(k)
				k++
			}
			ss.Items = append(ss.Items, item)
		}
		out.Sections = append(out.Sections, ss)
	}
	mirror(out.Sections)
	out.Blocks = blocksOf(out.Sections)
	out.Findings = ranked(out.Findings)
	out.Attention = attention(out.Findings, out.Sections)
	out.NeedsYou = needsYou(out.Attention)
	out.Reading = readingOf(out)
	return out, nil
}

// value: the word a section with no heading, the page's opening, is named by.
const openingLabel = "Top"

func sectionIcon(rests bool, items []Item) string {
	if rests {
		return "rests"
	}
	for _, it := range items {
		if g, ok := it.(*Group); ok && g.Intent == IntentGuard {
			return "guard"
		}
	}
	return "define"
}

func blocksOf(sections []ServedSection) int {
	n := 0
	var walk func(items []ServedItem)
	walk = func(items []ServedItem) {
		for _, it := range items {
			if it.Block != nil {
				n++
			}
			if it.Group != nil {
				walk(it.Group.Items)
			}
		}
	}
	for _, sec := range sections {
		walk(sec.Items)
	}
	return n
}

type server struct {
	c        Corpus
	readings map[string]BlockReading
	pages    map[string]*Page
	blocks   map[string]*Block
	groups   map[string]*Group
}

func (s *server) index() {
	for _, r := range s.c.Report.Readings {
		for _, b := range r.Blocks {
			s.readings[b.Address] = b
		}
	}
	for i := range s.c.Pages {
		p := &s.c.Pages[i]
		s.pages[p.ID] = p
		for _, sec := range p.Sections {
			for _, it := range sec.Items {
				s.indexItem(p.ID, it)
			}
		}
	}
}

func (s *server) indexItem(page string, it Item) {
	switch it := it.(type) {
	case *Block:
		s.blocks[it.Address] = it
	case *Group:
		s.groups[page+"#"+it.ID] = it
		for _, inner := range it.Items {
			s.indexItem(page, inner)
		}
	}
}

func (s *server) reading(id string) Reading {
	for _, r := range s.c.Report.Readings {
		if r.Page == id {
			return r
		}
	}
	return Reading{Page: id}
}

func (s *server) floor(i int) FloorHead {
	if i >= 0 && i < len(s.c.Schema.Floors) {
		f := s.c.Schema.Floors[i]
		return FloorHead{Index: i, Name: f.Name, Question: f.Question}
	}
	return FloorHead{Index: i}
}

func (s *server) head(p *Page) PageHead {
	r := s.reading(p.ID)
	h := PageHead{ID: p.ID, Seat: r.Seat, Floor: r.Floor, Word: p.Faces.Word, Line: plainOf(p.Faces.Line), Icon: p.Faces.Icon}
	if record, ok := s.c.Records[p.ID]; ok {
		h.Number, h.Title, h.Decision = record.Number, record.Title, s.record(p.ID)
	}
	for _, f := range s.c.Report.Findings {
		if f.Page != p.ID {
			continue
		}
		if f.Hard {
			h.Hard++
		} else {
			h.Soft++
		}
	}
	return h
}

// invariant: the page's first paragraph is its paragraph Face, served once
// under its title, so the prose that holds it does not say it again.
func (s *server) afterParagraph(items []Item, paragraph string) []Item {
	if len(items) == 0 || paragraph == "" {
		return items
	}
	prose, ok := items[0].(*Prose)
	if !ok {
		return items
	}
	rest := strings.TrimSpace(prose.Markdown)
	first, after, _ := strings.Cut(rest, "\n\n")
	if strings.Join(strings.Fields(first), " ") != strings.Join(strings.Fields(paragraph), " ") {
		return items
	}
	if strings.TrimSpace(after) == "" {
		return items[1:]
	}
	trimmed := *prose
	trimmed.Markdown = after
	return append([]Item{&trimmed}, items[1:]...)
}

func (s *server) item(p *Page, r Reading, it Item, grouped bool) ServedItem {
	switch it := it.(type) {
	case *Prose:
		paras := Paragraphs(it.Markdown)
		first, more := firstSentenceOf(paras)
		return ServedItem{Ranks: openRanks(), Max: DetailCard, Prose: &ServedProse{Paragraphs: paras, First: first, More: more}}
	case *Group:
		return ServedItem{Ranks: openRanks(), Max: DetailCard, Group: s.group(p, r, it)}
	case *Block:
		b := s.block(p, it)
		most := DetailFull
		if b.Figure != nil {
			most = DetailCard
		}
		return ServedItem{Ranks: blockRanks(b.Tier, grouped, most), Max: most, Block: b}
	}
	return ServedItem{}
}

// invariant: SPEC-v1 §15, read at the levels: at word (30 s) only needs-you
// Signals show, at line (2 min) remarks too, at card (10 min) ambient as
// well; a hard finding shows at every level.
func fromOf(f Finding) Detail {
	switch {
	case f.Hard || f.Level == LevelNeedsYou:
		return DetailWord
	case f.Level == LevelRemark:
		return DetailLine
	}
	return DetailCard
}

func restsOn(heading string) bool { return strings.EqualFold(heading, "rests on") }

func (s *server) group(p *Page, r Reading, g *Group) *ServedGroup {
	gr := s.readings[p.ID+"#"+g.ID]
	out := &ServedGroup{ID: g.ID, Address: p.ID + "#" + g.ID, Title: g.Title, Intent: g.Intent, ReadBy: gr.ReadBy}
	switch {
	case g.Fold:
		out.Folded = FoldAuthor
	case !slices.Contains(gr.ReadBy, r.Seat):
		out.Folded = FoldSeat
	}
	for _, inner := range g.Items {
		si := s.item(p, r, inner, true)
		if si.Block != nil && si.Block.Tier != "" {
			out.Blocks++
		}
		if si.Block != nil && si.Block.OwnFloor != nil {
			out.OwnFloor = si.Block.OwnFloor
		}
		if si.Block != nil && si.Block.NeedsYou {
			out.NeedsYou = true
		}
		out.Items = append(out.Items, si)
	}
	if b, ok := s.blocks[gr.Lead]; ok {
		out.Lead = b.Faces.Line
	}
	return out
}

func (s *server) block(p *Page, b *Block) *ServedBlock {
	r := s.readings[b.Address]
	kind, _ := s.c.Schema.kind(b.Kind)
	out := &ServedBlock{
		ID: b.ID, Address: b.Address, Kind: b.Kind, Title: titleOf(kind, b.Kind), Icon: cmp.Or(b.Faces.Icon, kind.Icon, b.Kind),
		Word: b.Faces.Word, Line: plainOf(b.Faces.Line), Floor: r.Floor,
	}
	if r.Weight != nil {
		out.Tier, out.Score, out.Why = r.Weight.Tier, r.Weight.Score, r.Weight.Why
	}
	if page := s.reading(p.ID).Floor; r.Floor != page {
		f := s.floor(r.Floor)
		out.OwnFloor = &f
	}
	for _, f := range s.c.Report.Findings {
		if f.Page == p.ID && f.Block == b.ID {
			out.Finding = append(out.Finding, served(f, b.Address, out.Word))
		}
	}
	out.Finding = ranked(out.Finding)
	out.NeedsYou = needsYou(out.Finding) > 0
	if b.Kind == "decision" {
		out.Decision = s.decision(b)
	}
	switch v := b.View.(type) {
	case *TableView:
		out.Table = s.table(v, r.Rows)
	case *SubPageView:
		if child, ok := s.pages[v.Child]; ok {
			h := s.head(child)
			out.Sub = &h
		} else {
			out.Sub = &PageHead{ID: v.Child}
		}
	case *FigureView:
		out.Figure = &ServedFigure{Form: v.Form, Caption: v.Caption, Shows: s.ref(v.Shows), Parts: s.parts(v), X: textOf(b.Props, "x"), Y: textOf(b.Props, "y")}
	default:
		out.Fields = s.fields(kind, b.Props)
	}
	return out
}

func titleOf(k Kind, id string) string {
	return cmp.Or(k.Title["en"], tagOf(id))
}

// invariant: SPEC-v1 §15's table: the budget shows a lead, a normal and a
// quiet block each at its size; a block that has no weight (a table, a
// figure, a sub-page) shows whole but at 30 s. At 30 s a group's one line
// carries its lead, and a lead no group holds is its own tile.
func shown(t Tier, b Budget, grouped bool) Shown {
	if b == Budget30s && t == TierLead && !grouped {
		return ShownTile
	}
	if b == Budget30s {
		return ShownNone
	}
	wide := b == Budget10min
	switch t {
	case TierLead:
		return pick(wide, ShownTile, ShownCard)
	case TierNormal:
		return pick(wide, ShownCard, ShownLine)
	case TierQuiet:
		return pick(wide, ShownLine, ShownCounted)
	default:
		return ShownTile
	}
}

func pick(wide bool, a, b Shown) Shown {
	if wide {
		return a
	}
	return b
}

func served(f Finding, at, word string) ServedFinding {
	return ServedFinding{Rule: f.Rule, Hard: f.Hard, Level: f.Level, Line: f.Line, Says: f.Says, At: at, Word: word, From: fromOf(f)}
}

func (s *server) ref(address string) ServedRef {
	if b, ok := s.blocks[address]; ok {
		return ServedRef{Address: address, Word: cmp.Or(b.Faces.Word, b.ID), Resolves: true}
	}
	if p, ok := s.pages[address]; ok {
		return ServedRef{Address: address, Word: p.Faces.Word, Resolves: true}
	}
	if g, ok := s.groups[address]; ok {
		return ServedRef{Address: address, Word: cmp.Or(g.Title, g.ID), Resolves: true}
	}
	return ServedRef{Address: address, Word: address}
}

func (s *server) fields(k Kind, props Props) []ServedField {
	var out []ServedField
	for _, f := range k.Fields {
		v, ok := props.Get(f.Name)
		if !ok || v.Kind == ValueNull {
			continue
		}
		out = append(out, s.field(f.Name, v))
	}
	return out
}

func (s *server) field(name string, v Value) ServedField {
	out := ServedField{Name: name}
	switch v.Kind {
	case ValueRef:
		out.Refs = []ServedRef{s.ref(v.Text)}
	case ValueList:
		var texts []string
		for _, it := range v.List {
			if it.Kind == ValueRef {
				out.Refs = append(out.Refs, s.ref(it.Text))
			} else {
				texts = append(texts, text(it))
			}
		}
		out.Text = strings.Join(texts, ", ")
	default:
		out.Text = text(v)
	}
	return out
}

func (s *server) table(v *TableView, rows []string) *ServedTable {
	kind, _ := s.c.Schema.kind(v.Of)
	out := &ServedTable{Of: v.Of, Title: titleOf(kind, v.Of), In: s.ref(v.In), Columns: v.Columns}
	if v.Where != nil {
		out.Where = s.where(v.Where)
	}
	for _, address := range rows {
		b, ok := s.blocks[address]
		if !ok {
			continue
		}
		row := ServedRow{Address: address, Word: cmp.Or(b.Faces.Word, b.ID), Icon: cmp.Or(b.Faces.Icon, kind.Icon, b.Kind)}
		r := s.readings[address]
		for _, col := range v.Columns {
			row.Cells = append(row.Cells, s.cell(col, b, r))
		}
		out.Rows = append(out.Rows, row)
	}
	out.Count = len(out.Rows)
	return out
}

// invariant: a table's filter is served as the words it reads, its field
// then what it names: `by is ref("product/stack#machines")` reads "by
// Machines", `carried_in has ref("product/stack#wiki")` "carried in Wiki".
func (s *server) where(q *Query) string {
	operand := text(q.Operand)
	if q.Operand.Kind == ValueRef {
		operand = s.ref(q.Operand.Text).Word
	}
	return strings.ReplaceAll(q.Field, "_", " ") + " " + operand
}

// invariant: a figure states nothing its shows does not (SPEC-v1 §11), so a
// figure with no parts written is drawn from what it shows: a group's
// blocks, or the one block it names, each by its word.
func (s *server) parts(v *FigureView) []FigurePart {
	if len(v.Parts) > 0 {
		return v.Parts
	}
	if counted := s.counted(v); counted != nil {
		return counted
	}
	if b, ok := s.blocks[v.Shows]; ok {
		return []FigurePart{{Name: cmp.Or(b.Faces.Word, b.ID)}}
	}
	g, ok := s.groups[v.Shows]
	if !ok {
		return nil
	}
	var out []FigurePart
	for _, inner := range g.Items {
		if b, ok := inner.(*Block); ok {
			out = append(out, FigurePart{Name: cmp.Or(b.Faces.Word, b.ID)})
		}
	}
	return out
}

// invariant: a stack with no parts written that shows a table counts what
// the table says: for each page or block its rows' refs name, how many rows
// name it, the most named first, ties in the order they are first named.
func (s *server) counted(v *FigureView) []FigurePart {
	b, ok := s.blocks[v.Shows]
	if !ok || v.Form != "stack" {
		return nil
	}
	t, ok := b.View.(*TableView)
	if !ok {
		return nil
	}
	var out []FigurePart
	at := map[string]int{}
	for _, row := range s.readings[v.Shows].Rows {
		rb, ok := s.blocks[row]
		if !ok {
			continue
		}
		for _, col := range t.Columns {
			value, ok := rb.Props.Get(col)
			if !ok {
				continue
			}
			for _, ref := range s.field(col, value).Refs {
				i, seen := at[ref.Address]
				if !seen {
					i = len(out)
					at[ref.Address] = i
					out = append(out, FigurePart{Name: ref.Word})
				}
				out[i].N++
			}
		}
	}
	slices.SortStableFunc(out, func(a, b FigurePart) int { return cmp.Compare(b.N, a.N) })
	return out
}

func (s *server) cell(col string, b *Block, r BlockReading) ServedField {
	switch col {
	case "@weight":
		if r.Weight == nil {
			return ServedField{Name: col}
		}
		return ServedField{Name: col, Text: string(r.Weight.Tier) + " " + strconv.Itoa(r.Weight.Score)}
	case "@state":
		kind, _ := s.c.Schema.kind(b.Kind)
		return ServedField{Name: col, Text: textOf(b.Props, kind.State)}
	case "@proof":
		return ServedField{Name: col, Text: textOf(b.Props, "proof")}
	}
	v, ok := b.Props.Get(col)
	if !ok {
		return ServedField{Name: col}
	}
	return s.field(col, v)
}
