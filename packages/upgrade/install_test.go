package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func makeTarGz(t *testing.T, files map[string][]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Size:     int64(len(content)),
			Mode:     0755,
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gz.Close()
	return path
}

func TestExtractBinary(t *testing.T) {
	archive := makeTarGz(t, map[string][]byte{
		"README.md": []byte("hello"),
		"spwn":      []byte("#!/bin/bash\necho spwn"),
	})
	dest := filepath.Join(t.TempDir(), "spwn.new")
	if err := ExtractBinary(archive, "spwn", dest); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("extracted file is empty")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		t.Error("extracted file is not executable")
	}
}

func TestExtractBinary_NestedPath(t *testing.T) {
	// GoReleaser sometimes nests the binary under a folder: "spwn/spwn".
	archive := makeTarGz(t, map[string][]byte{
		"spwn/README.md": []byte("x"),
		"spwn/spwn":      []byte("binary bytes"),
	})
	dest := filepath.Join(t.TempDir(), "out")
	if err := ExtractBinary(archive, "spwn", dest); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "binary bytes" {
		t.Errorf("content = %q", got)
	}
}

func TestExtractBinary_NotFound(t *testing.T) {
	archive := makeTarGz(t, map[string][]byte{"other": []byte("x")})
	dest := filepath.Join(t.TempDir(), "out")
	err := ExtractBinary(archive, "spwn", dest)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found', got %v", err)
	}
}

func TestAtomicReplace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping POSIX atomic rename test on windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "spwn")
	newFile := filepath.Join(dir, "spwn.new")
	if err := os.WriteFile(target, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := AtomicReplace(newFile, target); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "new" {
		t.Errorf("after replace: content = %q", got)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm()&0111 == 0 {
		t.Error("replaced file is not executable")
	}
	// The source should no longer exist after rename.
	if _, err := os.Stat(newFile); !os.IsNotExist(err) {
		t.Error("source file should have been moved")
	}
}

func TestIsWritable(t *testing.T) {
	dir := t.TempDir()
	if !IsWritable(filepath.Join(dir, "spwn")) {
		t.Error("tempdir should be writable")
	}
}

func TestAssetNameFor(t *testing.T) {
	tests := []struct {
		platform string
		want     string
	}{
		{"darwin_arm64", "spwn_darwin_arm64.tar.gz"},
		{"linux_amd64", "spwn_linux_amd64.tar.gz"},
		{"windows_amd64", "spwn_windows_amd64.zip"},
		{"windows_arm64", "spwn_windows_arm64.zip"},
	}
	for _, tt := range tests {
		got := AssetNameFor("spwn", tt.platform)
		if got != tt.want {
			t.Errorf("AssetNameFor(%q) = %q, want %q", tt.platform, got, tt.want)
		}
	}
}

func TestPlatformToken(t *testing.T) {
	got := PlatformToken()
	if !strings.Contains(got, "_") {
		t.Errorf("expected GOOS_GOARCH shape, got %q", got)
	}
}
