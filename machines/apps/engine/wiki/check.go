package wiki

import (
	"cmp"
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// value: the derived columns a table may show besides its kind's fields.
func derivedColumns() []string { return []string{"@state", "@vital", "@proof", "@opens", "@weight"} }

// value: the keys a page's frontmatter may hold, SPEC-v1 §2.
func frontKeys() []string {
	return []string{"kind", "floor", "needs", "line", "word", "icon", "rests_on", "retired"}
}

// value: the fields of a block that tie it to the code or to a record, the
// `anchored` cause of its weight.
func anchoringFields() []string {
	return []string{"code", "proof", "carried_by", "adr", "path", "source"}
}

const (
	// value: the caps of an authored Face and of a figure's caption, SPEC-v1 §6 and H28.
	lineCap = 120
	wordCap = 32
	// value: the most a tag spans, a group holds, a page holds without a
	// lead, and a table shows, SPEC-v1 §17.
	tagLines    = 12
	groupBlocks = 7
	pageBlocks  = 7
	pageLines   = 7
	tableCols   = 6
	// value: the most a grid of an icon spans each way.
	gridSide = 12
	// value: the readings of a page a reader has not ruled on.
	reading = "reading"
)

var (
	fragmentLink = regexp.MustCompile(`\]\(#([^)\s]+)\)`)
	tableRule    = regexp.MustCompile(`^\s*\|?[\s:|-]+\|?\s*$`)
)

type Report struct {
	// invariant: every finding of the corpus, the compile's and the check's,
	// sorted by page, line, column and rule.
	Findings []Finding `json:"findings"`
	Readings []Reading `json:"readings"`
}

type Reading struct {
	// invariant: what the check derives of a page, kept apart from it so a
	// compiled Page stays what its source says.
	Page   string         `json:"page"`
	Seat   string         `json:"seat"`
	Floor  int            `json:"floor"`
	Blocks []BlockReading `json:"blocks"`
}

type BlockReading struct {
	Address string  `json:"address"`
	Kind    string  `json:"kind"`
	Floor   int     `json:"floor"`
	Weight  *Weight `json:"weight"`
	// value: a table's rows, the addresses of the first MaxTableRows blocks
	// it finds, and how many more it finds.
	Rows []string `json:"rows,omitempty"`
	More int      `json:"more,omitempty"`
	// value: a group's lead and the seats that read its intent.
	Lead   string   `json:"lead,omitempty"`
	ReadBy []string `json:"readBy,omitempty"`
}

// value: a block where its page holds it.
type held struct {
	block   *Block
	group   *Group
	section int
}

type heldGroup struct {
	group   *Group
	section int
}

type target struct {
	page  *Page
	block *Block
	group *Group
}

type checker struct {
	schema    Schema
	pages     []*Page
	byID      map[string]*Page
	blocks    map[string][]held
	groups    map[string][]heldGroup
	targets   map[string]target
	addresses []string
	floors    map[string]int
	needsYou  map[string]bool
	rows      map[*Block]*rowSet
	kinds     map[scopeKey]*rowSet
	queries   map[queryKey]map[string]*rowSet
	inGroup   map[*Group][]held
	kids      map[string][]string
	// value: the findings of each page's compile, then the check's own, and
	// how many of the check's each page has of each rule.
	compiled []Finding
	findings []Finding
	said     map[ruleOn]int
}

func Check(ctx context.Context, schema Schema, pages []Page) (Report, error) {
	// invariant: a pure function of its arguments, like Compile: the corpus
	// is the pages given and its schema; no file, no git, no network and no
	// engine state is read, so a proof's freshness and a reader's last visit
	// are not judged here. Only an abandoned ask ends it early, between two
	// pages, with the context's error.
	k := newChecker(schema, pages)
	k.corpus()
	for _, p := range k.pages {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		k.page(p)
	}
	k.schemaRules()
	readings := make([]Reading, 0, len(k.pages))
	for _, p := range k.pages {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		readings = append(readings, k.weigh(p))
	}
	findings := append(append([]Finding{}, k.compiled...), capped(k.findings, k.said)...)
	sortFindings(findings)
	return Report{Findings: findings, Readings: readings}, nil
}

func sortFindings(findings []Finding) {
	slices.SortStableFunc(findings, func(a, b Finding) int {
		return cmp.Or(cmp.Compare(a.Page, b.Page), cmp.Compare(a.Line, b.Line), cmp.Compare(a.Col, b.Col), cmp.Compare(a.Rule, b.Rule))
	})
}

func newChecker(schema Schema, pages []Page) *checker {
	k := &checker{
		schema: schema, byID: map[string]*Page{}, blocks: map[string][]held{}, groups: map[string][]heldGroup{},
		targets: map[string]target{}, floors: map[string]int{}, needsYou: map[string]bool{}, rows: map[*Block]*rowSet{},
		kinds: map[scopeKey]*rowSet{}, queries: map[queryKey]map[string]*rowSet{}, inGroup: map[*Group][]held{},
		said: map[ruleOn]int{},
	}
	for i := range pages {
		p := &pages[i]
		k.compiled = append(k.compiled, p.Findings...)
		if _, twice := k.byID[p.ID]; twice {
			k.add("H3", p.ID, "", At{Line: 1, Col: 1}, fmt.Sprintf("page %q is read twice: a page id is its path, once in the corpus", p.ID))
			continue
		}
		k.pages = append(k.pages, p)
		k.byID[p.ID] = p
	}
	slices.SortFunc(k.pages, func(a, b *Page) int { return cmp.Compare(a.ID, b.ID) })
	for _, p := range k.pages {
		k.floors[p.ID] = k.pageFloor(p)
		k.targets[p.ID] = target{page: p}
		k.addresses = append(k.addresses, p.ID)
		for s, sec := range p.Sections {
			for _, it := range sec.Items {
				switch it := it.(type) {
				case *Block:
					k.hold(p, held{block: it, section: s})
				case *Group:
					k.groups[p.ID] = append(k.groups[p.ID], heldGroup{group: it, section: s})
					k.mark(p.ID+"#"+it.ID, it.ID, target{page: p, group: it})
					for _, inner := range it.Items {
						if b, ok := inner.(*Block); ok {
							k.hold(p, held{block: b, group: it, section: s})
						}
					}
				}
			}
		}
	}
	slices.Sort(k.addresses)
	k.kids = k.childrenOf()
	return k
}

func (k *checker) hold(p *Page, h held) {
	k.blocks[p.ID] = append(k.blocks[p.ID], h)
	if h.group != nil {
		k.inGroup[h.group] = append(k.inGroup[h.group], h)
	}
	if h.block.Kind != "page" {
		k.mark(h.block.Address, h.block.ID, target{page: p, block: h.block})
	}
}

func (k *checker) mark(address, id string, t target) {
	if id == "" {
		return
	}
	if _, twice := k.targets[address]; twice {
		return
	}
	k.targets[address] = t
	k.addresses = append(k.addresses, address)
}

func (k *checker) add(rule, page, block string, at At, says string) {
	on := ruleOn{page, rule}
	k.said[on]++
	if k.said[on] <= maxFindings+1 {
		k.findings = append(k.findings, finding(rule, page, block, at, says))
	}
}

// value: whether a page already has as many findings of a rule as are
// shown, so what the next would say need not be worked out.
func (k *checker) full(rule, page string) bool { return k.said[ruleOn{page, rule}] >= maxFindings }

func (k *checker) addSchema(rule string, line int, says string) {
	k.add(rule, k.schema.File, "", At{Line: line, Col: 1}, says)
}

// value: where a page-wide finding points: its title, else its first line.
func pageAt(p *Page) At {
	if p.Title != nil {
		return At{Line: p.Title.From, Col: 1}
	}
	return At{Line: 1, Col: 1}
}

func (k *checker) front(p *Page, key string) (FrontLine, bool) {
	if p.Front == nil {
		return FrontLine{}, false
	}
	for _, fl := range p.Front.Lines {
		if fl.Key == key {
			return fl, true
		}
	}
	return FrontLine{}, false
}

func (k *checker) pageFloor(p *Page) int {
	if fl, ok := k.front(p, "floor"); ok {
		if n, err := strconv.Atoi(fl.Value); err == nil && n >= 0 && n < len(k.schema.Floors) {
			return n
		}
	}
	return k.schema.floorOf(p.ID)
}

// value: a frontmatter list, `[a, b]` or one value, each item unquoted.
func frontList(value string) []string {
	value = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(value), "["), "]")
	var out []string
	for _, v := range strings.Split(value, ",") {
		if v = strings.Trim(strings.TrimSpace(v), `"'`); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (k *checker) corpus() {
	terms := map[string]string{}
	for _, p := range k.pages {
		for _, h := range k.blocks[p.ID] {
			b := h.block
			if b.Kind != "term" || b.ID == "" {
				continue
			}
			if first, twice := terms[b.ID]; twice {
				if first == p.ID {
					continue
				}
				k.add("H8", p.ID, b.ID, b.At, fmt.Sprintf("term %q is defined in %s and in %s: one owner", b.ID, first, p.ID))
				continue
			}
			terms[b.ID] = p.ID
			if _, retired := b.Props.Get("retired"); retired && blockIDForm.MatchString(b.ID) {
				k.retiredTerm(b)
			}
		}
	}
	k.cycles()
}

// invariant: a retired term is written nowhere: its words, the id's dashes
// read as spaces, are refused in the prose of every page.
func (k *checker) retiredTerm(term *Block) {
	words := strings.ReplaceAll(term.ID, "-", " ")
	pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(words) + `\b`)
	says := fmt.Sprintf("%q is retired: write what replaced it", words)
	if by, ok := term.Props.Get("replaced_by"); ok && by.Kind == ValueRef {
		_, id, _ := strings.Cut(by.Text, "#")
		says = fmt.Sprintf("%q is retired, write %q", words, strings.ReplaceAll(id, "-", " "))
	}
	for _, p := range k.pages {
		forProse(p, func(pr *Prose) {
			line, col, read := pr.Span.From, 1, 0
			for _, m := range pattern.FindAllStringIndex(pr.Markdown, -1) {
				for _, r := range pr.Markdown[read:m[0]] {
					col++
					if r == '\n' {
						line, col = line+1, 1
					}
				}
				read = m[0]
				k.add("H8", p.ID, "", At{Line: line, Col: col}, says)
			}
		})
	}
}

