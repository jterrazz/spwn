package tool

// Registry is the narrow contract any Tool-store must satisfy to
// receive tools registered by the dependency package. The concrete
// registry (typically *compile.Registry) satisfies it structurally
// — dependency stays free of a compile import so the two packages
// can live at the same architectural level without cycling.
//
// Callers that want to "wire every builtin into my registry" pass
// their registry to dependency.RegisterBuiltins, which iterates
// Tools() and calls Register for each. The same interface powers
// dependency.HydrateLocals during a spawn.
type Registry interface {
	Register(Tool) error
}
