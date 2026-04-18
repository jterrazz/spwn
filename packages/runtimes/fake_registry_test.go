package runtimes_test

import "spwn.sh/packages/dependency/tool"

// fakeToolRegistry is a minimal tool.Registry for RegisterDefaults
// tests. It records which tool names were registered so the test
// can assert coverage without pulling in the real resolver.Registry
// (which would create an import cycle via the dep-resolver's own
// tests).
type fakeToolRegistry struct {
	seen map[string]bool
}

func (r *fakeToolRegistry) Register(t tool.Tool) error {
	r.seen[t.Name()] = true
	return nil
}
