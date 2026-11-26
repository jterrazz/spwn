package automation

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// linkedProject makes <dir>/site and <dir>/outside, the outside holding an
// rc file and a state.json with known bytes, and links name in the site to
// target, a path under the outside.
func linkedProject(t *testing.T, name, target string) (site, outside string) {
	t.Helper()
	dir := t.TempDir()
	site, outside = filepath.Join(dir, "site"), filepath.Join(dir, "outside")
	for _, d := range []string{filepath.Join(site, ".spwn"), outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"rc", "state.json"} {
		if err := os.WriteFile(filepath.Join(outside, f), []byte("untouched\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(site, name)
	if err := os.RemoveAll(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, target), link); err != nil {
		t.Fatal(err)
	}
	return site, outside
}

func outsideIsUntouched(t *testing.T, outside string) {
	t.Helper()
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("something was created outside: %v", entries)
	}
	for _, f := range []string{"rc", "state.json"} {
		if data, err := os.ReadFile(filepath.Join(outside, f)); err != nil || string(data) != "untouched\n" {
			t.Errorf("%s outside was written: %q %v", f, data, err)
		}
	}
}

func TestAReceiptThroughALinkOutWritesNothingOutside(t *testing.T) {
	for name, target := range map[string]string{
		".spwn/runs.jsonl": "rc",
		".spwn":            ".",
	} {
		t.Run(name, func(t *testing.T) {
			site, outside := linkedProject(t, name, target)
			w := &FileReceiptWriter{Root: site, Path: filepath.Join(site, ".spwn", "runs.jsonl"), RotateSize: 1, RotateKeep: 2}
			for range 3 {
				if err := w.Write(Receipt{World: "neo", Automation: "$(touch pwned)", Fired: time.Now()}); err == nil {
					t.Fatal("a receipt was written through a link out")
				}
			}
			outsideIsUntouched(t, outside)
		})
	}
}

func TestAStateFileThroughALinkOutIsNeitherReadNorWritten(t *testing.T) {
	for name, target := range map[string]string{
		".spwn/automations": ".",
		".spwn":             ".",
	} {
		t.Run(name, func(t *testing.T) {
			site, outside := linkedProject(t, name, target)
			s := NewFileStateStore(site, filepath.Join(site, ".spwn", "automations", "state.json"))
			if err := s.RecordFire("neo", "daily", time.Now()); err == nil {
				t.Fatal("a state file was written through a link out")
			}
			if s.IsTainted() == nil {
				t.Error("a state file read through a link out is not reported")
			}
			outsideIsUntouched(t, outside)
		})
	}
}