func forProse(p *Page, each func(*Prose)) {
	for _, sec := range p.Sections {
		for _, it := range sec.Items {
			switch it := it.(type) {
			case *Prose:
				each(it)
			case *Group:
				for _, inner := range it.Items {
					if pr, ok := inner.(*Prose); ok {
						each(pr)
					}
				}
			}
		}
	}
}

// invariant: `needs` has no cycle (H10). The pages that need each other
// are found in one pass, Tarjan's strongly connected components, so the
// check is linear in the pages and their needs; each such set is said once,
// from its first page in id order, by the shortest cycle back to it.
func (k *checker) cycles() {
	needs := map[string][]string{}
	for _, p := range k.pages {
		if fl, ok := k.front(p, "needs"); ok {
			for _, n := range frontList(fl.Value) {
				if _, page := k.byID[n]; page {
					needs[p.ID] = append(needs[p.ID], n)
				}
			}
		}
	}
	for _, set := range components(k.pages, needs) {
		first := slices.Min(set)
		if len(set) == 1 && !slices.Contains(needs[first], first) {
			continue
		}
		fl, _ := k.front(k.byID[first], "needs")
		k.add("H10", first, "", At{Line: fl.Line, Col: 1}, "needs: "+strings.Join(shortestCycle(first, set, needs), " -> "))
	}
}

