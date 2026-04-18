package user

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestSimplifyArchitectDir_Fixture is the happy path via the shared
// fixture harness. The before/after tree lives at
// testdata/user/010_simplify_architect_dir/ — the transformation is
// visible on disk instead of hidden inside an inline setup.
func TestSimplifyArchitectDir_Fixture(t *testing.T) {
	runFixture(t, SimplifyArchitectDir, "010_simplify_architect_dir")
}

// TestSimplifyArchitectDir_NoArchitectDir covers the no-op path: a
// baseDir without architect/ should pass through without error.
func TestSimplifyArchitectDir_NoArchitectDir(t *testing.T) {
	dir := t.TempDir()
	if err := SimplifyArchitectDir.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
}

// TestSimplifyArchitectDir_OnlyTodo: todo.md present, no stack or
// directives. stack.md should stay unwritten; todo.md goes away.
func TestSimplifyArchitectDir_OnlyTodo(t *testing.T) {
	dir := t.TempDir()
	archDir := filepath.Join(dir, "architect")
	os.MkdirAll(archDir, 0755)
	os.WriteFile(filepath.Join(archDir, "todo.md"), []byte("- item"), 0644)

	if err := SimplifyArchitectDir.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(archDir, "todo.md")); !os.IsNotExist(err) {
		t.Error("todo.md should have been removed")
	}
}

// TestSimplifyArchitectDir_OnlyDirectivesNoStack: no stack.md
// present, so the directive content becomes the whole stack.md.
func TestSimplifyArchitectDir_OnlyDirectivesNoStack(t *testing.T) {
	dir := t.TempDir()
	archDir := filepath.Join(dir, "architect")
	os.MkdirAll(archDir, 0755)
	os.WriteFile(filepath.Join(archDir, "directives.md"), []byte("Be bold."), 0644)

	if err := SimplifyArchitectDir.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(archDir, "stack.md"))
	if err != nil {
		t.Fatal("stack.md should have been created:", err)
	}
	if string(data) != "Be bold." {
		t.Errorf("unexpected: %s", string(data))
	}

	if _, err := os.Stat(filepath.Join(archDir, "directives.md")); !os.IsNotExist(err) {
		t.Error("directives.md should have been removed")
	}
}

// TestSimplifyArchitectDir_Idempotent: running the migration twice
// is safe. After the first run directives.md is gone so the second
// run is a full no-op.
func TestSimplifyArchitectDir_Idempotent(t *testing.T) {
	dir := t.TempDir()
	archDir := filepath.Join(dir, "architect")
	os.MkdirAll(archDir, 0755)
	os.WriteFile(filepath.Join(archDir, "stack.md"), []byte("## Focus"), 0644)
	os.WriteFile(filepath.Join(archDir, "directives.md"), []byte("directive"), 0644)

	SimplifyArchitectDir.Apply(context.Background(), dir)

	if err := SimplifyArchitectDir.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(archDir, "stack.md"))
	if string(data) != "## Focus\n\n---\n\n# Archived Directives\n\ndirective" {
		t.Errorf("unexpected after idempotent run: %s", string(data))
	}
}
