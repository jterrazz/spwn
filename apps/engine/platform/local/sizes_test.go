package local

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
)

func reference(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				if st, ok := info.Sys().(*syscall.Stat_t); ok {
					total += min(info.Size(), st.Blocks*512)
				}
			}
		}
		return nil
	})
	return total
}

func write(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
		t.Fatal(err)
	}
}

func sizes(ctx context.Context, roots []string) []int64 {
	got := make([]int64, len(roots))
	Sizes(ctx, roots, func(i int, bytes int64) { got[i] = bytes })
	return got
}

func TestSizesEqualTheWalkReference(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "a", "deep", "b")
	write(t, filepath.Join(a, "small"), 100)
	write(t, filepath.Join(a, "big"), 3<<20)
	for i := range 40 {
		write(t, filepath.Join(a, "tree", strconv.Itoa(i%7), strconv.Itoa(i)), 5000+i)
	}
	write(t, filepath.Join(b, "inner"), 70000)
	sparse := filepath.Join(a, "disk.img")
	f, err := os.Create(sparse)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(64 << 20); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("x"), 1<<20); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if err := os.Symlink(filepath.Join(a, "big"), filepath.Join(a, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(a, "big"), filepath.Join(a, "tree", "hard")); err != nil {
		t.Fatal(err)
	}
	file, link, missing := filepath.Join(a, "big"), filepath.Join(a, "link"), filepath.Join(dir, "none")
	roots := []string{a, b, file, link, missing}
	got := sizes(context.Background(), roots)
	for i, root := range roots {
		if want := reference(root); got[i] != want {
			t.Errorf("%s: %d bytes, the WalkDir reference says %d", root, got[i], want)
		}
	}
	if got[0] >= 64<<20 {
		t.Errorf("the sparse file counted its virtual size: %d", got[0])
	}
}

func TestACancelledWalkStops(t *testing.T) {
	dir := t.TempDir()
	roots := make([]string, 0, 6)
	for r := range 6 {
		root := filepath.Join(dir, strconv.Itoa(r))
		for i := range 300 {
			write(t, filepath.Join(root, strconv.Itoa(i%30), strconv.Itoa(i)), 10)
		}
		roots = append(roots, root)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var answers []int64
	Sizes(ctx, roots, func(_ int, bytes int64) {
		answers = append(answers, bytes)
		cancel()
	})
	if len(answers) != len(roots) {
		t.Fatalf("%d answers for %d roots", len(answers), len(roots))
	}
	if answers[0] <= 0 {
		t.Errorf("the first root answered %d before the cancel", answers[0])
	}
	for _, bytes := range answers[1:] {
		if bytes != -1 {
			t.Errorf("a root answered %d after the cancel, want -1", bytes)
		}
	}
}

// value: a context cancelled once its first check has passed, so the walk is
// cancelled while one directory's first batch is being read.
type cancelledAfterFirst struct {
	context.Context
	checks int
}

func (c *cancelledAfterFirst) Err() error {
	c.checks++
	if c.checks > 1 {
		return context.Canceled
	}
	return nil
}

func TestACancelledWalkStopsInsideOneDirectory(t *testing.T) {
	dir := t.TempDir()
	const subs = 12000
	for i := range subs {
		if err := os.Mkdir(filepath.Join(dir, "d"+strconv.Itoa(i)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	_, _, dev := lstat(dir)
	found := 0
	readDir(&cancelledAfterFirst{Context: context.Background()}, dir, dev, make([]byte, 256<<10), func(string) { found++ })
	if found == 0 || found >= subs {
		t.Fatalf("a walk cancelled after its first batch listed %d of %d directories", found, subs)
	}
	all := 0
	readDir(context.Background(), dir, dev, make([]byte, 256<<10), func(string) { all++ })
	if all != subs {
		t.Fatalf("an uncancelled read listed %d of %d directories", all, subs)
	}
}