// value: the strongly connected components of the needs graph, each its
// pages, found from the pages in id order.
func components(pages []*Page, needs map[string][]string) [][]string {
	index, low, on := map[string]int{}, map[string]int{}, map[string]bool{}
	var stack []string
	var out [][]string
	var visit func(v string)
	visit = func(v string) {
		index[v], low[v] = len(index), len(index)
		stack = append(stack, v)
		on[v] = true
		for _, w := range needs[v] {
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if on[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] != index[v] {
			return
		}
		var set []string
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			on[w] = false
			set = append(set, w)
			if w == v {
				break
			}
		}
		out = append(out, set)
	}
	for _, p := range pages {
		if _, seen := index[p.ID]; !seen {
			visit(p.ID)
		}
	}
	return out
}

// value: the fewest needs from first back to itself inside its component,
// each step taken in the order its page lists them.
func shortestCycle(first string, set []string, needs map[string][]string) []string {
	in := map[string]bool{}
	for _, v := range set {
		in[v] = true
	}
	from := map[string]string{first: ""}
	queue := []string{first}
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		for _, w := range needs[v] {
			if w == first {
				cycle := []string{first}
				for at := v; at != first; at = from[at] {
					cycle = append(cycle, at)
				}
				slices.Reverse(cycle[1:])
				return append(cycle, first)
			}
			if _, seen := from[w]; !seen && in[w] {
				from[w] = v
				queue = append(queue, w)
			}
		}
	}
	return []string{first, first}
}

