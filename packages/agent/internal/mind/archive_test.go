package mind

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/platform"
)

type entry struct {
	hdr  tar.Header
	body string
}

func writeArchive(t *testing.T, entries []entry) string {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		h := e.hdr
		if h.Typeflag == tar.TypeReg {
			h.Size = int64(len(e.body))
		}
		if h.Mode == 0 {
			h.Mode = 0o644
		}
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if e.body != "" {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "mind.tar.gz")
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func importProject(t *testing.T) (agents, outside string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", filepath.Join(dir, "spwn-home"))
	t.Setenv("HOME", filepath.Join(dir, "home"))
	root, outside := filepath.Join(dir, "site"), filepath.Join(dir, "outside")
	agents = filepath.Join(root, "agents")
	for _, d := range []string{agents, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	platform.SetProject(root, platform.Folders{Agents: "agents"})
	t.Cleanup(func() { platform.SetProject("", platform.Folders{}) })
	return agents, outside
}

func reg(name, body string) entry {
	return entry{hdr: tar.Header{Name: name, Typeflag: tar.TypeReg}, body: body}
}

func TestImportRefusesEachHostileArchiveAndWritesNothing(t *testing.T) {
	cases := []struct {
		name    string
		entries []entry
	}{
		{"an absolute path", []entry{reg("/tmp/x", "x")}},
		{"a climb", []entry{reg("../../outside/x", "x")}},
		{"a climb inside", []entry{reg("playbooks/../../x", "x")}},
		{"a symlink out", []entry{{hdr: tar.Header{Name: "SOUL.md", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"}}}},
		{"a hard link", []entry{{hdr: tar.Header{Name: "SOUL.md", Typeflag: tar.TypeLink, Linkname: "../x"}}}},
		{"a device", []entry{{hdr: tar.Header{Name: "dev", Typeflag: tar.TypeChar, Devmajor: 1, Devminor: 3}}}},
		{"a FIFO", []entry{{hdr: tar.Header{Name: "fifo", Typeflag: tar.TypeFifo}}}},
		{"an entry too large", nil},
		{"a good file then a bad one", []entry{reg("SOUL.md", "soul\n"), reg("../x", "x")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			agents, outside := importProject(t)
			var archive string
			if c.entries == nil {
				archive = oversized(t)
			} else {
				archive = writeArchive(t, c.entries)
			}
			err := Import("neo", archive)
			if err == nil || !strings.HasPrefix(err.Error(), ArchiveRefusedCode+": ") {
				t.Fatalf("Import = %v, want %s", err, ArchiveRefusedCode)
			}
			if _, err := os.Lstat(filepath.Join(agents, "neo")); err == nil {
				t.Errorf("a refused archive created the agent's folder")
			}
			if left, _ := os.ReadDir(outside); len(left) != 0 {
				t.Errorf("a refused archive wrote outside: %v", left)
			}
		})
	}
}

func oversized(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{Name: "big", Typeflag: tar.TypeReg, Mode: 0o644, Size: maxImportEntry + 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(make([]byte, maxImportEntry+1)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "big.tar.gz")
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAnArchivePastTheTotalBoundIsRefusedThoughEachEntryIsWithinItsOwn(t *testing.T) {
	archive := writeArchive(t, []entry{reg("a", "12345"), reg("b", "12345"), reg("c", "12345")})
	if _, err := importEntries(archive, importBounds{entry: 5, total: 15}); err != nil {
		t.Fatalf("an archive at the bound was refused: %v", err)
	}
	_, err := importEntries(archive, importBounds{entry: 5, total: 14})
	if err == nil || err.Error() != ArchiveRefusedCode+`: entry "c" takes the archive past 14 bytes` {
		t.Fatalf("importEntries = %v, want the total bound's refusal", err)
	}
}

func TestImportDoesNotWriteThroughAProjectLinkOut(t *testing.T) {
	agents, outside := importProject(t)
	if err := os.Symlink(outside, filepath.Join(agents, "neo")); err != nil {
		t.Fatal(err)
	}
	if err := Import("neo", writeArchive(t, []entry{reg("SOUL.md", "soul\n")})); err == nil {
		t.Fatal("Import wrote through agents/neo linked out")
	}
	if left, _ := os.ReadDir(outside); len(left) != 0 {
		t.Errorf("Import wrote outside: %v", left)
	}
}

func TestImportExtractsALegitimateArchiveByteForByte(t *testing.T) {
	agents, _ := importProject(t)
	files := map[string]string{"SOUL.md": "# soul\n", "playbooks/review.md": "review\x00bytes\n", "journal/2026-11-26.md": ""}
	entries := []entry{
		{hdr: tar.Header{Name: "./", Typeflag: tar.TypeDir, Mode: 0o755}},
		{hdr: tar.Header{Name: "playbooks/", Typeflag: tar.TypeDir, Mode: 0o755}},
		{hdr: tar.Header{Name: "journal/", Typeflag: tar.TypeDir, Mode: 0o755}},
	}
	for _, n := range []string{"SOUL.md", "playbooks/review.md", "journal/2026-11-26.md"} {
		entries = append(entries, reg(n, files[n]))
	}
	if err := Import("neo", writeArchive(t, entries)); err != nil {
		t.Fatalf("Import: %v", err)
	}
	for n, want := range files {
		got, err := os.ReadFile(filepath.Join(agents, "neo", filepath.FromSlash(n)))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", n, got, err, want)
		}
	}
}
