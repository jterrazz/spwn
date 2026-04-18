package user

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"spwn.sh/packages/migration"
)

// runFixture drives a migration against a golden before/after tree
// under packages/migration/user/testdata/<fixtureName>/.
//
// Layout:
//
//	packages/migration/user/testdata/<fixtureName>/
//	  before/   → seeded into a fresh t.TempDir() before Apply runs
//	  after/    → expected state of the tempdir once Apply returns
//
// The harness walks both after/ and the tempdir: any missing file,
// extra file, or content mismatch fails the test with the path. This
// makes migration transformations self-documenting — the before/
// tree is the realistic input, the after/ tree is the realistic
// output, and the test code stays a one-liner.
//
// A missing before/ is treated as "empty input" (covers initializer
// migrations like 013 and 015). A missing after/ directory means
// "every seeded file must be gone" (covers eviction migrations like
// 016).
func runFixture(t *testing.T, m migration.Migration, fixtureName string) {
	t.Helper()

	fixtureRoot := filepath.Join("testdata", fixtureName)
	beforeDir := filepath.Join(fixtureRoot, "before")
	afterDir := filepath.Join(fixtureRoot, "after")

	tempDir := t.TempDir()
	if err := copyTree(beforeDir, tempDir); err != nil {
		t.Fatalf("seed before/: %v", err)
	}

	if err := m.Apply(context.Background(), tempDir); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	wantPaths, err := listFilesRel(afterDir)
	if err != nil {
		t.Fatalf("read after/: %v", err)
	}
	gotPaths, err := listFilesRel(tempDir)
	if err != nil {
		t.Fatalf("read tempdir: %v", err)
	}

	want := map[string]bool{}
	for _, p := range wantPaths {
		want[p] = true
	}
	got := map[string]bool{}
	for _, p := range gotPaths {
		got[p] = true
	}

	// Extra files the migration produced but after/ doesn't declare.
	extra := []string{}
	for p := range got {
		if !want[p] {
			extra = append(extra, p)
		}
	}
	// Files after/ declares but the migration failed to produce.
	missing := []string{}
	for p := range want {
		if !got[p] {
			missing = append(missing, p)
		}
	}
	sort.Strings(extra)
	sort.Strings(missing)
	if len(extra) > 0 {
		t.Errorf("unexpected files in tempdir:\n  %v", extra)
	}
	if len(missing) > 0 {
		t.Errorf("files missing from tempdir:\n  %v", missing)
	}

	// Content diff for every file present on both sides.
	for _, rel := range wantPaths {
		if !got[rel] {
			continue
		}
		wantBytes, err := os.ReadFile(filepath.Join(afterDir, rel))
		if err != nil {
			t.Fatalf("read after/%s: %v", rel, err)
		}
		gotBytes, err := os.ReadFile(filepath.Join(tempDir, rel))
		if err != nil {
			t.Fatalf("read tempdir/%s: %v", rel, err)
		}
		if !bytes.Equal(wantBytes, gotBytes) {
			t.Errorf("content mismatch for %s:\n--- want ---\n%s\n--- got ---\n%s",
				rel, truncate(wantBytes, 400), truncate(gotBytes, 400))
		}
	}
}

// listFilesRel returns every file under root, paths relative to root,
// sorted. A missing root is treated as "empty set" so after/ can be
// absent when the expected end state is "nothing left".
//
// `.gitkeep` markers are filtered: testdata fixtures use them to
// preserve otherwise-empty directories in git, but migrations don't
// create them. Treating .gitkeep as invisible lets fixtures declare
// directory presence without breaking the diff.
func listFilesRel(root string) ([]string, error) {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == ".gitkeep" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// copyTree mirrors src into dst, creating dst if it does not exist.
// A missing src is treated as "nothing to copy" so initializer
// migrations can have an absent before/ tree.
func copyTree(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dst, 0o755)
		}
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…(truncated)"
}