func (k *checker) page(p *Page) {
	at := pageAt(p)
	if k.schema.seatOf(p.ID) == "" && p.ID != "README" {
		k.add("H32", p.ID, "", at, p.ID+".md is under no seat's path")
	}
	k.frontmatter(p)
	if len(k.blocks[p.ID]) == 0 && len(k.groups[p.ID]) == 0 {
		k.add("S8", p.ID, "", at, fmt.Sprintf("page %q is prose only", p.ID))
	}
	for _, sec := range p.Sections {
		if sec.Heading != nil && restsOn(sec.Heading.Text) {
			k.add("S6", p.ID, "", sec.Heading.At, fmt.Sprintf("%q rests on this page's reading: you have not ruled", p.ID))
			k.needsYou[p.ID] = true
		}
	}
	forProse(p, func(pr *Prose) {
		k.markdownTables(p, pr)
		k.proseLinks(p, pr)
	})
	ids := map[string]bool{}
	unique := func(id string, where At) {
		if id == "" {
			return
		}
		if ids[id] {
			k.add("H3", p.ID, id, where, fmt.Sprintf("id %q is defined twice in %s", id, p.ID))
		}
		ids[id] = true
	}
	for _, h := range k.groups[p.ID] {
		k.group(p, h)
	}
	placed := map[string]bool{}
	for _, h := range k.blocks[p.ID] {
		if h.block.Kind != "page" {
			unique(h.block.ID, h.block.At)
		}
		k.block(p, h, placed)
	}
	for _, h := range k.groups[p.ID] {
		unique(h.group.ID, h.group.At)
	}
	k.children(p, placed)
}

func (k *checker) frontmatter(p *Page) {
	if p.Front == nil {
		return
	}
	for _, fl := range p.Front.Lines {
		at := At{Line: fl.Line, Col: 1}
		switch fl.Key {
		case "floor":
			if n, err := strconv.Atoi(fl.Value); err != nil || n < 0 || n >= len(k.schema.Floors) {
				k.add("H2", p.ID, "", at, fmt.Sprintf("floor: %s is no floor: 0 to %d", fl.Value, len(k.schema.Floors)-1))
			}
		case "line":
			if n := len([]rune(fl.Value)); n > lineCap {
				k.add("H12", p.ID, "", at, fmt.Sprintf("line face of %s is %d characters, %d at most", p.ID, n, lineCap))
			}
		case "word":
			if n := len([]rune(fl.Value)); n > wordCap {
				k.add("H12", p.ID, "", at, fmt.Sprintf("word face of %s is %d characters, %d at most", p.ID, n, wordCap))
			}
		case "icon":
			if !k.schema.iconExists(fl.Value) {
				k.add("H29", p.ID, "", at, fmt.Sprintf("icon %q is in no icons: of wiki.yaml and is not built in", fl.Value))
			}
		case "needs":
			for _, n := range frontList(fl.Value) {
				k.resolve(p, "", Value{Kind: ValueRef, Text: n}, at)
			}
		case "rests_on":
			if fl.Value == reading {
				k.add("S6", p.ID, "", at, fmt.Sprintf("%q rests on this page's reading: you have not ruled", p.ID))
				k.needsYou[p.ID] = true
			} else {
				k.resolve(p, "", Value{Kind: ValueRef, Text: fl.Value}, at)
			}
		case "kind", "retired":
		default:
			k.add("H41", p.ID, "", at, fmt.Sprintf("%q is no key of a page's frontmatter: %s", fl.Key, orList(frontKeys())))
		}
	}
}

