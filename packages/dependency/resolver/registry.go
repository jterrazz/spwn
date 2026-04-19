// Package resolver is the dependency resolver: given a flat list of
// requested tool refs, it transitively expands their dependencies,
// deduplicates, and returns a topologically sorted build order.
//
// The Registry is a plain map keyed by canonical ref. Callers
// hydrate it from adapters (spwn catalog builtins via
// dependency.RegisterBuiltins, project-local tool:<name> refs via
// dependency.HydrateLocals) and then call Resolve to drive an image
// build. CollectSkills is a small aggregation helper on top of the
// resolved slice.
package resolver

import (
	"fmt"

	"spwn.sh/packages/dependency/refs"
	"spwn.sh/packages/dependency/tool"
)

// Registry holds all registered tools and resolves dependency graphs.
// Keys are the canonical scheme-form ref (`spwn:unix`, `github:owner/repo`).
type Registry struct {
	tools map[string]tool.Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]tool.Tool)}
}

// Register adds a tool to the registry. Returns error on duplicate name.
func (r *Registry) Register(t tool.Tool) error {
	name := t.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateTool, name)
	}
	r.tools[name] = t
	return nil
}

// Get returns a tool by name, or nil if not found.
func (r *Registry) Get(name string) tool.Tool {
	if t, ok := r.tools[name]; ok {
		return t
	}
	return r.tools[refs.Canonical(name)]
}

// List returns all registered tools.
func (r *Registry) List() []tool.Tool {
	result := make([]tool.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// Resolve takes a list of requested tool names, expands transitive dependencies,
// deduplicates, and returns a topologically sorted build order.
func (r *Registry) Resolve(requested []string) ([]tool.Tool, error) {

	// Canonicalise every input ref up-front so the rest of the
	// algorithm deals in one consistent key space.
	canon := make([]string, len(requested))
	for i, name := range requested {
		if _, ok := r.tools[name]; ok {
			canon[i] = name
		} else {
			canon[i] = refs.Canonical(name)
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

	var sorted []tool.Tool
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
