package compile

import "spwn.sh/packages/dependency/tool"

import (
	"fmt"
	"testing"
)

// BenchmarkRegistryResolve measures the cost of topologically
// sorting a dependency chain. Baseline for guarding the hot path
// that runs on every image build.
//
// Graph shape: each tool depends on the previous one, so all N
// tools form a chain and the sort must traverse the full list.
func BenchmarkRegistryResolve(b *testing.B) {
	for _, n := range []int{10, 50, 200} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			reg := NewRegistry()
			names := make([]string, n)
			for i := 0; i < n; i++ {
				name := fmt.Sprintf("spwn:tool-%d", i)
				names[i] = name
				tool := &benchTool{baseTool: baseTool{name: name}}
				if i > 0 {
					tool.deps = []string{fmt.Sprintf("spwn:tool-%d", i-1)}
				}
				_ = reg.Register(tool)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = reg.Resolve([]string{names[n-1]})
			}
		})
	}
}

// benchTool extends the test baseTool with an overridable
// Dependencies() so we can build dependency chains for the sort.
type benchTool struct {
	baseTool
	deps []string
}

func (t *benchTool) Dependencies() []string { return t.deps }
func (t *benchTool) Kind() tool.Kind  { return tool.KindTool }
