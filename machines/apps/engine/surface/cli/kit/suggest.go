package kit

import (
	"cmp"
	"slices"
	"strings"
)

// value: the most words a suggestion names.
const suggestionsMost = 3

func Suggest(word string, known []string) []string {
	// invariant: a word the tree does not know is answered by the words it
	// knows within two edits (cobra's SuggestionsMinimumDistance), a short
	// word within one, a very short one by none: the nearest first, then by name, never the word
	// itself.
	word = strings.ToLower(word)
	limit := min(2, (len([]rune(word))-1)/2)
	type near struct {
		name string
		by   int
	}
	var found []near
	for _, name := range slices.Compact(slices.Sorted(slices.Values(known))) {
		if by := edits(word, strings.ToLower(name)); by > 0 && by <= limit {
			found = append(found, near{name, by})
		}
	}
	slices.SortStableFunc(found, func(a, b near) int { return cmp.Compare(a.by, b.by) })
	out := make([]string, 0, suggestionsMost)
	for _, n := range found[:min(len(found), suggestionsMost)] {
		out = append(out, n.name)
	}
	return out
}

func edits(a, b string) int {
	x, y := []rune(a), []rune(b)
	row := make([]int, len(y)+1)
	for j := range row {
		row[j] = j
	}
	for i := 1; i <= len(x); i++ {
		diagonal := row[0]
		row[0] = i
		for j := 1; j <= len(y); j++ {
			above := row[j]
			cost := 1
			if x[i-1] == y[j-1] {
				cost = 0
			}
			row[j] = min(row[j]+1, row[j-1]+1, diagonal+cost)
			diagonal = above
		}
	}
	return row[len(y)]
}
