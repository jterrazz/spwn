package migrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOrgYAMLFieldRename(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, testdataPath("006_before.yaml"), filepath.Join(dir, "org.yaml"))

	if err := OrgYAMLFieldRename.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "org.yaml"))
	want, _ := os.ReadFile(testdataPath("006_after.yaml"))
	if string(got) != string(want) {
		t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestOrgYAMLFieldRename_Idempotent(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, testdataPath("006_before.yaml"), filepath.Join(dir, "org.yaml"))

	ctx := context.Background()
	if err := OrgYAMLFieldRename.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(filepath.Join(dir, "org.yaml"))

	if err := OrgYAMLFieldRename.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(dir, "org.yaml"))

	if string(first) != string(second) {
		t.Error("second run changed output")
	}
}

func TestOrgYAMLFieldRename_MissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := OrgYAMLFieldRename.Apply(context.Background(), dir); err != nil {
		t.Fatalf("expected no error on missing file, got: %v", err)
	}
}

func TestOrgYAMLFieldRename_AlreadyMigrated(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, testdataPath("006_after.yaml"), filepath.Join(dir, "org.yaml"))

	if err := OrgYAMLFieldRename.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "org.yaml"))
	want, _ := os.ReadFile(testdataPath("006_after.yaml"))
	if string(got) != string(want) {
		t.Error("already-migrated file was changed")
	}
}
