package wire

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/pages/menubar"
	"github.com/jterrazz/studio-station/src/ports"
)

// invariant: SPEC A5 Hints: every station line a hint sheet gives the
// owner to copy is a leaf station has today, its flags its own: asked for
// its help, station answers with that leaf's usage, never its parent's.
func TestEveryStationLineAHintGivesIsALeafOfStation(t *testing.T) {
	lines := map[string]string{
		menubar.AddMacCommands()[1]: "station machines add",
		menubar.DeclareCommand(ports.Port{Machine: coremachine.Ref{Name: "mac"}, Port: 5000}, "mac", "mac"):                 "station ports declare",
		menubar.DeclareCommand(ports.Port{Machine: coremachine.Ref{Name: "linux-1", On: "mac"}, Port: 4321}, "mac", "mac"):  "station ports declare",
		menubar.DeclareCommand(ports.Port{Machine: coremachine.Ref{Name: "mac-mini"}, Port: 10250}, "mac-mini", "this-mac"): "station ports declare",
	}
	for line, leaf := range lines {
		words, ok := strings.CutPrefix(line, "station ")
		if !ok {
			t.Fatalf("%q is no station line", line)
		}
		usage, code := askedHelp(t, append(strings.Fields(words), "--help"))
		if code != 0 || !strings.Contains(usage, leaf+" <") {
			t.Errorf("%q: station answers %d with\n%s\nnot the usage of %s", line, code, usage, leaf)
		}
	}
}

func askedHelp(t *testing.T, args []string) (string, int) {
	t.Helper()
	dir := t.TempDir()
	open := func(name string) *os.File {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.Close() })
		return f
	}
	in, out, errs := open("in"), open("out"), open("err")
	code := Run(args, []string{"HOME=" + dir, "PATH=/usr/bin:/bin", "NO_COLOR=1"}, in, out, errs)
	said, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(said), code
}
