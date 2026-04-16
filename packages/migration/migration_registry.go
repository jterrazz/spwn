package migration

import "fmt"

// Registry collects migrations and enforces ordering.
type Registry struct {
	migrations []Migration
}

// NewRegistry creates an empty migration registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a migration to the registry. Panics if the migration number
// is not strictly greater than the last registered migration.
func (r *Registry) Register(m Migration) {
	if len(r.migrations) > 0 {
		last := r.migrations[len(r.migrations)-1].Number
		if m.Number <= last {
			panic(fmt.Sprintf("migration %d must be > %d", m.Number, last))
		}
	}
	r.migrations = append(r.migrations, m)
}

// All returns a copy of all registered migrations.
func (r *Registry) All() []Migration {
	out := make([]Migration, len(r.migrations))
	copy(out, r.migrations)
	return out
}