// invariant: a markdown table whose every column is a field of one kind
// asks for blocks and a <Table> (S28, a remark: SPEC-v1 C9).
func (k *checker) markdownTables(p *Page, pr *Prose) {
	lines := strings.Split(pr.Markdown, "\n")
	for i := 0; i+1 < len(lines); i++ {
		head := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(head, "|") || !tableRule.MatchString(lines[i+1]) || !strings.Contains(lines[i+1], "-") {
			continue
		}
		heads := strings.Split(strings.Trim(head, "|"), "|")
		cols := make([]string, 0, len(heads))
		for _, c := range heads {
			cols = append(cols, strings.ReplaceAll(strings.ToLower(strings.TrimSpace(c)), " ", "_"))
		}
		if len(cols) < 2 {
			continue
		}
		for _, id := range k.schema.contentKindIDs() {
			kind, _ := k.schema.kind(id)
			if !slices.ContainsFunc(cols, func(c string) bool { _, ok := kind.field(c); return !ok }) {
				k.add("S28", p.ID, "", At{Line: pr.Span.From + i, Col: 1}, fmt.Sprintf("this table's columns are the fields of %s: write blocks and a <Table>", id))
				break
			}
		}
	}
}

// value: the most of a refused link's target a finding quotes.
const quotedTarget = 40

// invariant: a prose link is judged as Paragraphs serves it, each refused
// target found after the one before it, so a page of links is read once;
// a target the joined lines no longer spell as written keeps the line of
// the last one found.
func (k *checker) proseLinks(p *Page, pr *Prose) {
	from, line, lost := 0, pr.Span.From, false
	paragraphs(pr.Markdown, &linking{lands: LandsOn(p.ID, k.holds), refused: func(target string) {
		if !lost {
			if i := strings.Index(pr.Markdown[from:], "]("+target); i >= 0 {
				line += strings.Count(pr.Markdown[from:from+i], "\n")
				from += i
			} else {
				lost = true
			}
		}
		quoted := target
		if len(quoted) > quotedTarget {
			quoted = strings.ToValidUTF8(quoted[:quotedTarget], "") + "..."
		}
		k.add("S30", p.ID, "", At{Line: line, Col: 1}, fmt.Sprintf("link to %q is no page, anchor or http or https address: served as its words", quoted))
	}})
}

func (k *checker) holds(id string) bool {
	_, ok := k.byID[id]
	return ok
}

func (k *checker) group(p *Page, h heldGroup) {
	g := h.group
	kind, _ := k.schema.kind("group")
	k.idForm(p, g.ID, "Group", g.At)
	k.fields(p, g.ID, kind, g.Props, g.At, "intent", "fold")
	k.common(p, g.ID, kind, g.Props, g.At)
	if v, ok := g.Props.Get("intent"); ok && !isNamed(intents(), text(v)) {
		k.add("H17", p.ID, g.ID, g.Props.at("intent", g.At), fmt.Sprintf("intent: %s does not exist: tell, define, decide, locate, guard or prove", text(v)))
	}
	if v, ok := g.Props.Get("fold"); ok && v.Kind != ValueBool {
		k.add("H35", p.ID, g.ID, g.Props.at("fold", g.At), fmt.Sprintf("fold: %s is true or false", text(v)))
	}
	n, floors := 0, []int{}
	for _, b := range k.inGroup[g] {
		n++
		if f := k.blockFloor(p, b.block); !slices.Contains(floors, f) {
			floors = append(floors, f)
		}
	}
	switch {
	case n == 0:
		k.add("S14", p.ID, g.ID, g.At, fmt.Sprintf("group %q is empty", g.ID))
	case n > groupBlocks:
		k.add("S15", p.ID, g.ID, g.At, fmt.Sprintf("group %q holds %d blocks: split it", g.ID, n))
	}
	if len(floors) > 1 {
		k.add("H40", p.ID, g.ID, g.At, fmt.Sprintf("%s holds blocks of floors %d and %d: a group lives on one floor", groupName(g), floors[0], floors[1]))
	}
	if g.Tag.To-g.Tag.From+1 > tagLines {
		k.add("S26", p.ID, g.ID, g.At, fmt.Sprintf("%s runs %d lines: put attributes two to a line", groupName(g), g.Tag.To-g.Tag.From+1))
	}
}

func (k *checker) idForm(p *Page, id, tag string, at At) {
	switch {
	case id == "":
		k.add("H3", p.ID, "", at, fmt.Sprintf("<%s at line %d> names no id: a block is reached by its id", tag, at.Line))
	case !blockIDForm.MatchString(id):
		k.add("H3", p.ID, id, at, fmt.Sprintf("id %q is no id: lower-case letters and digits, joined by dashes", id))
	}
}

