package tool

// Registry is the narrow contract any Tool-store must satisfy to
// receive tools registered by the dependency package. The concrete
// registry — *resolver.Registry under packages/dependency/resolver
// — satisfies it structurally. Keeping the interface in the tool
// subpackage lets adapters (spwn catalog, project-local) hydrate
// any registry without importing resolver.
//
// Callers that want to "wire every builtin into my registry" pass
// their registry to dependency.RegisterBuiltins, which iterates
// Tools() and calls Register for each. The same interface powers
// dependency.HydrateLocals during a spawn.
type Registry interface {
	Register(Tool) error
}
