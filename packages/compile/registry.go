package compile

import (
	"fmt"

	"spwn.sh/packages/dependency"
)

// Registry holds all registered tools and resolves dependency graphs.
// Keys are the canonical scheme-form ref (`spwn:unix`, `github:owner/repo`).
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry. Returns error on duplicate name.
func (r *Registry) Register(t Tool) error {
	name := t.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateTool, name)
	}
	r.tools[name] = t
	return nil
}

// Get returns a tool by name, or nil if not found.
func (r *Registry) Get(name string) Tool {
	if t, ok := r.tools[name]; ok {
		return t
	}
	return r.tools[dependency.Canonical(name)]
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// Resolve takes a list of requested tool names, expands transitive dependencies,
// deduplicates, and returns a topologically sorted build order.
func (r *Registry) Resolve(requested []string) ([]Tool, error) {
	// Canonicalise every input ref up-front so the rest of the
	// algorithm deals in one consistent key space.
	canon := make([]string, len(requested))
	for i, name := range requested {
		if _, ok := r.tools[name]; ok {
			canon[i] = name
		} else {
			canon[i] = dependency.Canonical(name)
		}
	}
	requested = canon

	// Validate all requested tools exist
	for _, name := range requested {
		if r.tools[name] == nil {
			return nil, fmt.Errorf("%w: %s", ErrToolNotFound, name)
		}
	}

	// Collect all tools transitively
	visited := make(map[string]bool)
	var allNames []string
	var collect func(name string) error
	collect = func(name string) error {
		if visited[name] {
			return nil
		}
		t := r.tools[name]
		if t == nil {
			return &MissingDependencyError{Tool: name, Dependency: name}
		}
		visited[name] = true
		for _, dep := range t.Dependencies() {
			if r.tools[dep] == nil {
				return &MissingDependencyError{Tool: name, Dependency: dep}
			}
			if err := collect(dep); err != nil {
				return err
			}
		}
		allNames = append(allNames, name)
		return nil
	}

	for _, name := range requested {
		if err := collect(name); err != nil {
			return nil, err
		}
	}

	// Topological sort (Kahn's algorithm)
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // dep → tools that depend on it
	nameSet := make(map[string]bool)

	for _, name := range allNames {
		nameSet[name] = true
		inDegree[name] = 0
	}
	for _, name := range allNames {
		for _, dep := range r.tools[name].Dependencies() {
			if nameSet[dep] {
				inDegree[name]++
				dependents[dep] = append(dependents[dep], name)
			}
		}
	}

	var queue []string
	for _, name := range allNames {
		if inDegree[name] == 0 {
			queue = append(queue, name)
		}
	}

	var sorted []Tool
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		sorted = append(sorted, r.tools[name])
		for _, dep := range dependents[name] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(sorted) != len(allNames) {
		return nil, ErrDependencyCycle
	}

	return sorted, nil
}
