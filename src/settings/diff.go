package settings

import (
	"strconv"
	"strings"
)

const diffContext = 2

func UnifiedDiff(from, to, fromName, toName string) string {
	// value: the unified diff of two texts, the first's lines "-" and the
	// second's "+", with two lines of context; empty when they are the same.
	// A setting draws station's text first, then the machine's.
	if from == to {
		return ""
	}
	a, b := lines(from), lines(to)
	ops := script(a, b)
	var out strings.Builder
	out.WriteString("--- " + fromName + "\n+++ " + toName + "\n")
	for start := 0; start < len(ops); {
		for start < len(ops) && ops[start].kind == ' ' {
			start++
		}
		if start == len(ops) {
			break
		}
		from := max(0, start-diffContext)
		end := start
		for quiet := 0; end < len(ops) && quiet <= 2*diffContext; end++ {
			if ops[end].kind == ' ' {
				quiet++
			} else {
				quiet = 0
			}
		}
		end = min(len(ops), end)
		for end > start && ops[end-1].kind == ' ' && trailing(ops[start:end]) > diffContext {
			end--
		}
		hunk := ops[from:end]
		ai, bi, an, bn := ops[from].a+1, ops[from].b+1, 0, 0
		for _, o := range hunk {
			if o.kind != '+' {
				an++
			}
			if o.kind != '-' {
				bn++
			}
		}
		out.WriteString("@@ -" + strconv.Itoa(ai) + "," + strconv.Itoa(an) + " +" + strconv.Itoa(bi) + "," + strconv.Itoa(bn) + " @@\n")
		for _, o := range hunk {
			out.WriteString(string(o.kind) + o.text + "\n")
		}
		start = end
	}
	return out.String()
}

type op struct {
	kind rune
	text string
	a, b int
}

func trailing(ops []op) int {
	n := 0
	for i := len(ops) - 1; i >= 0 && ops[i].kind == ' '; i-- {
		n++
	}
	return n
}

func lines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

func script(a, b []string) []op {
	// invariant: the longest common subsequence of lines, walked into keeps,
	// removals and additions.
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out []op
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && a[i] == b[j]:
			out = append(out, op{' ', a[i], i, j})
			i, j = i+1, j+1
		case i < n && (j == m || lcs[i+1][j] >= lcs[i][j+1]):
			out = append(out, op{'-', a[i], i, j})
			i++
		default:
			out = append(out, op{'+', b[j], i, j})
			j++
		}
	}
	return out
}

func ReversedDiff(diff string) string {
	// value: a unified diff drawn the other way, from its second text to its
	// first: the two names swapped, each hunk's ranges swapped, and in each
	// run of changed lines the first text's lines, now "+", after the
	// second's, now "-".
	var out strings.Builder
	var minus, plus []string
	flush := func() {
		for _, l := range plus {
			out.WriteString("-" + l + "\n")
		}
		for _, l := range minus {
			out.WriteString("+" + l + "\n")
		}
		minus, plus = nil, nil
	}
	rows := strings.Split(strings.TrimSuffix(diff, "\n"), "\n")
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		switch {
		case diff == "":
		case strings.HasPrefix(row, "--- ") && i+1 < len(rows) && strings.HasPrefix(rows[i+1], "+++ "):
			flush()
			out.WriteString("--- " + strings.TrimPrefix(rows[i+1], "+++ ") + "\n+++ " + strings.TrimPrefix(row, "--- ") + "\n")
			i++
		case strings.HasPrefix(row, "@@ "):
			flush()
			if f := strings.Fields(row); len(f) == 4 {
				row = "@@ -" + strings.TrimPrefix(f[2], "+") + " +" + strings.TrimPrefix(f[1], "-") + " @@"
			}
			out.WriteString(row + "\n")
		case strings.HasPrefix(row, "-"):
			minus = append(minus, row[1:])
		case strings.HasPrefix(row, "+"):
			plus = append(plus, row[1:])
		default:
			flush()
			out.WriteString(row + "\n")
		}
	}
	flush()
	return out.String()
}
