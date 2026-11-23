package settings

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func referenceScript(a, b []string) []op {
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

func sequences(alphabet string, longest int) [][]string {
	all := [][]string{nil}
	for from, to := 0, 1; longest > 0; longest-- {
		for _, prefix := range all[from:to] {
			for _, c := range alphabet {
				all = append(all, append(append([]string(nil), prefix...), string(c)))
			}
		}
		from, to = to, len(all)
	}
	return all
}

func TestTheScriptIsTheFullTablesOnEveryInput(t *testing.T) {
	all := sequences("abc", 5)
	for _, a := range all {
		for _, b := range all {
			got, want := script(a, b), referenceScript(a, b)
			if len(got) != len(want) {
				t.Fatalf("script(%q, %q) = %v, want %v", a, b, got, want)
			}
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("script(%q, %q) = %v, want %v", a, b, got, want)
				}
			}
		}
	}
}

func TestAOneLineEditOfALongFileCostsNoTable(t *testing.T) {
	a := make([]string, 3000)
	for i := range a {
		a[i] = "export LINE_" + strconv.Itoa(i) + "=1"
	}
	b := append([]string(nil), a...)
	b[1500] = "changed"
	from, to := strings.Join(a, "\n")+"\n", strings.Join(b, "\n")+"\n"
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	got := UnifiedDiff(from, to, "station", "~/x")
	runtime.ReadMemStats(&after)
	if !strings.Contains(got, "-export LINE_1500=1\n+changed\n") {
		t.Fatalf("diff = %s", got)
	}
	if spent := after.TotalAlloc - before.TotalAlloc; spent > 4<<20 {
		t.Fatalf("a one-line edit allocated %d bytes", spent)
	}
}
