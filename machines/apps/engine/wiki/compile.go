package wiki

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// value: what the line before a line was, which decides whether a `<` there
// opens a tag.
type before int

const (
	afterStart before = iota
	afterBlank
	afterHeading
	afterTag
	afterText
)

type fence struct {
	char byte
	size int
}

type compiler struct {
	page    *Page
	schema  Schema
	x       lineIndex
	lines   []string
	section *Section
	group   *Group
	refused int
	prose   []string
	proseAt int
	// value: whether a line of prose since the last flush holds a word.
	worded   bool
	anchors  map[string]int
	intro    bool
	findings []Finding
	// value: the headings past MaxSections, read as prose, and the first.
	folded   int
	foldedAt At
}

func Compile(id, source string, schema Schema) Page {
	// invariant: a pure function of its arguments: it reads no file, runs
	// nothing, and the same source gives the same Page. A page with a finding
	// still compiles as far as it reads; its findings say what was refused.
	if len(source) > MaxPageBytes {
		return tooLong(id, int64(len(source)), schema)
	}
	source = strings.ReplaceAll(source, "\r\n", "\n")
	id, inID := audible(id)
	source, inSource := audible(source)
	c := &compiler{
		page:    &Page{ID: id, Sections: []Section{{Items: []Item{}}}},
		schema:  schema,
		x:       indexLines(source),
		lines:   strings.Split(source, "\n"),
		anchors: map[string]int{},
		intro:   true,
	}
	c.section = &c.page.Sections[0]
	c.run()
	c.page.Findings = capped(c.findings, nil)
	if n := inID + inSource; n > 0 {
		c.page.Findings = append([]Finding{silenced(id, n)}, c.page.Findings...)
	}
	c.faces()
	return *c.page
}

func (c *compiler) run() {
	i := c.frontmatter()
	prev := afterStart
	var open *fence
	for i < len(c.lines) {
		line := c.lines[i]
		n := i + 1
		if open != nil {
			if closesFence(line, *open) {
				open = nil
			}
			c.text(n, line)
			prev = afterText
			i++
			continue
		}
		if f, ok := opensFence(line); ok {
			open = &f
			c.text(n, line)
			prev = afterText
			i++
			continue
		}
		switch {
		case strings.TrimSpace(line) == "":
			c.text(n, line)
			prev = afterBlank
		case strings.TrimRight(line, " \t") == "</Group>":
			c.flush()
			c.closeGroup(n)
			prev = afterTag
		case len(line) > 1 && line[0] == '<' && isUpper(line[1]) && prev != afterText:
			c.flush()
			i = c.tag(i)
			prev = afterTag
			continue
		case c.heading(n, line):
			prev = afterHeading
		default:
			c.text(n, line)
			prev = afterText
		}
		i++
	}
	c.flush()
	if c.folded > 0 {
		c.refuse("S31", "", c.foldedAt, fmt.Sprintf("the page keeps its first %d sections: %d more headings are read as the prose of the last", MaxSections, c.folded))
	}
	if c.group != nil {
		c.refuse("H40", c.group.ID, c.group.At, groupName(c.group)+" is never closed: end it with </Group>")
		c.group.Span.To = c.x.at(max(0, len(c.x.src)-1)).Line
		c.group = nil
	}
}

func (c *compiler) refuse(rule, block string, at At, says string) {
	c.findings = append(c.findings, finding(rule, c.page.ID, block, at, says))
}

// value: the first line after the frontmatter, 0 when the page has none.
func (c *compiler) frontmatter() int {
	if len(c.lines) == 0 || strings.TrimRight(c.lines[0], " \t") != "---" {
		return 0
	}
	for i := 1; i < len(c.lines); i++ {
		if strings.TrimRight(c.lines[i], " \t") == "---" {
			front := &Frontmatter{Span: Span{From: 1, To: i + 1}, Lines: []FrontLine{}}
			for j := 1; j < i; j++ {
				key, value, ok := strings.Cut(c.lines[j], ":")
				key = strings.TrimSpace(key)
				if !ok || key == "" {
					continue
				}
				value = strings.Trim(strings.TrimSpace(value), `"'`)
				front.Lines = append(front.Lines, FrontLine{Key: key, Value: value, Line: j + 1})
				if key == "kind" {
					c.page.Kind = value
				}
			}
			c.page.Front = front
			return i + 1
		}
	}
	return 0
}

func opensFence(line string) (fence, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || len(trimmed) < 3 || (trimmed[0] != '`' && trimmed[0] != '~') {
		return fence{}, false
	}
	size := 0
	for size < len(trimmed) && trimmed[size] == trimmed[0] {
		size++
	}
	return fence{char: trimmed[0], size: size}, size >= 3
}

