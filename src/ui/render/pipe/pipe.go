package pipe

import (
	"errors"
	"strings"

	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ui/atom"
	"github.com/jterrazz/studio-station/src/ui/token"
)

var ErrPending = errors.New("a page with a pending part reached the pipe")

func Render(p page.Page) ([]byte, error) {
	if p.State == page.Loading {
		return nil, ErrPending
	}
	if p.Error != nil {
		return nil, nil
	}
	var b strings.Builder
	for _, s := range p.Sections {
		section(&b, s)
	}
	return []byte(b.String()), nil
}

func section(b *strings.Builder, s page.Section) {
	switch s.Role {
	case page.Actions, page.Summary, page.Lifted:
		return
	case page.Banner:
	case page.Plain:
		if s.PipeGroups {
			break
		}
		fields := []string{"# " + s.Title}
		if s.PipeCounts {
			for _, t := range s.Counts {
				fields = append(fields, atom.Count(t))
			}
		}
		line(b, fields)
	}
	cells := make([]bool, max(len(s.Columns)-1, 0))
	for i := range cells {
		cells[i] = s.Columns[i+1].Show != page.EyeOnly && s.Columns[i+1].Show != page.DetailOnly
	}
	for _, it := range s.Items {
		if s.PipeGroups && it.Group != nil && it.Group.Heading() {
			line(b, []string{"# " + it.Group.Head.Label})
			for _, c := range it.Group.Items {
				item(b, c, cells)
			}
			continue
		}
		item(b, it, cells)
	}
}

func item(b *strings.Builder, it page.Item, cells []bool) {
	switch {
	case it.Row != nil:
		row(b, *it.Row, cells)
	case it.Group != nil:
		if !it.Group.Heading() {
			row(b, it.Group.Head, cells)
		}
		for _, c := range it.Group.Items {
			// invariant: a group's empty child is for the eye: the head's own
			// fields already say it holds nothing.
			if c.Empty == nil {
				item(b, c, cells)
			}
		}
	case it.Fact != nil:
		line(b, []string{it.Fact.Key, atom.Plain(it.Fact.Value)})
	case it.Empty != nil:
		line(b, []string{"", "none"})
	case it.Error != nil:
		line(b, []string{token.MarkBroken, "error", it.Error.Message + token.Sep + it.Error.Hint})
	case it.Note != nil:
		line(b, []string{"", it.Note.Text})
	}
}

func row(b *strings.Builder, r page.Row, cells []bool) {
	fields := make([]string, 0, len(cells)+3)
	fields = append(fields, atom.Glyph(r.Mark), r.Label)
	for i := range max(len(cells), len(r.Cells)) {
		if i < len(cells) && !cells[i] {
			continue
		}
		v := page.None()
		if i < len(r.Cells) {
			v = r.Cells[i]
		}
		fields = append(fields, atom.Plain(v))
	}
	var last []string
	if s := atom.Plain(atom.Detail(r)); s != "" {
		last = append(last, s)
	}
	if r.Hint != nil {
		last = append(last, r.Hint.Command)
	}
	fields = append(fields, strings.Join(last, token.Sep))
	line(b, fields)
}

func line(b *strings.Builder, fields []string) {
	for len(fields) > 1 && fields[len(fields)-1] == "" {
		fields = fields[:len(fields)-1]
	}
	b.WriteString(strings.Join(fields, "\t"))
	b.WriteByte('\n')
}
