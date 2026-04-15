package compile

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestTreeAddAndGet(t *testing.T) {
	tr := New()
	tr.AddString("a.md", "hello")
	tr.Add("dir/b.md", []byte("world"))

	if !tr.Has("a.md") {
		t.Fatal("expected a.md")
	}
	got, ok := tr.Get("dir/b.md")
	if !ok || string(got) != "world" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	if tr.Has("missing.md") {
		t.Fatal("missing.md should not exist")
	}
}

func TestTreePathNormalisation(t *testing.T) {
	tr := New()
	tr.AddString("/leading/slash.md", "x")
	tr.AddString("./dot/slash.md", "y")
	tr.AddString("back\\slash.md", "z")

	for _, p := range []string{"leading/slash.md", "dot/slash.md", "back/slash.md"} {
		if !tr.Has(p) {
			t.Errorf("expected normalised path %s", p)
		}
	}
}

func TestTreePathsSorted(t *testing.T) {
	tr := New()
	tr.AddString("z.md", "")
	tr.AddString("a.md", "")
	tr.AddString("m/n.md", "")

	want := []string{"a.md", "m/n.md", "z.md"}
	if got := tr.Paths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestTreeOverwrite(t *testing.T) {
	tr := New()
	tr.AddString("a.md", "one")
	tr.AddString("a.md", "two")
	got, _ := tr.Get("a.md")
	if string(got) != "two" {
		t.Fatalf("got %q want two", got)
	}
}

func TestTreeWalkOrder(t *testing.T) {
	tr := New()
	tr.AddString("b.md", "2")
	tr.AddString("a.md", "1")

	var seen []string
	tr.Walk(func(path string, _ []byte) {
		seen = append(seen, path)
	})
	if !reflect.DeepEqual(seen, []string{"a.md", "b.md"}) {
		t.Fatalf("walk order wrong: %v", seen)
	}
}

func TestTreeWriteToMaterialises(t *testing.T) {
	dir := t.TempDir()
	tr := New()
	tr.AddString("top.md", "top")
	tr.AddString("sub/inner.md", "inner")
	tr.AddString("sub/deep/more.md", "deep")

	if err := tr.WriteTo(dir); err != nil {
		t.Fatal(err)
	}

	check := func(rel, want string) {
		t.Helper()
		got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if string(got) != want {
			t.Fatalf("%s: got %q want %q", rel, got, want)
		}
	}
	check("top.md", "top")
	check("sub/inner.md", "inner")
	check("sub/deep/more.md", "deep")
}

func TestTreeTarRoundTrip(t *testing.T) {
	tr := New()
	tr.AddString("top.md", "top")
	tr.AddString("sub/inner.md", "inner")
	tr.AddString("sub/deep/more.md", "deep")

	var buf bytes.Buffer
	if err := tr.Tar(&buf); err != nil {
		t.Fatalf("Tar: %v", err)
	}

	reloaded := New()
	tr2 := tar.NewReader(bytes.NewReader(buf.Bytes()))
	for {
		hdr, err := tr2.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		data, err := io.ReadAll(tr2)
		if err != nil {
			t.Fatalf("tar payload: %v", err)
		}
		reloaded.Add(hdr.Name, data)
	}

	if !reflect.DeepEqual(tr.Paths(), reloaded.Paths()) {
		t.Fatalf("paths differ: %v vs %v", tr.Paths(), reloaded.Paths())
	}
	for _, p := range tr.Paths() {
		a, _ := tr.Get(p)
		b, _ := reloaded.Get(p)
		if !bytes.Equal(a, b) {
			t.Fatalf("%s content differs", p)
		}
	}
}

func TestTreeTarDeterministic(t *testing.T) {
	mk := func() *Tree {
		tr := New()
		tr.AddString("a.md", "alpha")
		tr.AddString("nested/b.md", "beta")
		tr.AddString("z/y/x.md", "gamma")
		return tr
	}
	var buf1, buf2 bytes.Buffer
	if err := mk().Tar(&buf1); err != nil {
		t.Fatalf("Tar 1: %v", err)
	}
	if err := mk().Tar(&buf2); err != nil {
		t.Fatalf("Tar 2: %v", err)
	}
	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Fatal("tar output is not deterministic across identical inputs")
	}
}

func TestTreeWriteToRoundTrip(t *testing.T) {
	dir := t.TempDir()
	original := New()
	original.AddString("a.md", "A")
	original.AddString("b/c.md", "C")

	if err := original.WriteTo(dir); err != nil {
		t.Fatal(err)
	}

	// Reload and compare.
	reloaded := New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		reloaded.Add(filepath.ToSlash(rel), data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(original.Paths(), reloaded.Paths()) {
		t.Fatalf("paths differ: %v vs %v", original.Paths(), reloaded.Paths())
	}
	for _, p := range original.Paths() {
		a, _ := original.Get(p)
		b, _ := reloaded.Get(p)
		if !bytes.Equal(a, b) {
			t.Fatalf("%s content differs", p)
		}
	}
}