func closesFence(line string, f fence) bool {
	trimmed := strings.TrimSpace(line)
	return len(trimmed) >= f.size && strings.Trim(trimmed, string(f.char)) == ""
}

func (c *compiler) text(n int, line string) {
	if len(c.prose) == 0 {
		c.proseAt = n
	}
	c.prose = append(c.prose, line)
	c.worded = c.worded || strings.TrimSpace(line) != ""
}

func (c *compiler) flush() {
	lines := c.prose
	from := c.proseAt
	c.prose, c.worded = nil, false
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
		from++
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return
	}
	c.add(&Prose{T: "prose", Markdown: strings.Join(lines, "\n"), Span: Span{From: from, To: from + len(lines) - 1}})
}

func (c *compiler) add(it Item) {
	if c.group != nil {
		c.group.Items = append(c.group.Items, it)
		return
	}
	c.section.Items = append(c.section.Items, it)
}

func (c *compiler) heading(n int, line string) bool {
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || (level < len(line) && line[level] != ' ' && line[level] != '\t') {
		return false
	}
	text := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(line[level:]), "#"))
	if level == 1 && c.page.Title == nil && c.intro && len(c.section.Items) == 0 && !c.worded {
		c.prose = nil
		c.page.Title = &Span{From: n, To: n}
		c.page.Faces.Word = text
		return true
	}
	if level != 2 && level != 3 {
		return false
	}
	if len(c.page.Sections) >= MaxSections {
		if c.folded == 0 {
			c.foldedAt = At{Line: n, Col: 1}
		}
		c.folded++
		return false
	}
	c.flush()
	if c.group != nil {
		c.refuse("H40", c.group.ID, At{Line: n, Col: 1}, fmt.Sprintf("%s is still open at %q: close it before the heading", groupName(c.group), strings.TrimSpace(line)))
		c.group.Span.To = n - 1
		c.group, c.refused = nil, 0
	}
	c.intro = false
	c.page.Sections = append(c.page.Sections, Section{Heading: &Heading{Level: level, Text: text, Anchor: c.anchor(text), At: At{Line: n, Col: 1}}, Items: []Item{}})
	c.section = &c.page.Sections[len(c.page.Sections)-1]
	return true
}

// value: the fragment GitHub gives a heading: lower case, letters, digits,
// spaces, `-` and `_` kept, each space a `-`, a repeat suffixed `-1`, `-2`.
func (c *compiler) anchor(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	slug := b.String()
	seen := c.anchors[slug]
	c.anchors[slug] = seen + 1
	if seen > 0 {
		return slug + "-" + strconv.Itoa(seen)
	}
	return slug
}

func groupName(g *Group) string {
	if g.ID != "" {
		return fmt.Sprintf("<Group %q>", g.ID)
	}
	return fmt.Sprintf("<Group at line %d>", g.At.Line)
}

func (c *compiler) closeGroup(n int) {
	switch {
	case c.refused > 0:
		c.refused--
	case c.group != nil:
		c.group.Span.To = n
		c.group = nil
	default:
		c.refuse("H40", "", At{Line: n, Col: 1}, "</Group> closes no group")
	}
}

// value: the index of the line after the tag that starts on line i.
func (c *compiler) tag(i int) int {
	t, r := parseTag(c.x, c.x.starts[i])
	if r == nil {
		rest, _, _ := strings.Cut(c.x.src[t.end:], "\n")
		if strings.TrimSpace(rest) != "" {
			r = &refusal{rule: "H38", at: c.x.at(t.end), says: fmt.Sprintf("<%s at line %d: text follows the tag on its last line: a tag ends its line", t.name, t.at.Line)}
		}
	}
	if r != nil {
		c.refuse(r.rule, idOf(t.props), r.at, r.says)
		return c.resume(i, r)
	}
	last := c.x.at(t.end - 1).Line
	span := Span{From: i + 1, To: last}
	if t.name == "Group" {
		c.openGroup(t, span)
		return last
	}
	if t.opens {
		c.refuse("H38", idOf(t.props), t.at, fmt.Sprintf("<%s at line %d ends with >: a block ends with />, only a Group holds what follows it", t.name, t.at.Line))
		return last
	}
	if b := c.block(t, span); b != nil {
		c.add(b)
	}
	return last
}

