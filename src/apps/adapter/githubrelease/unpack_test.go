package githubrelease

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io/fs"
	"strings"
	"testing"
)

type file struct {
	name string
	mode int64
	body string
}

func tarred(t *testing.T, files []file) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := gzip.NewWriter(&buf)
	w := tar.NewWriter(z)
	for _, f := range files {
		if err := w.WriteHeader(&tar.Header{Name: f.name, Typeflag: tar.TypeReg, Mode: f.mode, Size: int64(len(f.body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(f.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestOnlyTheCommandsOfAnArchiveAreRead(t *testing.T) {
	junk := strings.Repeat("\x00", 4<<20)
	files := []file{{name: "tool/tool", mode: 0o755, body: "T"}, {name: "tool/a", mode: 0o644, body: junk}, {name: "tool/b", mode: 0o644, body: junk}, {name: "tool/c", mode: 0o644, body: junk}}

	es, _, err := entries("x.tar.gz", tarred(t, files), []string{"tool"}, within)

	if err != nil || len(es) != 1 || string(es[0].data) != "T" {
		t.Errorf("read %d files (%v); want the command alone", len(es), err)
	}
}

func TestAnArchivePastItsBoundIsRefused(t *testing.T) {
	// invariant: the archive is read through at most its bound, all its
	// files together, the ones no command names included.
	body := tarred(t, []file{{name: "junk", mode: 0o644, body: strings.Repeat("\x00", 3<<20)}, {name: "tool", mode: 0o755, body: "T"}})

	if _, err := unpackWithin("x.tar.gz", body, []string{"tool"}, bounds{command: 1 << 20, archive: 2 << 20}); err == nil || !strings.Contains(err.Error(), "an archive past 2 MiB") {
		t.Errorf("unpacked with %v; want the archive refused past its bound", err)
	}
	if _, err := unpackWithin("x.tar.gz", body, []string{"tool"}, bounds{command: 1 << 20, archive: 4 << 20}); err != nil {
		t.Errorf("an archive within its bound refused: %v", err)
	}
	big := tarred(t, []file{{name: "tool", mode: 0o755, body: strings.Repeat("\x00", 2<<20)}})
	if _, err := unpackWithin("x.tar.gz", big, []string{"tool"}, bounds{command: 1 << 20, archive: 4 << 20}); err == nil || !strings.Contains(err.Error(), "a file past 1 MiB") {
		t.Errorf("unpacked with %v; want the command refused past its bound", err)
	}
}

func TestACommandIsARunnableFile(t *testing.T) {
	for name, body := range map[string][]byte{
		"a lone match that is no program": tarred(t, []file{{name: "completions/tool", mode: 0o644, body: "complete -F _tool tool"}}),
		"a zip's lone match no program":   zipped(t, "tool", 0o644),
	} {
		asset := "x.tar.gz"
		if strings.HasPrefix(name, "a zip") {
			asset = "x.zip"
		}
		if out, err := unpack(asset, body, []string{"tool"}); err == nil || !strings.Contains(err.Error(), "no runnable one") {
			t.Errorf("%s: unpacked %q (%v); want it refused", name, out, err)
		}
	}
	out, err := unpack("x.tar.gz", tarred(t, []file{{name: "doc/tool", mode: 0o644, body: "DOC"}, {name: "bin/tool", mode: 0o755, body: "BIN"}}), []string{"tool"})
	if err != nil || string(out["tool"]) != "BIN" {
		t.Errorf("unpacked %q (%v); want the runnable one", out, err)
	}
}

func zipped(t *testing.T, name string, mode fs.FileMode) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: name}
	h.SetMode(mode)
	f, err := w.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("#!/bin/sh\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestACompressedFileIsNoCommand(t *testing.T) {
	var gz bytes.Buffer
	z := gzip.NewWriter(&gz)
	_, _ = z.Write([]byte("\x7fELF"))
	_ = z.Close()
	for _, asset := range []string{"rust-analyzer-aarch64-unknown-linux-gnu.gz", "tool.xz", "tool.zst", "tool.bz2"} {
		if out, err := unpack(asset, gz.Bytes(), []string{"tool"}); err == nil || !strings.Contains(err.Error(), "is a compressed file, no command") {
			t.Errorf("%s unpacked %d commands (%v); want it refused", asset, len(out), err)
		}
	}
}
