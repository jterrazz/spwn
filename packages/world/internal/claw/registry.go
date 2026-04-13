package claw

import "fmt"

var registry = map[string]Claw{}

// Register adds a claw to the global registry.
func Register(c Claw) {
	registry[c.Name()] = c
}

// Get returns a claw by name.
func Get(name string) (Claw, error) {
	c, ok := registry[name]
	if !ok {
		names := make([]string, 0, len(registry))
		for k := range registry {
			names = append(names, k)
		}
		return nil, fmt.Errorf("unknown claw %q, available: %v", name, names)
	}
	return c, nil
}

// All returns all registered claws.
func All() map[string]Claw {
	return registry
}