// value: where reading goes on after a refused tag: the first blank line
// after the line the refusal names, or after the tag's opening line when the
// tag is never closed.
func (c *compiler) resume(i int, r *refusal) int {
	from := max(i, r.at.Line-1)
	for j := from + 1; j < len(c.lines); j++ {
		if strings.TrimSpace(c.lines[j]) == "" {
			return j
		}
	}
	return len(c.lines)
}

func idOf(props Props) string {
	if v, ok := props.Get("id"); ok && v.Kind == ValueString {
		return v.Text
	}
	return ""
}

func (c *compiler) openGroup(t tag, span Span) {
	if c.group != nil {
		c.refuse("H40", idOf(t.props), t.at, fmt.Sprintf("<Group %q> opens inside %s: a group never holds a group", idOf(t.props), groupName(c.group)))
		if t.opens {
			c.refused++
		}
		return
	}
	kind, _ := c.schema.kind("group")
	c.attributes(kind, t)
	g := &Group{T: "group", ID: idOf(t.props), Title: textOf(t.props, "title"), Intent: intentOf(textOf(t.props, "intent")), Props: t.props, Items: []Item{}, At: t.at, Tag: span, Span: span}
	if v, ok := t.props.Get("fold"); ok && v.Kind == ValueBool {
		g.Fold = v.Bool
	}
	if g.Props == nil {
		g.Props = Props{}
	}
	c.add(g)
	if t.opens {
		c.group = g
	}
}

func textOf(props Props, name string) string {
	v, _ := props.Get(name)
	return text(v)
}

func text(v Value) string {
	switch v.Kind {
	case ValueString, ValueNumber, ValueRef, ValueState:
		return v.Text
	case ValueBool:
		return strconv.FormatBool(v.Bool)
	case ValueList:
		parts := make([]string, 0, len(v.List))
		for _, it := range v.List {
			parts = append(parts, text(it))
		}
		return strings.Join(parts, ", ")
	default:
		return ""
	}
}

func (c *compiler) attributes(kind Kind, t tag) {
	for _, prop := range t.props {
		switch {
		case kind.takes(prop.Name):
		case kind.ID == "table" && prop.Name == "rows":
			c.refuse("H22", idOf(t.props), prop.At, fmt.Sprintf("a table is not written row by row: a row is a block of kind %s; write the block, the table finds it", textOf(t.props, "of")))
		default:
			c.refuse("H41", idOf(t.props), prop.At, fmt.Sprintf("%q is not a field of %s", prop.Name, kind.ID))
		}
	}
}

func (c *compiler) block(t tag, span Span) *Block {
	kind, ok := c.schema.kind(kindOf(t.name))
	if !ok {
		says := fmt.Sprintf("unknown kind %q", t.name)
		if near := nearest(t.name, c.schema.tagNames()); near != "" {
			says += fmt.Sprintf(" (did you mean %s?)", near)
		}
		c.refuse("H1", idOf(t.props), t.at, says)
		return nil
	}
	c.attributes(kind, t)
	b := &Block{T: "block", ID: idOf(t.props), Kind: kind.ID, Props: t.props, At: t.at, Span: span}
	if b.Props == nil {
		b.Props = Props{}
	}
	switch {
	case kind.ID == "page":
		b.Address = b.ID
	case b.ID != "":
		b.Address = c.page.ID + "#" + b.ID
	}
	switch kind.ID {
	case "table":
		c.keepFirst(b, "columns", MaxTableColumns, "columns")
	case "figure":
		c.keepFirst(b, "parts", MaxTableRows, "parts")
	}
	b.Faces = facesOf(kind, b)
	b.View = viewOf(kind.ID, b.Props)
	return b
}

// invariant: a list a block declares past its limit keeps its first n
// entries; the rest is dropped before anything reads it, and one S31 says
// how many.
func (c *compiler) keepFirst(b *Block, field string, n int, what string) {
	for i := range b.Props {
		v := &b.Props[i].Value
		if b.Props[i].Name != field || v.Kind != ValueList || len(v.List) <= n {
			continue
		}
		c.refuse("S31", b.ID, b.At, fmt.Sprintf("%s %q keeps its first %d %s, and drops %d more", b.Kind, b.ID, n, what, len(v.List)-n))
		v.List = slices.Clone(v.List[:n])
	}
}

func facesOf(kind Kind, b *Block) BlockFaces {
	word := kind.Faces.Word
	if word == "" {
		word = "{id}"
	}
	f := BlockFaces{Word: render(word, b), Line: render(kind.Faces.Line, b)}
	switch {
	case textOf(b.Props, "icon") != "":
		f.Icon = textOf(b.Props, "icon")
	case kind.Faces.Icon != "":
		f.Icon = render(kind.Faces.Icon, b)
	case kind.Icon != "":
		f.Icon = kind.Icon
	default:
		f.Icon = kind.ID
	}
	return f
}

