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
	// value: a page of a corpus as a reader is served it: its blocks with
	// their reading (tier, how the budget shows each, the findings that sit
	// on it), its groups folded or open, its tables' rows found, and the bar
	// and the tree it is read among. Every judgement in it is the engine's;
	// the window draws it.
	Page      PageHead
	Paragraph []Paragraph
	Floor     FloorHead
	Budget    Budget
	Budgets   []Budget
	Seats     []SeatHead
	Pages     []PageHead
	Sections  []ServedSection
	Findings  []ServedFinding
	Icons     map[string][]string
	Verdict   Verdict
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
}

type FloorHead struct {
	Index    int
	Name     string
	Question string
}

type SeatHead struct {
	ID   string
	Name string
	Icon string
	Home string
	// value: the seat a reader holds the blueprint with, read as a seat
	// without owning one.
	Plan bool
	// value: the seat of the page served.
	Active bool
}

type Verdict struct{ Hard, Soft int }

type ServedSection struct {
	Heading string
	Anchor  string
	// value: the page's own "Rests on" section, the one S6 reads: what the
	// page stands on, served where it is written.
	Rests bool
	Items []ServedItem
}

type ServedItem struct {
	// invariant: one of Prose, Group or Block is set.
	Prose []Paragraph
	Group *ServedGroup
	Block *ServedBlock
}

type ServedGroup struct {
	ID     string
	Title  string
	Intent Intent
	// value: why it opens folded: its author wrote fold, the seat of the
	// page reads no group of its intent, or the budget folds every group;
	// empty when it opens.
	Folded Fold
	Blocks int
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
	FoldBudget Fold = "budget"
)

type Shown string

const (
	// value: how the budget shows a block: a wide tile, a card, one line,
	// counted beside its set, or not at all.
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
	Tier    Tier
	Score   int
	Why     []Cause
	Shown   Shown
	Fields  []ServedField
	Table   *ServedTable
	Sub     *PageHead
	Figure  *ServedFigure
	Finding []ServedFinding
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
	for _, id := range ids {
		pages = append(pages, Compile(id, string(files[id+".md"]), schema))
	}
	return Corpus{Schema: schema, Pages: pages, Report: Check(schema, pages)}, nil
}

func (c Corpus) Home() string {
	// value: the page a reader opens when none is named: the home of the
	// first seat.
	if len(c.Schema.Seats) == 0 {
		return ""
	}
	return c.Schema.Seats[0].Home
}

func Budgets() []Budget { return []Budget{Budget30s, Budget2min, Budget10min} }

func (c Corpus) Serve(id string, budget Budget) (Served, error) {
	i := slices.IndexFunc(c.Pages, func(p Page) bool { return p.ID == id })
	if i < 0 {
		return Served{}, ErrUnknownPage
	}
	if budget == "" {
		budget = c.Schema.Budget
	}
	s := server{c: c, budget: budget, readings: map[string]BlockReading{}, pages: map[string]*Page{}, blocks: map[string]*Block{}, groups: map[string]*Group{}}
	s.index()
	p := &c.Pages[i]
	reading := s.reading(p.ID)
	out := Served{
		Page: s.head(p), Paragraph: Paragraphs(p.Faces.Paragraph), Budget: budget, Budgets: Budgets(),
		Floor: s.floor(reading.Floor), Icons: map[string][]string{},
	}
	for _, seat := range c.Schema.Seats {
		out.Seats = append(out.Seats, SeatHead{ID: seat.ID, Name: seat.Name, Icon: seat.ID, Home: seat.Home, Plan: seat.Plan, Active: seat.ID == reading.Seat})
	}
	for j := range c.Pages {
		out.Pages = append(out.Pages, s.head(&c.Pages[j]))
	}
	for _, icon := range c.Schema.Icons {
		out.Icons[icon.Name] = icon.Rows
	}
	for _, f := range c.Report.Findings {
		if f.Hard {
			out.Verdict.Hard++
		} else {
			out.Verdict.Soft++
		}
		if f.Page == p.ID && f.Block == "" && s.signals(f) {
			out.Findings = append(out.Findings, served(f))
		}
	}
	for si, sec := range p.Sections {
		ss := ServedSection{}
		if sec.Heading != nil {
			ss.Heading, ss.Anchor, ss.Rests = sec.Heading.Text, sec.Heading.Anchor, restsOn(sec.Heading.Text)
		}
		if si == 0 && sec.Heading == nil {
			sec.Items = s.afterParagraph(sec.Items, p.Faces.Paragraph)
		}
		for _, it := range sec.Items {
			if _, prose := it.(*Prose); prose && budget == Budget30s {
				continue
			}
			ss.Items = append(ss.Items, s.item(p, reading, it, false))
		}
		out.Sections = append(out.Sections, ss)
	}
	return out, nil
}

type server struct {
	c        Corpus
	budget   Budget
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
		return ServedItem{Prose: Paragraphs(it.Markdown)}
	case *Group:
		return ServedItem{Group: s.group(p, r, it)}
	case *Block:
		return ServedItem{Block: s.block(p, it, grouped)}
	}
	return ServedItem{}
}

// invariant: SPEC-v1 §15: at 30 s only needs-you Signals show, at 2 min
// remarks too, at 10 min ambient as well; a hard finding shows at every budget.
func (s *server) signals(f Finding) bool {
	switch {
	case f.Hard || f.Level == LevelNeedsYou:
		return true
	case f.Level == LevelRemark:
		return s.budget != Budget30s
	}
	return s.budget == Budget10min
}

func restsOn(heading string) bool { return strings.EqualFold(heading, "rests on") }

func (s *server) group(p *Page, r Reading, g *Group) *ServedGroup {
	gr := s.readings[p.ID+"#"+g.ID]
	out := &ServedGroup{ID: g.ID, Title: g.Title, Intent: g.Intent, ReadBy: gr.ReadBy}
	switch {
	case g.Fold:
		out.Folded = FoldAuthor
	case s.budget == Budget30s:
		out.Folded = FoldBudget
	case !slices.Contains(gr.ReadBy, r.Seat):
		out.Folded = FoldSeat
	}
	for _, inner := range g.Items {
		si := s.item(p, r, inner, true)
		if si.Block != nil && si.Block.Tier != "" {
			out.Blocks++
		}
		out.Items = append(out.Items, si)
	}
	if b, ok := s.blocks[gr.Lead]; ok {
		out.Lead = b.Faces.Line
	}
	return out
}

func (s *server) block(p *Page, b *Block, grouped bool) *ServedBlock {
	r := s.readings[b.Address]
	kind, _ := s.c.Schema.kind(b.Kind)
	out := &ServedBlock{
		ID: b.ID, Address: b.Address, Kind: b.Kind, Title: titleOf(kind, b.Kind), Icon: cmp.Or(b.Faces.Icon, kind.Icon, b.Kind),
		Word: b.Faces.Word, Line: plainOf(b.Faces.Line), Floor: r.Floor,
	}
	if r.Weight != nil {
		out.Tier, out.Score, out.Why = r.Weight.Tier, r.Weight.Score, r.Weight.Why
	}
	out.Shown = shown(out.Tier, s.budget, grouped)
	for _, f := range s.c.Report.Findings {
		if f.Page == p.ID && f.Block == b.ID && s.signals(f) {
			out.Finding = append(out.Finding, served(f))
		}
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

func served(f Finding) ServedFinding {
	return ServedFinding{Rule: f.Rule, Hard: f.Hard, Level: f.Level, Line: f.Line, Says: f.Says}
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
