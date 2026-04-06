package migrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSimplifyArchitectDir(t *testing.T) {
	dir := t.TempDir()
	archDir := filepath.Join(dir, "architect")
	os.MkdirAll(archDir, 0755)

	os.WriteFile(filepath.Join(archDir, "stack.md"), []byte("## Focus\n- [ ] Ship v2"), 0644)
	os.WriteFile(filepath.Join(archDir, "directives.md"), []byte("Always test first."), 0644)
	os.WriteFile(filepath.Join(archDir, "todo.md"), []byte("- old todo item"), 0644)

	if err := SimplifyArchitectDir.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	// stack.md should contain merged content
	data, err := os.ReadFile(filepath.Join(archDir, "stack.md"))
	if err != nil {
		t.Fatal("stack.md should exist:", err)
	}
	content := string(data)
	if content != "## Focus\n- [ ] Ship v2\n\n---\n\n# Archived Directives\n\nAlways test first." {
		t.Errorf("unexpected stack.md content:\n%s", content)
	}

	// directives.md should be gone
	if _, err := os.Stat(filepath.Join(archDir, "directives.md")); !os.IsNotExist(err) {
		t.Error("directives.md should have been removed")
	}

	// todo.md should be gone
	if _, err := os.Stat(filepath.Join(archDir, "todo.md")); !os.IsNotExist(err) {
		t.Error("todo.md should have been removed")
	}
}

func TestSimplifyArchitectDir_NoArchitectDir(t *testing.T) {
	dir := t.TempDir()
	if err := SimplifyArchitectDir.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
}

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

func TestSimplifyArchitectDir_Idempotent(t *testing.T) {
	dir := t.TempDir()
	archDir := filepath.Join(dir, "architect")
	os.MkdirAll(archDir, 0755)
	os.WriteFile(filepath.Join(archDir, "stack.md"), []byte("## Focus"), 0644)
	os.WriteFile(filepath.Join(archDir, "directives.md"), []byte("directive"), 0644)

	SimplifyArchitectDir.Apply(context.Background(), dir)

	// Second run — files already gone, should be a no-op
	if err := SimplifyArchitectDir.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(archDir, "stack.md"))
	if string(data) != "## Focus\n\n---\n\n# Archived Directives\n\ndirective" {
		t.Errorf("unexpected after idempotent run: %s", string(data))
	}
}
