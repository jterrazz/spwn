package wiki

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
)

type Map struct {
	// value: the graph a reader navigates a corpus by, every judgement in it
	// the engine's: each seat, each seat's pages and, under the page Open
	// names, that page's sections and their items. Roots are the seats in
	// the schema's order; Nodes hold every node served, each parent before
	// its kids. A page not open is served with no kids, its sections only
	// counted and marked.
	Home   MapHome
	Floors []MapFloor
	Roots  []string
	Open   string
	Nodes  []MapNode
	// value: the dot icons the nodes are drawn with, by name, each its rows.
	Icons map[string][]string
}

type MapHome struct {
	// value: the head the root is drawn with: its word and line, the pages
	// and the seats the map holds, the findings that need the owner and the
	// pages they sit on, and the corpus's hard and soft findings.
	Word       string
	Line       string
	Pages      int
	Seats      int
	Needs      int
	NeedsPages int
	Hard       int
	Soft       int
}

type MapFloor struct {
	// value: a floor of the ladder and how the band of the canvas it names
	// is labelled.
	Index    int
	Name     string
	Question string
	Label    string
}

type NodeKind string

const (
	NodeSeat    NodeKind = "seat"
	NodePage    NodeKind = "page"
	NodeSection NodeKind = "section"
	NodeBlock   NodeKind = "block"
	NodeGroup   NodeKind = "group"
	NodeProse   NodeKind = "prose"
)

func (NodeKind) Enum() []string {
	return []string{string(NodeSeat), string(NodePage), string(NodeSection), string(NodeBlock), string(NodeGroup), string(NodeProse)}
}

type MapNode struct {
	// value: a node of the graph: S:<seat>, P:<page>, C:<page>:<section's
	// index in wiki.read's Sections> or B:<page>:<item's id>, at its depth
	// from 1 for a seat to 4 for an item, with its parent, null for a seat,
	// and its kids in reading order. Ref is the id wiki.read serves it by: the
	// seat's, the page's, the section's anchor (empty for the opening) or the
	// block's, the group's or the prose's.
	ID     string
	Depth  int
	Kind   NodeKind
	Parent *string
	Kids   []string
	Ref    string
	// value: the findings under it that need the owner, its own and its
	// kids', and one mark per kid in order, true where that kid holds one; a
	// group marks its blocks. Marks holds MaxMarks at most, More counts the
	// rest.
	Needs int
	Marks []bool
	More  int `json:",omitempty"`
	// value: the band of the canvas its seat is drawn in: the lowest floor
	// of the seat's pages.
	Band int
	// value: what it is drawn with: its label on a pill, a head and its
	// accessible name; its title, kicker and line on a card, each served only
	// when it says something the label does not; its icon; and Count, what it
	// holds, said with its kind's noun: pages, sections, items or blocks.
	Label  string
	Title  string `json:",omitempty"`
	Kicker string `json:",omitempty"`
	Line   string `json:",omitempty"`
	Icon   string
	Count  int `json:",omitempty"`
	// value: on a seat, the node of the page a reader opens it at.
	Home string `json:",omitempty"`
}

// value: the most marks a node is served with, the most a face draws.
const MaxMarks = 11

// value: the words the map's root and its cards are drawn with.
const (
	mapWord        = "Wiki"
	mapLine        = "Every page of this Mac's wiki, by the seat that reads it."
	sectionKicker  = "Section"
	openingKicker  = "Opening"
	proseKicker    = "Paragraph"
	groupKicker    = "Group · "
	proseIcon      = "para"
	labelWords     = 3
	titleWords     = 4
	floorLetter    = "E"
	floorSeparator = " · "
)

func (c Corpus) Map(open string) (Map, error) {
	m := mapper{c: c, server: server{c: c, readings: map[string]BlockReading{}, pages: map[string]*Page{}, blocks: map[string]*Block{}, groups: map[string]*Group{}}, read: map[string]Reading{}, own: map[string]int{}, needs: map[string]int{}, served: map[string]bool{}}
	m.server.index()
	for _, r := range c.Report.Readings {
		m.read[r.Page] = r
	}
	for _, f := range c.Report.Findings {
		if f.Hard {
			m.out.Home.Hard++
		} else {
			m.out.Home.Soft++
		}
		if f.Level != LevelNeedsYou {
			continue
		}
		if f.Block == "" {
			m.own[f.Page]++
		} else {
			m.needs[f.Page+"#"+f.Block]++
		}
	}
	bySeat := map[string][]*Page{}
	for i := range c.Pages {
		p := &c.Pages[i]
		if seat := m.read[p.ID].Seat; seat != "" {
			bySeat[seat] = append(bySeat[seat], p)
		}
	}
	if open != "" && !slices.ContainsFunc(c.Pages, func(p Page) bool { return p.ID == open && m.read[p.ID].Seat != "" }) {
		return Map{}, ErrUnknownPage
	}
	m.out.Home.Word, m.out.Home.Line = mapWord, mapLine
	for i, f := range c.Schema.Floors {
		m.out.Floors = append(m.out.Floors, MapFloor{Index: i, Name: f.Name, Question: f.Question, Label: floorLetter + strconv.Itoa(i) + floorSeparator + f.Name})
	}
	for _, seat := range c.Schema.Seats {
		m.seat(seat, m.ordered(bySeat[seat.ID]), open)
	}
	m.out.Icons = map[string][]string{}
	for _, icon := range c.Schema.Icons {
		if slices.ContainsFunc(m.out.Nodes, func(n MapNode) bool { return n.Icon == icon.Name }) {
			m.out.Icons[icon.Name] = icon.Rows
		}
	}
	return m.out, nil
}

