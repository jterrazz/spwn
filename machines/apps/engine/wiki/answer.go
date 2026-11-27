package wiki

import "unicode/utf8"

// invariant: a meter counts down the bytes an answer may still take, as an
// upper bound of what its JSON says: each string at its JSON cost, its bytes
// and its quotes, six for each byte JSON escapes, six for each byte of
// invalid UTF-8 (written as �) and for U+2028 and U+2029; and each
// object its keys. It stops counting a string once spent, so measuring a
// hostile entry costs at most the budget; and once spent it stays spent, so
// what is served is a prefix in reading order.
type meter struct{ left int }

func newMeter() *meter { return &meter{left: MaxAnswerBytes} }

func (m *meter) spent() bool { return m.left < 0 }

func (m *meter) object(keys int) { m.left -= 16 * keys }

func (m *meter) str(s string) {
	m.left -= len(s) + 2
	for i := 0; i < len(s) && m.left >= 0; {
		c := s[i]
		if c < utf8.RuneSelf {
			if c < 0x20 || c == '<' || c == '>' || c == '&' || c == '"' || c == '\\' {
				m.left -= 5
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			m.left -= 5
		case r == ' ' || r == ' ':
			m.left -= 3
		}
		i += size
	}
}

func (m *meter) floorHead(f *FloorHead) {
	m.object(3)
	m.str(f.Name)
	m.str(f.Question)
}

func (m *meter) strs(list []string) {
	for _, s := range list {
		if m.spent() {
			return
		}
		m.str(s)
	}
}

func (m *meter) runs(runs []Run) {
	for _, r := range runs {
		if m.spent() {
			return
		}
		m.object(6)
		m.str(r.Text)
		m.str(r.Link)
		m.str(r.Anchor)
	}
}

// value: a section's own bytes; its items are measured as they are built.
func (m *meter) section(s ServedSection) {
	m.object(7)
	m.strs([]string{s.Heading, s.Anchor, s.Label, s.Icon})
}

// value: an item's own bytes; a group's kids, a prose's paragraphs and a
// table's rows are measured as they are built, each on its own.
func (m *meter) item(it ServedItem) {
	m.object(5)
	switch {
	case it.Prose != nil:
		m.object(4)
		m.str(it.Prose.ID)
		m.runs(it.Prose.First)
	case it.Group != nil:
		g := it.Group
		m.object(14)
		m.strs([]string{g.ID, g.Address, g.Title, g.MirroredBy, g.Lead})
		m.strs(g.ReadBy)
		if g.OwnFloor != nil {
			m.floorHead(g.OwnFloor)
		}
		m.findings(g.Finding)
	case it.Block != nil:
		m.block(it.Block)
	}
}

func (m *meter) block(b *ServedBlock) {
	m.object(22)
	m.strs([]string{b.ID, b.Address, b.Kind, b.Title, b.Icon, b.Word, b.Line})
	if b.OwnFloor != nil {
		m.floorHead(b.OwnFloor)
	}
	for _, c := range b.Why {
		m.object(2)
		m.str(c.Name)
	}
	m.fields(b.Fields)
	m.findings(b.Finding)
	if b.Sub != nil {
		m.head(b.Sub)
	}
	if b.Decision != nil {
		m.decision(b.Decision)
	}
	if t := b.Table; t != nil {
		m.object(9)
		m.strs([]string{t.Of, t.Title, t.Where, t.Mirrors, t.In.Address, t.In.Word})
		m.strs(t.Columns)
	}
	if f := b.Figure; f != nil {
		m.object(7)
		m.strs([]string{f.Form, f.Caption, f.X, f.Y, f.Shows.Address, f.Shows.Word})
		for _, p := range f.Parts {
			if m.spent() {
				return
			}
			m.object(4)
			m.strs([]string{p.Name, p.To, p.At})
		}
	}
}

// value: a table's row, measured as the table builds it, so a table stops
// at the first row that does not fit; its block counts none of its rows.
func (m *meter) row(r ServedRow) {
	m.object(4)
	m.strs([]string{r.Address, r.Word, r.Icon})
	m.fields(r.Cells)
}

func (m *meter) fields(fields []ServedField) {
	for _, f := range fields {
		if m.spent() {
			return
		}
		m.object(4)
		m.str(f.Name)
		m.str(f.Text)
		for _, r := range f.Refs {
			if m.spent() {
				return
			}
			m.object(3)
			m.str(r.Address)
			m.str(r.Word)
		}
	}
}

func (m *meter) findings(findings []ServedFinding) {
	for _, f := range findings {
		if m.spent() {
			return
		}
		m.finding(f)
	}
}

func (m *meter) finding(f ServedFinding) {
	m.object(9)
	m.strs([]string{f.Rule, f.Says, f.At, f.Word})
}

// value: the prefix of findings the answer has room for, each measured as
// it is kept, and how many are left out.
func (m *meter) kept(findings []ServedFinding) (kept []ServedFinding, cut int) {
	for i, f := range findings {
		m.finding(f)
		if m.spent() {
			return findings[:i], len(findings) - i
		}
	}
	return findings, 0
}

func (m *meter) head(h *PageHead) {
	m.object(12)
	m.strs([]string{h.ID, h.Seat, h.Word, h.Line, h.Icon, h.Number, h.Title})
	if h.Decision != nil {
		m.decision(h.Decision)
	}
}

func (m *meter) decision(d *ServedDecision) {
	m.object(10)
	m.strs([]string{d.Page, d.Number, d.Title, d.Status, d.Date, d.Decides})
	m.strs(d.AmendedBy)
	m.strs(d.Amends)
}

func (m *meter) icon(i Icon) {
	m.str(i.Name)
	m.strs(i.Rows)
}

func (m *meter) floor(f MapFloor) {
	m.object(4)
	m.strs([]string{f.Name, f.Question})
	m.strs(f.LabelParts)
}

// value: a node's own bytes, its id counted twice, as its parent's kids
// name it, and its parent's id once.
func (m *meter) node(n MapNode) {
	m.object(20)
	m.strs([]string{n.ID, n.ID, n.Ref, n.Label, n.Icon, n.Home})
	if n.Parent != nil {
		m.str(*n.Parent)
	}
	m.strs(n.TitleParts)
	m.strs(n.KickerParts)
	m.strs(n.LineParts)
	m.object(MaxMarks)
}
