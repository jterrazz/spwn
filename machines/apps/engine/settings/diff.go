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
	// removals and additions. The lines both texts open with are kept without
	// a table, and the table of the rest is held as every k-th row of it, the
	// rows between recomputed as the walk reaches them, so its memory is the
	// square root of the full table's and its walk is the full table's.
	head := 0
	for head < len(a) && head < len(b) && a[head] == b[head] {
		head++
	}
	out := make([]op, 0, len(a)+len(b)-head)
	for i := range head {
		out = append(out, op{' ', a[i], i, i})
	}
	return walk(out, a[head:], b[head:], head)
}

func fillRow(cur, next []int32, line string, b []string) {
	cur[len(b)] = 0
	for j := len(b) - 1; j >= 0; j-- {
		if line == b[j] {
			cur[j] = next[j+1] + 1
		} else {
			cur[j] = max(next[j], cur[j+1])
		}
	}
}

func walk(out []op, a, b []string, at int) []op {
	n, m := len(a), len(b)
	k := 1
	for k*k < n {
		k++
	}
	zero := make([]int32, m+1)
	checks := make(map[int][]int32)
	cur, spare := make([]int32, m+1), make([]int32, m+1)
	for i := n - 1; i >= 0; i-- {
		fillRow(spare, cur, a[i], b)
		cur, spare = spare, cur
		if i%k == 0 {
			checks[i/k] = append([]int32(nil), cur...)
		}
	}
	checkAt := func(i int) []int32 {
		if i >= n {
			return zero
		}
		return checks[i/k]
	}
	block := make([][]int32, k+1)
	for r := range block {
		block[r] = make([]int32, m+1)
	}
	lo, hi := 0, 0
	load := func(i int) {
		lo = i / k * k
		hi = min(lo+k, n)
		copy(block[hi-lo], checkAt(hi))
		for r := hi - 1; r >= lo; r-- {
			fillRow(block[r-lo], block[r+1-lo], a[r], b)
		}
	}
	i, j := 0, 0
	for i < n || j < m {
		if i < n && i >= hi {
			load(i)
		}
		switch {
		case i < n && j < m && a[i] == b[j]:
			out = append(out, op{' ', a[i], at + i, at + j})
			i, j = i+1, j+1
		case i < n && (j == m || block[i+1-lo][j] >= block[i-lo][j+1]):
			out = append(out, op{'-', a[i], at + i, at + j})
			i++
		default:
			out = append(out, op{'+', b[j], at + i, at + j})
			j++
		}
	}
	return out
}

func ReversedDiff(diff string) string {
	// value: a unified diff drawn the other way, from its second text to its
	// first: the two names swapped, each hunk's ranges swapped, and in each
	// run of changed lines the first text's lines, now "+", after the
	// second's, now "-". A hunk's lines are counted by its ranges, so a line
	// whose text opens like a header or a hunk is read as the line it is.
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
	left, right := 0, 0
	for i := 0; i < len(rows) && diff != ""; i++ {
		row := rows[i]
		switch {
		case left+right > 0 && strings.HasPrefix(row, "-"):
			minus, left = append(minus, row[1:]), left-1
		case left+right > 0 && strings.HasPrefix(row, "+"):
			plus, right = append(plus, row[1:]), right-1
		case left+right > 0:
			flush()
			out.WriteString(row + "\n")
			left, right = left-1, right-1
		case strings.HasPrefix(row, "--- ") && i+1 < len(rows) && strings.HasPrefix(rows[i+1], "+++ "):
			flush()
			out.WriteString("--- " + strings.TrimPrefix(rows[i+1], "+++ ") + "\n+++ " + strings.TrimPrefix(row, "--- ") + "\n")
			i++
		case strings.HasPrefix(row, "@@ "):
			flush()
			if f := strings.Fields(row); len(f) == 4 {
				left, right = hunkLines(f[1]), hunkLines(f[2])
				row = "@@ -" + strings.TrimPrefix(f[2], "+") + " +" + strings.TrimPrefix(f[1], "-") + " @@"
			}
			out.WriteString(row + "\n")
		default:
			flush()
			out.WriteString(row + "\n")
		}
	}
	flush()
	return out.String()
}

// value: the number of lines a hunk's range counts, "-3,4" or "+5" (one).
func hunkLines(r string) int {
	_, n, ok := strings.Cut(r[1:], ",")
	if !ok {
		return 1
	}
	count, err := strconv.Atoi(n)
	if err != nil {
		return 0
	}
	return count
}