type mapper struct {
	c      Corpus
	server server
	read   map[string]Reading
	// value: the findings that need the owner by page, on its head, and by
	// block address.
	own    map[string]int
	needs  map[string]int
	served map[string]bool
	out    Map
}

// invariant: a seat's pages read lowest floor first, the decision records
// after the other pages of their floor, then by their word.
func (m *mapper) ordered(pages []*Page) []*Page {
	slices.SortStableFunc(pages, func(a, b *Page) int {
		_, ra := m.c.Records[a.ID]
		_, rb := m.c.Records[b.ID]
		return cmp.Or(
			cmp.Compare(m.read[a.ID].Floor, m.read[b.ID].Floor),
			cmp.Compare(boolRank(ra), boolRank(rb)),
			cmp.Compare(strings.ToLower(a.Faces.Word), strings.ToLower(b.Faces.Word)),
			cmp.Compare(a.Faces.Word, b.Faces.Word),
		)
	})
	return pages
}

func boolRank(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (m *mapper) add(n MapNode) int {
	m.out.Nodes = append(m.out.Nodes, n)
	return len(m.out.Nodes) - 1
}

func (m *mapper) seat(seat Seat, pages []*Page, open string) {
	id := "S:" + seat.ID
	band := 0
	if len(seat.Paths) > 0 {
		band = m.c.Schema.floorOf(seat.Paths[0])
	}
	if len(pages) > 0 {
		band = m.read[pages[0].ID].Floor
	}
	at := m.add(MapNode{ID: id, Depth: 1, Kind: NodeSeat, Ref: seat.ID, Band: band, Label: seat.Name, Icon: seat.ID, Count: len(pages)})
	m.out.Roots = append(m.out.Roots, id)
	m.out.Home.Seats++
	var kids []int
	for _, p := range pages {
		if seat.Home == p.ID {
			m.out.Nodes[at].Home = "P:" + p.ID
		}
		kids = append(kids, m.page(p, id, band, p.ID == open))
	}
	if m.out.Nodes[at].Home == "" && len(pages) > 0 {
		m.out.Nodes[at].Home = "P:" + pages[0].ID
	}
	m.close(at, kids, 0)
}

func (m *mapper) close(at int, kids []int, own int) {
	n := &m.out.Nodes[at]
	n.Needs = own
	marks := []bool{}
	n.Kids = []string{}
	for _, k := range kids {
		kid := m.out.Nodes[k]
		n.Kids = append(n.Kids, kid.ID)
		n.Needs += kid.Needs
		marks = append(marks, kid.Needs > 0)
	}
	n.Marks, n.More = capped(marks)
}

func capped(marks []bool) ([]bool, int) {
	if len(marks) <= MaxMarks {
		return marks, 0
	}
	return marks[:MaxMarks], len(marks) - MaxMarks
}

func (m *mapper) page(p *Page, parent string, band int, open bool) int {
	id := "P:" + p.ID
	floor := m.read[p.ID].Floor
	n := MapNode{ID: id, Depth: 2, Kind: NodePage, Parent: &parent, Ref: p.ID, Band: band, Label: p.Faces.Word, Line: plainOf(p.Faces.Line), Icon: m.read[p.ID].Seat}
	if f, ok := m.floorAt(floor); ok {
		n.Kicker = floorLetter + strconv.Itoa(floor) + " " + f.Name
	}
	if r, ok := m.c.Records[p.ID]; ok {
		n.Label, n.Title = r.Number, r.Number+floorSeparator+r.Title
		if d := m.server.record(p.ID); d.Decides != "" {
			n.Line = d.Decides
		}
	}
	at := m.add(n)
	if open {
		m.out.Open = id
	}
	m.out.Home.Pages++
	var sections []int
	marks := []bool{}
	needs := 0
	for si, sec := range p.Sections {
		items := sec.Items
		if si == 0 && sec.Heading == nil {
			items = m.server.afterParagraph(items, p.Faces.Paragraph)
		}
		if len(items) == 0 {
			continue
		}
		if open {
			sections = append(sections, m.section(p, si, sec.Heading, items, id, band))
			continue
		}
		count := m.needsOf(p.ID, items)
		needs += count
		marks = append(marks, count > 0)
	}
	if open {
		m.close(at, sections, m.own[p.ID])
		m.out.Nodes[at].Count = len(sections)
	} else {
		pn := &m.out.Nodes[at]
		pn.Kids, pn.Needs, pn.Count = []string{}, m.own[p.ID]+needs, len(marks)
		pn.Marks, pn.More = capped(marks)
	}
	if m.out.Nodes[at].Needs > 0 {
		m.out.Home.Needs += m.out.Nodes[at].Needs
		m.out.Home.NeedsPages++
	}
	return at
}

func (m *mapper) floorAt(i int) (Floor, bool) {
	if i < 0 || i >= len(m.c.Schema.Floors) {
		return Floor{}, false
	}
	return m.c.Schema.Floors[i], true
}

func (m *mapper) needsOf(page string, items []Item) int {
	n := 0
	for _, it := range items {
		switch it := it.(type) {
		case *Block:
			n += m.needs[page+"#"+it.ID]
		case *Group:
			n += m.needsOf(page, it.Items)
		}
	}
	return n
}

func (m *mapper) section(p *Page, si int, h *Heading, items []Item, parent string, band int) int {
	id := "C:" + p.ID + ":" + strconv.Itoa(si)
	n := MapNode{ID: id, Depth: 3, Kind: NodeSection, Parent: &parent, Band: band, Label: openingLabel, Kicker: openingKicker, Icon: "home"}
	if h != nil {
		rests := restsOn(h.Text)
		n.Ref, n.Label, n.Kicker, n.Icon = h.Anchor, h.Text, sectionKicker, sectionIcon(rests, items)
	}
	at := m.add(n)
	var kids []int
	var words []string
	first := ""
	k := 0
	for _, it := range items {
		var item MapNode
		switch it := it.(type) {
		case *Block:
			item = m.block(p, it)
			words = append(words, item.Label)
		case *Group:
			item = m.group(p, it)
			words = append(words, item.Label)
		case *Prose:
			item = m.prose(it, "p"+strconv.Itoa(si)+"."+strconv.Itoa(k))
			k++
			if first == "" {
				first = item.Line
			}
		default:
			continue
		}
		// invariant: a node is served once: an item whose id its page already
		// holds, which the check finds, is left out of the graph.
		if m.served["B:"+p.ID+":"+item.Ref] {
			continue
		}
		m.served["B:"+p.ID+":"+item.Ref] = true
		item.ID, item.Depth, item.Parent, item.Band, item.Kids = "B:"+p.ID+":"+item.Ref, 4, &id, band, []string{}
		kids = append(kids, m.add(item))
	}
	m.out.Nodes[at].Line, m.out.Nodes[at].Count = first, len(kids)
	if len(words) > 0 {
		m.out.Nodes[at].Line = strings.Join(words, floorSeparator)
	}
	m.close(at, kids, 0)
	return at
}

func (m *mapper) block(p *Page, b *Block) MapNode {
	kind, _ := m.c.Schema.kind(b.Kind)
	needs := m.needs[p.ID+"#"+b.ID]
	return MapNode{Kind: NodeBlock, Ref: b.ID, Needs: needs, Marks: []bool{}, Label: b.Faces.Word, Kicker: titleOf(kind, b.Kind), Line: plainOf(b.Faces.Line), Icon: cmp.Or(b.Faces.Icon, kind.Icon, b.Kind)}
}

func (m *mapper) group(p *Page, g *Group) MapNode {
	n := MapNode{Kind: NodeGroup, Ref: g.ID, Label: g.Title, Kicker: groupKicker + string(g.Intent), Icon: "define"}
	if g.Intent == IntentGuard {
		n.Icon = "guard"
	}
	marks := []bool{}
	var words []string
	for _, it := range g.Items {
		if b, ok := it.(*Block); ok {
			marks = append(marks, m.needs[p.ID+"#"+b.ID] > 0)
			words = append(words, b.Faces.Word)
		}
	}
	n.Needs, n.Count = m.needsOf(p.ID, g.Items), len(marks)
	n.Marks, n.More = capped(marks)
	n.Line = strings.Join(words, floorSeparator)
	if lead, ok := m.server.blocks[m.server.readings[p.ID+"#"+g.ID].Lead]; ok && lead.Faces.Line != "" {
		n.Line = lead.Faces.Line
	}
	return n
}

func (m *mapper) prose(pr *Prose, id string) MapNode {
	paras := Paragraphs(pr.Markdown)
	first, _ := firstSentenceOf(paras)
	var texts []string
	for _, p := range paras {
		if p.Cells == nil {
			texts = append(texts, runsText(p.Runs))
			continue
		}
		cells := make([]string, 0, len(p.Cells))
		for _, c := range p.Cells {
			cells = append(cells, runsText(c))
		}
		texts = append(texts, strings.Join(cells, " "))
	}
	words := strings.Fields(strings.Join(texts, " "))
	n := MapNode{Kind: NodeProse, Ref: id, Marks: []bool{}, Label: firstWords(words, labelWords), Kicker: proseKicker, Line: strings.TrimSpace(runsText(first)), Icon: proseIcon}
	if title := firstWords(words, titleWords); title != n.Label {
		n.Title = title
	}
	return n
}

func firstWords(words []string, n int) string {
	if len(words) <= n {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:n], " ") + "…"
}
