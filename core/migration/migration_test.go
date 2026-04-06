package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRunnerAppliesAllMigrations(t *testing.T) {
	dir := t.TempDir()

	// Seed a .json file so backup has something to copy
	os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{}`), 0644)

	var applied []int
	migrations := []Migration{
		{Number: 1, Description: "first", Apply: func(_ context.Context, _ string) error {
			applied = append(applied, 1)
			return nil
		}},
		{Number: 2, Description: "second", Apply: func(_ context.Context, _ string) error {
			applied = append(applied, 2)
			return nil
		}},
		{Number: 3, Description: "third", Apply: func(_ context.Context, _ string) error {
			applied = append(applied, 3)
			return nil
		}},
	}

	runner := NewRunner(dir, migrations)
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(applied) != 3 {
		t.Fatalf("expected 3 migrations applied, got %d", len(applied))
	}
	for i, n := range applied {
		if n != i+1 {
			t.Errorf("applied[%d] = %d, want %d", i, n, i+1)
		}
	}

	// Verify version.json
	v, err := LoadVersion(dir)
	if err != nil {
		t.Fatalf("LoadVersion: %v", err)
	}
	if v.Version != 3 {
		t.Errorf("version = %d, want 3", v.Version)
	}
	if len(v.Applied) != 3 {
		t.Errorf("applied count = %d, want 3", len(v.Applied))
	}
}

func TestRunnerSkipsAlreadyApplied(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{}`), 0644)

	callCount := 0
	migrations := []Migration{
		{Number: 1, Description: "first", Apply: func(_ context.Context, _ string) error {
			callCount++
			return nil
		}},
		{Number: 2, Description: "second", Apply: func(_ context.Context, _ string) error {
			callCount++
			return nil
		}},
	}

	runner := NewRunner(dir, migrations)
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls, got %d", callCount)
	}

	// Run again - should skip all
	callCount = 0
	runner2 := NewRunner(dir, migrations)
	if err := runner2.Run(context.Background()); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected 0 calls on re-run, got %d", callCount)
	}
}

func TestRunnerPartialFailure(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{}`), 0644)

	migrations := []Migration{
		{Number: 1, Description: "ok", Apply: func(_ context.Context, _ string) error {
			return nil
		}},
		{Number: 2, Description: "fail", Apply: func(_ context.Context, _ string) error {
			return fmt.Errorf("boom")
		}},
		{Number: 3, Description: "never", Apply: func(_ context.Context, _ string) error {
			t.Error("migration 3 should not run")
			return nil
		}},
	}

	runner := NewRunner(dir, migrations)
	err := runner.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Version should be 1 (the successful one)
	v, err := LoadVersion(dir)
	if err != nil {
		t.Fatalf("LoadVersion: %v", err)
	}
	if v.Version != 1 {
		t.Errorf("version = %d, want 1", v.Version)
	}
	if len(v.Applied) != 1 {
		t.Errorf("applied count = %d, want 1", len(v.Applied))
	}
}

func TestRunnerNoPendingIsNoop(t *testing.T) {
	dir := t.TempDir()

	// Pre-set version to 5
	v := &SchemaVersion{Version: 5}
	if err := SaveVersion(dir, v); err != nil {
		t.Fatalf("SaveVersion: %v", err)
	}

	migrations := []Migration{
		{Number: 1, Description: "old", Apply: func(_ context.Context, _ string) error {
			t.Error("should not run")
			return nil
		}},
	}

	runner := NewRunner(dir, migrations)
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
