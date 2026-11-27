package wiki

import (
	"cmp"
	"slices"
	"strings"
)

func lede(head PageHead, paragraph []Paragraph) ServedLede {
	if head.Decision != nil {
		return ServedLede{Ranks: []Detail{DetailLine, DetailLine, DetailFull, DetailFull}}
	}
	first, more := firstSentenceOf(paragraph)
	return ServedLede{Ranks: openRanks(), First: first, More: more}
}

// invariant: a group is mirrored by the first table of its page whose rows
// are exactly its blocks; from the first level at which that table shows its
// rows (a card), the group shows its blocks' words and the table its rows.
func mirror(sections []ServedSection) {
	var groups []*ServedGroup
	var tables []ServedItem
	for _, sec := range sections {
		for _, it := range sec.Items {
			if it.Group != nil {
				groups = append(groups, it.Group)
			}
			if it.Block != nil && it.Block.Table != nil && len(it.Block.Table.Rows) > 0 {
				tables = append(tables, it)
			}
		}
	}
	for _, g := range groups {
		var kids []string
		for _, it := range g.Items {
			if it.Block != nil {
				kids = append(kids, it.Block.Address)
			}
		}
		for _, t := range tables {
			rows := t.Block.Table.Rows
			if t.Block.Table.Mirrors != "" || len(rows) != len(kids) || slices.ContainsFunc(rows, func(r ServedRow) bool { return !slices.Contains(kids, r.Address) }) {
				continue
			}
			g.MirroredBy, t.Block.Table.Mirrors = t.Block.ID, g.ID
			if at := slices.IndexFunc(t.Ranks, func(d Detail) bool { return d.rank() >= DetailCard.rank() }); at >= 0 {
				g.MirroredFrom = Details()[at]
			}
			break
		}
	}
}

// invariant: a level's reading time counts the words it shows: the title,
// the lede at its face, each heading, and each block, group and prose at its
// face; a level at which all of them, and the lede, take the faces of the
// level before it shows the same.
func readingOf(s Served) []ServedReading {
	words := make([]int, len(Details()))
	same := make([]bool, len(Details()))
	var before string
	for level := range Details() {
		var faces strings.Builder
		n := wordCount(cmp.Or(strings.TrimSpace(s.Page.Number+" "+s.Page.Title), s.Page.Word))
		n += ledeWords(s, level)
		faces.WriteString(string(s.Lede.Ranks[level]))
		for _, sec := range s.Sections {
			n += wordCount(sec.Heading)
			for _, it := range sec.Items {
				n += itemWords(it, level, false, &faces)
			}
		}
		words[level] = n
		same[level] = level > 0 && faces.String() == before
		before = faces.String()
	}
	return readings(words, same)
}

func ledeWords(s Served, level int) int {
	face := s.Lede.Ranks[level]
	if d := s.Page.Decision; d != nil {
		return decisionWords(d, face)
	}
	if face == DetailWord {
		return wordCount(runsText(s.Lede.First))
	}
	return wordCount(paragraphsText(s.Paragraph))
}

func itemWords(it ServedItem, level int, folded bool, faces *strings.Builder) int {
	face := it.Ranks[level]
	if folded {
		face = DetailWord
	}
	faces.WriteString(" " + string(face))
	switch {
	case it.Prose != nil:
		if face.rank() < DetailCard.rank() {
			return wordCount(runsText(it.Prose.First))
		}
		return wordCount(paragraphsText(it.Prose.Paragraphs))
	case it.Group != nil:
		g := it.Group
		rows := face == DetailWord || (g.MirroredFrom != "" && level >= g.MirroredFrom.rank())
		if rows {
			faces.WriteString(":rows")
		}
		n := wordCount(g.Title)
		if !rows && face.rank() >= DetailCard.rank() {
			n += wordCount(g.Lead)
		}
		for _, kid := range g.Items {
			n += itemWords(kid, level, rows, faces)
		}
		return n
	case it.Block != nil:
		return blockWords(it.Block, face, Details()[level])
	}
	return 0
}

// invariant: a block counts at a face the most words of that face and of
// every face before it, so no wiki.yaml template, a line written longer
// than its card, makes a level read faster than the level before it.
func blockWords(b *ServedBlock, face, level Detail) int {
	n := wordCount(b.Word)
	for _, d := range Details()[1 : face.rank()+1] {
		n = max(n, faceWords(b, d))
	}
	if face == DetailWord {
		return n
	}
	for _, f := range b.Finding {
		if level.rank() >= f.From.rank() {
			n += wordCount(f.Says)
		}
	}
	return n
}

// invariant: a block's line shows its line, a table its title, its count and
// its rows' words, a figure its caption, a decision its record's number,
// title, status and date; its card its kind's title, its word and its
// fields, a table its rows' words under its title and filter, a figure its
// parts and what it shows; its full face its card, a table its columns and
// cells too.
func faceWords(b *ServedBlock, face Detail) int {
	if d := b.Decision; d != nil && d.Resolves {
		return decisionWords(d, face)
	}
	switch {
	case b.Table != nil:
		t := b.Table
		n := rowWords(t)
		if face == DetailLine {
			return n + wordCount(t.Title) + 1
		}
		n += 1 + wordCount(t.Title, t.Where)
		if face == DetailFull {
			n += wordCount(t.Columns...)
			for _, r := range t.Rows {
				n += fieldWords(r.Cells, false)
			}
		}
		return n
	case b.Figure != nil:
		f := b.Figure
		if face == DetailLine {
			return wordCount(f.Caption)
		}
		n := wordCount(b.Word, f.Caption, f.Shows.Word)
		for _, p := range f.Parts {
			n += wordCount(p.Name)
		}
		return n
	}
	if face == DetailLine {
		return wordCount(cmp.Or(b.Line, b.Word))
	}
	return wordCount(b.Title, b.Word) + fieldWords(b.Fields, true)
}

func decisionWords(d *ServedDecision, face Detail) int {
	n := wordCount(d.Number, d.Title, d.Status, d.Date)
	if face == DetailFull {
		n += wordCount(d.Decides) + wordCount(d.AmendedBy...) + wordCount(d.Amends...)
	}
	return n
}

func rowWords(t *ServedTable) int {
	n := 0
	for _, r := range t.Rows {
		n += wordCount(r.Word)
	}
	return n
}

func fieldWords(fields []ServedField, named bool) int {
	n := 0
	for _, f := range fields {
		if named {
			n += wordCount(strings.ReplaceAll(f.Name, "_", " "))
		}
		n += wordCount(f.Text)
		for _, r := range f.Refs {
			n += wordCount(r.Word)
		}
	}
	return n
}
