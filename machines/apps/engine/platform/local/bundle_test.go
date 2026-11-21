package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTheEngineInsideAnAppIsFoundThroughALinkAndABareOneIsNot(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(dir, "Applications", "Spwn.app")
	engine := filepath.Join(bundle, "Contents", "MacOS", "Spwn Engine")
	bare := filepath.Join(dir, "bin", "station")
	link := filepath.Join(dir, ".local", "bin", "station")
	for _, file := range []string{engine, bare} {
		if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(engine, link); err != nil {
		t.Fatal(err)
	}
	for executable, want := range map[string]string{engine: bundle, link: bundle, bare: "", filepath.Join(dir, "gone"): ""} {
		if got := BundleOf(executable, "Contents/MacOS/Spwn Engine"); got != want {
			t.Errorf("BundleOf(%s) = %q, want %q", executable, got, want)
		}
	}
}

func TestAJobsStartIsReadFromWhatEngineStartRecorded(t *testing.T) {
	start := filepath.Join(t.TempDir(), "jobs.json")
	if err := os.WriteFile(start, []byte(`{"env":{"PATH":"/opt/homebrew/bin"},"logs":{"com.jterrazz.station.engine":"/s/engine.log"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadJobStart(start)
	if err != nil || got.Env["PATH"] != "/opt/homebrew/bin" || got.Logs["com.jterrazz.station.engine"] != "/s/engine.log" {
		t.Errorf("start = %+v, %v", got, err)
	}
}
