package architecture_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/wire"
)

func startEnv(home string) []string {
	return []string{
		"HOME=" + home, "PATH=" + filepath.Join(home, "bin"), "TZ=UTC", "NO_COLOR=1", "STATION_MACHINE_KIND=mac",
		"STATION_SYSTEM_DIRS=", "STATION_SYSTEM_ROOT=" + filepath.Join(home, "root"),
	}
}

func started(tb testing.TB, null *os.File, home string, args ...string) *wire.Program {
	tb.Helper()
	p, code := wire.Build(args, startEnv(home), null, null, null, func(code int) { tb.Fatalf("station exited %d", code) })
	if p == nil {
		tb.Fatalf("station built no tree: exit %d", code)
	}
	return p
}

func devNull(tb testing.TB) *os.File {
	tb.Helper()
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { _ = null.Close() })
	return null
}

// invariant: a start decodes no built-in file, the catalog being compiled
// into the binary: neither --version nor a command that reads the catalog
// pays for it, and an owner's package is the one file a start decodes.
func TestAStartDecodesNoBuiltInFileAndAnOwnersPackageOnce(t *testing.T) {
	null := devNull(t)
	for _, args := range [][]string{{"--version"}, {"apps", "--json"}} {
		before := catalog.Decoded()
		p := started(t, null, t.TempDir(), args...)
		if n := catalog.Decoded() - before; n != 0 {
			t.Errorf("station %v decoded %d files at its start, want none", args, n)
		}
		_ = p.Close()
	}
	home := t.TempDir()
	body, err := fs.ReadFile(catalog.BuiltinFiles(), "packages/command-line/bat.yaml")
	if err != nil {
		t.Fatal(err)
	}
	owned := filepath.Join(home, ".config", "station", "catalog", "command-line", "bat.yaml")
	if err := os.MkdirAll(filepath.Dir(owned), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(owned, append([]byte("# the owner's\n"), body...), 0o600); err != nil {
		t.Fatal(err)
	}
	before := catalog.Decoded()
	p := started(t, null, home, "apps", "--json")
	defer func() { _ = p.Close() }()
	if n := catalog.Decoded() - before; n != 1 {
		t.Errorf("a start over one owner's package decoded %d files, want that one", n)
	}
}

func BenchmarkBuild(b *testing.B) {
	null, home := devNull(b), b.TempDir()
	b.ReportAllocs()
	for b.Loop() {
		_ = started(b, null, home).Close()
	}
}

// value: a start as a run makes it, up to the command it dispatches to:
// the tree built, the arguments resolved, nothing executed.
func BenchmarkStartToDispatch(b *testing.B) {
	null, home := devNull(b), b.TempDir()
	args := []string{"machines", "--json"}
	b.ReportAllocs()
	for b.Loop() {
		p := started(b, null, home, args...)
		if _, _, err := p.Root().Find(args); err != nil {
			b.Fatal(err)
		}
		_ = p.Close()
	}
}