func (k *checker) blockFloor(p *Page, b *Block) int {
	if v, ok := b.Props.Get("floor"); ok && v.Kind == ValueNumber {
		if n, err := strconv.Atoi(v.Text); err == nil && n >= 0 && n < len(k.schema.Floors) {
			return n
		}
	}
	return k.floors[p.ID]
}

// value: the fields of a kind its own rule judges, not H2.
func (k *checker) fields(p *Page, id string, kind Kind, props Props, at At, own ...string) {
	exempt := kind.ID == "scenario" && strings.HasSuffix(textOf(props, "proof"), ".spec.yaml")
	for _, f := range kind.Fields {
		v, ok := props.Get(f.Name)
		required := f.Required || kind.ID == "scenario" && !exempt && slices.Contains([]string{"given", "when", "then"}, f.Name)
		if !ok || v.Kind == ValueNull {
			if required {
				k.add("H2", p.ID, id, at, fmt.Sprintf("%s %q: required field %q is missing", kind.ID, id, f.Name))
			}
			continue
		}
		if slices.Contains(own, f.Name) {
			continue
		}
		if problem := fit(f.Type, f, v); problem != "" {
			k.add("H2", p.ID, id, props.at(f.Name, at), fmt.Sprintf("%s %q: %s", kind.ID, id, problem))
			continue
		}
		k.anchors(p, id, f, v, props.at(f.Name, at))
		if f.To != "" {
			k.narrowed(p, id, kind, f, v, props.at(f.Name, at))
		}
	}
	for _, prop := range props {
		if kind.ID == "page" && prop.Name == "id" {
			continue
		}
		for _, r := range refsIn(prop.Value) {
			k.resolve(p, id, r, whereOf(r, prop.At))
		}
	}
}

// invariant: an anchor names a repository of wiki.yaml's repos (H5, its
// pure half: whether the file and the symbol are there reads the clone).
func (k *checker) anchors(p *Page, id string, f Field, v Value, at At) {
	if !hasBase(f.Type, "anchor") {
		return
	}
	for _, s := range stringsIn(v) {
		m := anchorForm.FindStringSubmatch(s.Text)
		if m == nil {
			continue
		}
		if !slices.ContainsFunc(k.schema.Repos, func(r Repo) bool { return r.Name == m[1] }) {
			k.add("H5", p.ID, id, whereOf(s, at), fmt.Sprintf("anchor %s does not resolve: %q is no repository of wiki.yaml's repos", s.Text, m[1]))
		}
	}
}

// invariant: a ref narrowed by `to` points at a block of that kind (H2).
func (k *checker) narrowed(p *Page, id string, kind Kind, f Field, v Value, at At) {
	for _, r := range refsIn(v) {
		t, ok := k.targets[r.Text]
		if !ok {
			continue
		}
		is := "a page"
		switch {
		case t.block != nil && t.block.Kind == f.To:
			continue
		case t.block != nil:
			is = "a " + t.block.Kind
		case t.group != nil:
			is = "a group"
		}
		k.add("H2", p.ID, id, whereOf(r, at), fmt.Sprintf("%s %q: %s points to %s, %s; it takes a %s", kind.ID, id, f.Name, r.Text, is, f.To))
	}
}

// invariant: every ref resolves to a page of the corpus or a block or group
// of one (H4), and says the nearest address when one is two edits away.
func (k *checker) resolve(p *Page, id string, r Value, at At) {
	if _, ok := k.targets[r.Text]; ok {
		return
	}
	if k.full("H4", p.ID) {
		k.add("H4", p.ID, id, at, "")
		return
	}
	page, _, block := strings.Cut(r.Text, "#")
	says := fmt.Sprintf("%q is no page", r.Text)
	if _, ok := k.byID[page]; ok && block {
		says = fmt.Sprintf("%q is no block of %s", r.Text, page)
	}
	if near := nearest(r.Text, k.addresses); near != "" {
		says += fmt.Sprintf(" (did you mean %s?)", near)
	}
	k.add("H4", p.ID, id, at, says)
}

