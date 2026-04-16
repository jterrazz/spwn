package runtime

import "fmt"

var registry = map[string]Runtime{}

// Register adds a runtime to the global registry.
func Register(r Runtime) {
	registry[r.Name()] = r
}

// Get returns a runtime by name.
func Get(name string) (Runtime, error) {
	r, ok := registry[name]
	if !ok {
		names := make([]string, 0, len(registry))
		for k := range registry {
			names = append(names, k)
		}
		return nil, fmt.Errorf("unknown runtime %q, available: %v", name, names)
	}
	return r, nil
}

// All returns all registered runtimes.
func All() map[string]Runtime {
	return registry
}
