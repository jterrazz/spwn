package wiki

import (
	"cmp"
	"slices"
)

// invariant: what the reader is asked to look at reads in one order wherever
// it is listed: what needs the owner first, then what is broken, then the
// remarks, then the ambient findings; the order they were found within each.
func rank(f ServedFinding) int {
	switch {
	case f.Level == LevelNeedsYou:
		return 0
	case f.Hard:
		return 1
	case f.Level == LevelRemark:
		return 2
	}
	return 3
}

func ranked(findings []ServedFinding) []ServedFinding {
	slices.SortStableFunc(findings, func(a, b ServedFinding) int { return cmp.Compare(rank(a), rank(b)) })
	return findings
}

// value: every finding served on the page and on its blocks, groups folded
// or not, each with where the reader finds it: the block, or the group that
// holds it when the group is served folded.
func attention(page []ServedFinding, sections []ServedSection) []ServedFinding {
	out := slices.Clone(page)
	var walk func(items []ServedItem, folded string)
	walk = func(items []ServedItem, folded string) {
		for _, it := range items {
			if b := it.Block; b != nil {
				for _, f := range b.Finding {
					f.At = cmp.Or(folded, f.At)
					out = append(out, f)
				}
			}
			if g := it.Group; g != nil {
				at := folded
				if g.Folded != FoldNone && at == "" {
					at = g.Address
				}
				walk(g.Items, at)
			}
		}
	}
	for _, sec := range sections {
		walk(sec.Items, "")
	}
	return ranked(out)
}

func needsYou(findings []ServedFinding) int {
	n := 0
	for _, f := range findings {
		if f.Level == LevelNeedsYou {
			n++
		}
	}
	return n
}