// value: the attributes every kind takes, judged by their own rules.
func (k *checker) common(p *Page, id string, kind Kind, props Props, at At) {
	if v, ok := props.Get("weight"); ok {
		switch {
		case v.Kind != ValueString || v.Text != string(TierLead) && v.Text != string(TierQuiet):
			k.add("H15", p.ID, id, props.at("weight", at), fmt.Sprintf("weight: %s does not exist: lead or quiet, or nothing and the engine weighs", text(v)))
		case kind.Structural:
			k.add("H15", p.ID, id, props.at("weight", at), fmt.Sprintf("a %s is not weighed: weight is written on a block the engine weighs", kind.ID))
		}
	}
	if v, ok := props.Get("icon"); ok && !k.schema.iconExists(text(v)) {
		k.add("H29", p.ID, id, props.at("icon", at), fmt.Sprintf("icon %q is in no icons: of wiki.yaml and is not built in", text(v)))
	}
	if v, ok := props.Get("floor"); ok {
		n, err := strconv.Atoi(v.Text)
		switch {
		case v.Kind != ValueNumber || err != nil || n < 0 || n >= len(k.schema.Floors):
			k.add("H2", p.ID, id, props.at("floor", at), fmt.Sprintf("floor: %s is no floor: 0 to %d", text(v), len(k.schema.Floors)-1))
		case n < k.floors[p.ID]:
			k.add("H9", p.ID, id, props.at("floor", at), fmt.Sprintf("%q is on floor %d, above its page's floor %d: a block is on its page's floor or deeper", id, n, k.floors[p.ID]))
		}
	}
	if v, ok := props.Get("rests_on"); ok {
		switch {
		case v.Kind == ValueString && v.Text == reading:
			k.add("S6", p.ID, id, props.at("rests_on", at), fmt.Sprintf("%q rests on this page's reading: you have not ruled", id))
			k.needsYou[p.ID+"#"+id] = true
		case v.Kind != ValueRef:
			k.add("H2", p.ID, id, props.at("rests_on", at), fmt.Sprintf("rests_on is %s: %q, or a ref to what it rests on", describe(v), reading))
		}
	}
}

func (k *checker) block(p *Page, h held, placed map[string]bool) {
	b := h.block
	kind, _ := k.schema.kind(b.Kind)
	if b.Kind != "page" {
		k.idForm(p, b.ID, tagOf(b.Kind), b.At)
	}
	var own []string
	switch b.Kind {
	case "table":
		own = []string{"of", "columns", "where"}
	case "figure":
		own = []string{"form", "caption"}
	}
	k.fields(p, b.ID, kind, b.Props, b.At, own...)
	k.common(p, b.ID, kind, b.Props, b.At)
	if lines := b.Span.To - b.Span.From + 1; lines > tagLines {
		k.add("S26", p.ID, b.ID, b.At, fmt.Sprintf("<%s %q> runs %d lines: put attributes two to a line", tagOf(b.Kind), b.ID, lines))
	}
	switch b.Kind {
	case "rule":
		k.rule(p, b)
	case "scenario":
		if _, ok := b.Props.Get("proof"); !ok {
			k.add("S4", p.ID, b.ID, b.At, fmt.Sprintf("scenario %q has no proof: shown as pending", b.ID))
		}
	case "link":
		if _, ok := b.Props.Get("proof"); !ok && textOf(b.Props, "right") == "refused" {
			k.add("S3", p.ID, b.ID, b.At, fmt.Sprintf("link %q is refused and names no proof: nothing would catch it", b.ID))
		}
	case "component":
		k.component(p, b)
	case "table":
		k.table(p, b)
	case "page":
		k.subPage(p, b, placed)
	case "figure":
		k.figure(p, b)
	}
}

func (k *checker) rule(p *Page, b *Block) {
	force := textOf(b.Props, "force")
	_, proof := b.Props.Get("proof")
	_, readAt := b.Props.Get("read_at")
	if (force == "hard" || force == "hardened") && !proof {
		k.add("H6", p.ID, b.ID, b.At, fmt.Sprintf("rule %q is %s and names no proof: write proof, or make it soft", b.ID, force))
	}
	if (force == "soft" || force == "hardened") && !readAt {
		k.add("H6", p.ID, b.ID, b.At, fmt.Sprintf("rule %q is %s and names no read_at: write where an agent reads it", b.ID, force))
	}
	if v, ok := b.Props.Get("precedents"); ok && force == "soft" && len(v.List) >= 2 {
		k.add("S2", p.ID, b.ID, b.Props.at("precedents", b.At), fmt.Sprintf("rule %q is soft and broke %d times: it asks for a proof", b.ID, len(v.List)))
	}
	if v, ok := b.Props.Get("waiver"); ok {
		k.add("S29", p.ID, b.ID, b.Props.at("waiver", b.At), fmt.Sprintf("rule %q asks to be waived: %s", b.ID, text(v)))
		k.needsYou[b.Address] = true
	}
}