// invariant: a Face template is filled from the block's own fields; a dotted
// name (`{adr.number}`) reads outside the page and renders empty here.
func render(template string, b *Block) string {
	var out strings.Builder
	for {
		open := strings.IndexByte(template, '{')
		if open < 0 {
			out.WriteString(template)
			return out.String()
		}
		end := strings.IndexByte(template[open:], '}')
		if end < 0 {
			out.WriteString(template)
			return out.String()
		}
		out.WriteString(template[:open])
		name := template[open+1 : open+end]
		if name == "id" {
			out.WriteString(b.ID)
		} else if !strings.Contains(name, ".") {
			out.WriteString(textOf(b.Props, name))
		}
		template = template[open+end+1:]
	}
}

func viewOf(kind string, props Props) View {
	switch kind {
	case "table":
		v := &TableView{Of: textOf(props, "of"), In: textOf(props, "in"), Columns: []string{}, Sort: textOf(props, "sort")}
		if cols, ok := props.Get("columns"); ok {
			for _, col := range cols.List {
				v.Columns = append(v.Columns, text(col))
			}
		}
		if where, ok := props.Get("where"); ok && where.Kind == ValueQuery {
			v.Where = where.Query
		}
		return v
	case "page":
		face := textOf(props, "face")
		if face == "" {
			face = "line"
		}
		return &SubPageView{Child: textOf(props, "id"), Face: face}
	case "figure":
		v := &FigureView{Form: textOf(props, "form"), Shows: textOf(props, "shows"), Caption: textOf(props, "caption"), Parts: []FigurePart{}}
		if parts, ok := props.Get("parts"); ok {
			for _, part := range parts.List {
				member := func(key string) string {
					for _, m := range part.Members {
						if m.Key == key {
							return text(m.Value)
						}
					}
					return ""
				}
				n, _ := strconv.Atoi(member("n"))
				v.Parts = append(v.Parts, FigurePart{Name: member("name"), To: member("to"), N: n, At: member("at")})
			}
		}
		return v
	default:
		return nil
	}
}

// value: the name within two edits of tag, the first in order on a tie;
// empty when none is that close.
// invariant: a name whose length is as far from the tag's as the best
// distance yet is never compared, and a comparison stops once every
// alignment is that far, so a corpus of thousands of addresses suggests
// in time linear in them.
func nearest(tag string, names []string) string {
	best, score := "", 3
	low := strings.ToLower(tag)
	var rows [2][]int
	for _, n := range names {
		name := strings.ToLower(n)
		if abs(len(name)-len(low)) >= score {
			continue
		}
		if d := distance(low, name, score, &rows); d < score {
			best, score = n, d
		}
	}
	return best
}

func abs(n int) int { return max(n, -n) }

// value: the edit distance of a and b, or limit when it is limit or more.
func distance(a, b string, limit int, rows *[2][]int) int {
	prev, cur := rows[0][:0], rows[1][:0]
	for j := range len(b) + 1 {
		prev = append(prev, j)
		cur = append(cur, 0)
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		least := i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			least = min(least, cur[j])
		}
		if least >= limit {
			rows[0], rows[1] = prev, cur
			return limit
		}
		prev, cur = cur, prev
	}
	rows[0], rows[1] = prev, cur
	return min(prev[len(b)], limit)
}

// invariant: a page's word Face is its `# title`, its paragraph Face the
// first paragraph before any heading or tag, its line Face that paragraph's
// first sentence; frontmatter `word`, `line` and `icon` stand above them.
func (c *compiler) faces() {
	f := &c.page.Faces
	if items := c.page.Sections[0].Items; len(items) > 0 {
		if p, ok := items[0].(*Prose); ok {
			para, _, _ := strings.Cut(p.Markdown, "\n\n")
			f.Paragraph = strings.Join(strings.Fields(para), " ")
			f.Line = firstSentence(f.Paragraph)
		}
	}
	if c.page.Front == nil {
		return
	}
	for _, fl := range c.page.Front.Lines {
		switch fl.Key {
		case "word":
			f.Word = fl.Value
		case "line":
			f.Line = fl.Value
		case "icon":
			f.Icon = fl.Value
		}
	}
}

func firstSentence(s string) string {
	for i := range len(s) {
		if strings.IndexByte(".!?", s[i]) >= 0 && (i+1 == len(s) || s[i+1] == ' ') {
			return s[:i+1]
		}
	}
	return s
}
