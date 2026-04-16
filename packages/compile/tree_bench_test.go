package compile_test

import (
	"fmt"
	"testing"

	"spwn.sh/packages/compile"
)

// BenchmarkTreeAddWalk measures the cost of building + walking a
// tree with N entries. Representative of compile pipelines that
// render every agent + every skill + every world file.
func BenchmarkTreeAddWalk(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				t := compile.New()
				for j := 0; j < n; j++ {
					t.AddString(fmt.Sprintf("agents/a%d/CLAUDE.md", j), "content")
				}
				t.Walk(func(string, []byte) {})
			}
		})
	}
}
