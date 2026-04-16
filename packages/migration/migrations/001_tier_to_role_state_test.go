package migrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func testdataPath(name string) string {
	return filepath.Join("..", "testdata", name)
}

func TestTierToRoleState(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, testdataPath("001_before.json"), filepath.Join(dir, "state.json"))

	if err := TierToRoleState.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	want, _ := os.ReadFile(testdataPath("001_after.json"))
	if string(got) != string(want) {
		t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestTierToRoleState_Idempotent(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, testdataPath("001_before.json"), filepath.Join(dir, "state.json"))

	ctx := context.Background()
	if err := TierToRoleState.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}
	if err := TierToRoleState.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	want, _ := os.ReadFile(testdataPath("001_after.json"))
	if string(got) != string(want) {
		t.Errorf("second run changed output.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestTierToRoleState_MissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := TierToRoleState.Apply(context.Background(), dir); err != nil {
		t.Fatalf("expected no error on missing file, got: %v", err)
	}
}
