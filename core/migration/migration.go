package migration

import (
	"context"
	"fmt"
	"time"
)

// Migration represents a single versioned schema change.
type Migration struct {
	Number      int
	Description string
	Apply       func(ctx context.Context, baseDir string) error
}

// Runner executes pending migrations in order.
type Runner struct {
	migrations []Migration
	baseDir    string
}

func NewRunner(baseDir string, migrations []Migration) *Runner {
	return &Runner{migrations: migrations, baseDir: baseDir}
}

// Run loads current version, creates backup, runs pending migrations, updates version.json after each.
func (r *Runner) Run(ctx context.Context) error {
	current, err := LoadVersion(r.baseDir)
	if err != nil {
		return fmt.Errorf("load schema version: %w", err)
	}

	var pending []Migration
	for _, m := range r.migrations {
		if m.Number > current.Version {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		return nil
	}

	if err := BackupBaseDir(r.baseDir); err != nil {
		return fmt.Errorf("pre-migration backup: %w", err)
	}

	for _, m := range pending {
		if err := m.Apply(ctx, r.baseDir); err != nil {
			return fmt.Errorf("migration %03d (%s): %w", m.Number, m.Description, err)
		}
		current.Version = m.Number
		current.UpdatedAt = time.Now().UTC()
		current.Applied = append(current.Applied, AppliedMigration{
			Number:      m.Number,
			Description: m.Description,
			AppliedAt:   time.Now().UTC(),
		})
		if err := SaveVersion(r.baseDir, current); err != nil {
			return fmt.Errorf("save version after migration %03d: %w", m.Number, err)
		}
	}
	return nil
}
