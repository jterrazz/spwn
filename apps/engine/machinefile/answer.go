package machinefile

import (
	"maps"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

// invariant: a plan, a run, an upgrade and a night leave their call as an
// answer, screened once and only here: an option's value withheld whole,
// the owner's lines, a block's text and every word a package, a run or a
// source said screened word by word. What the call carries out keeps its
// values; the answer alone is screened, before any surface or another Mac
// reads it.

func (p Plan) Answered() Plan {
	p.Lines = each(p.Lines, Line.answered)
	p.Extra = each(p.Extra, func(e ExtraLine) ExtraLine {
		e.Line = e.answered()
		return e
	})
	p.Blocks = each(p.Blocks, Block.answered)
	p.Scripts = each(p.Scripts, Script.answered)
	return p
}

func (a Applied) Answered() Applied {
	a.Plan = a.Plan.Answered()
	a.Done = each(a.Done, Done.answered)
	a.Removed = each(a.Removed, Done.answered)
	a.Ran = a.Ran.answered()
	a.Blocks = each(a.Blocks, func(b BlockDone) BlockDone {
		b.Block, b.Detail = b.answered(), screened(b.Detail)
		return b
	})
	return a
}

func (u Upgraded) Answered() Upgraded {
	u.Plan = u.Plan.Answered()
	u.Done = each(u.Done, Done.answered)
	u.Ran = u.Ran.answered()
	u.Unpulled = screened(u.Unpulled)
	return u
}

func (l Line) answered() Line {
	if l.Step.Values != nil {
		values := maps.Clone(l.Step.Values)
		for k, v := range values {
			shown, _ := withhold(string(v))
			values[k] = catalog.OptionValue(shown)
		}
		l.Step.Values = values
	}
	l.Step.From = screened(l.Step.From)
	l.Seen.Detail = screened(l.Seen.Detail)
	return l
}

func (d Done) answered() Done {
	d.Line, d.Detail = d.Line.answered(), screened(d.Detail)
	return d
}

func (r *Ran) answered() *Ran {
	if r == nil {
		return nil
	}
	out := *r
	out.Script, out.Detail = r.Script.answered(), screened(r.Detail)
	return &out
}

func (s Script) answered() Script {
	s.Lines = screened(s.Lines)
	return s
}

func (b Block) answered() Block {
	b.Zsh, b.Diff = screened(b.Zsh), screened(b.Diff)
	return b
}

func each[T any](in []T, answered func(T) T) []T {
	if in == nil {
		return nil
	}
	out := make([]T, len(in))
	for i, v := range in {
		out[i] = answered(v)
	}
	return out
}

func screened(text string) string {
	kept, _ := secret.Screen(text)
	return kept
}