// invariant: a component is made of lower levels only (H14), its levels
// lowest first.
func (k *checker) component(p *Page, b *Block) {
	level := slices.Index(levels(), textOf(b.Props, "level"))
	v, _ := b.Props.Get("made_of")
	for _, r := range refsIn(v) {
		t, ok := k.targets[r.Text]
		if !ok || t.block == nil || t.block.Kind != "component" {
			continue
		}
		if other := textOf(t.block.Props, "level"); slices.Index(levels(), other) >= level {
			k.add("H14", p.ID, b.ID, whereOf(r, b.At), fmt.Sprintf("component %q is a %s made of %q, a %s: a component is made of lower levels only", b.ID, textOf(b.Props, "level"), t.block.ID, other))
		}
	}
}

func (k *checker) figure(p *Page, b *Block) {
	if v, ok := b.Props.Get("form"); ok && !slices.Contains(forms(), text(v)) {
		k.add("H26", p.ID, b.ID, b.Props.at("form", b.At), fmt.Sprintf("form: %s does not exist: %s", text(v), orList(forms())))
	}
	if v, ok := b.Props.Get("caption"); ok {
		if n := len([]rune(text(v))); n > lineCap {
			k.add("H28", p.ID, b.ID, b.Props.at("caption", b.At), fmt.Sprintf("the caption is %d characters, %d at most", n, lineCap))
		}
	}
}

// invariant: a sub-page is a page of the corpus (H23), a child of this one
// (H24): under its folder, or naming it in its `needs`; and placed once (H25).
func (k *checker) subPage(p *Page, b *Block, placed map[string]bool) {
	child := b.ID
	c, ok := k.byID[child]
	switch {
	case !ok:
		k.add("H23", p.ID, b.ID, b.At, fmt.Sprintf("sub-page %q is no page", child))
	case !k.childOf(c, p):
		k.add("H24", p.ID, b.ID, b.At, fmt.Sprintf("%q is not a child of this page: a page that is not yours is linked, not placed", child))
	}
	if placed[child] {
		k.add("H25", p.ID, b.ID, b.At, fmt.Sprintf("%q is placed twice in this page", child))
	}
	placed[child] = true
}

// value: each page's children by childOf, found once for the corpus: a page
// is the child of each page its id lies under and of each it needs.
func (k *checker) childrenOf() map[string][]string {
	kids := map[string][]string{}
	for _, c := range k.pages {
		parents := map[string]bool{}
		for i := range len(c.ID) {
			if c.ID[i] != '/' {
				continue
			}
			if _, ok := k.byID[c.ID[:i]]; ok {
				parents[c.ID[:i]] = true
			}
		}
		if fl, ok := k.front(c, "needs"); ok {
			for _, n := range frontList(fl.Value) {
				if _, ok := k.byID[n]; ok && n != c.ID {
					parents[n] = true
				}
			}
		}
		for p := range parents {
			kids[p] = append(kids[p], c.ID)
		}
	}
	return kids
}

func (k *checker) childOf(c, p *Page) bool {
	if strings.HasPrefix(c.ID, p.ID+"/") {
		return true
	}
	fl, ok := k.front(c, "needs")
	return ok && slices.Contains(frontList(fl.Value), p.ID)
}

// invariant: a page of more than seven blocks with children places one of
// them (S18).
func (k *checker) children(p *Page, placed map[string]bool) {
	kids := k.kids[p.ID]
	if len(kids) == 0 || len(k.blocks[p.ID]) <= pageBlocks {
		return
	}
	for _, id := range kids {
		if placed[id] {
			return
		}
	}
	k.add("S18", p.ID, "", pageAt(p), fmt.Sprintf("page %q shows none of its %d children", p.ID, len(kids)))
}
