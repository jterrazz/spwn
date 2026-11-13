package files_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func TestTheOwnersFoldersKeepTheContract(t *testing.T) {
	contracttest.Source(t, func(t *testing.T, held map[string]string) contract.Source {
		root := t.TempDir()
		for p, body := range held {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, p), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return files.New(local.New(nil, nil, nil, nil).FS, contract.FileLinks, filepath.Join(root, "catalog"), filepath.Join(root, "machinefiles"), machinefile.User, "~/.config/station")
	})
}

func TestAnEmbeddedTreeKeepsTheContract(t *testing.T) {
	contracttest.Source(t, func(_ *testing.T, held map[string]string) contract.Source {
		fsys := fstest.MapFS{}
		for p, body := range held {
			fsys[p] = &fstest.MapFile{Data: []byte(body)}
		}
		return files.New(files.Embedded{FS: fsys}, contract.NoLinks, "catalog", "machinefiles", machinefile.Builtin, "built-in")
	})
}

// value: a tree whose links point where targets says: the mode Stat
// reads through each link.
type linked struct {
	fstest.MapFS
	targets map[string]fs.FileMode
}

func (l linked) ReadDir(name string) ([]fs.DirEntry, error) { return fs.ReadDir(l.MapFS, name) }
func (linked) ReadFile(string) ([]byte, error) {
	return []byte("machinefile: station.machinefile/v1\n"), nil
}
func (l linked) Stat(name string) (fs.FileInfo, error) {
	if mode, ok := l.targets[name]; ok {
		return fstest.MapFS{name: &fstest.MapFile{Mode: mode}}.Stat(name)
	}
	return l.MapFS.Stat(name)
}

func TestTheOwnersFoldersFollowALinkToAFileAlone(t *testing.T) {
	cases := []struct {
		about, at string
		mode, to  fs.FileMode
		refused   bool
	}{
		{"a link to a file", "machinefiles/laptop.yaml", fs.ModeSymlink, 0, false},
		{"a link to a folder", "machinefiles/laptop.yaml", fs.ModeSymlink, fs.ModeDir, true},
		{"a link to a folder named as an item", "catalog/apps/jq.yaml", fs.ModeSymlink, fs.ModeDir, true},
		{"a pipe", "catalog/apps/jq.yaml", fs.ModeNamedPipe, 0, true},
		{"a device", "machinefiles/laptop.yaml", fs.ModeDevice, 0, true},
	}
	for _, c := range cases {
		dir := linked{MapFS: fstest.MapFS{c.at: &fstest.MapFile{Mode: c.mode}}, targets: map[string]fs.FileMode{c.at: c.to}}
		tree, err := files.New(dir, contract.FileLinks, "catalog", "machinefiles", machinefile.User, "~/.config/station").Read(t.Context())
		switch {
		case c.refused && (err == nil || !strings.Contains(err.Error(), c.at)):
			t.Errorf("%s read with %v, want it refused with its path", c.about, err)
		case !c.refused && (err != nil || len(tree.Files) != 1):
			t.Errorf("%s read %v, %v; want the file it points at", c.about, tree.Files, err)
		}
	}
}

func TestASharedSourceFollowsNoLink(t *testing.T) {
	for _, at := range []string{"machinefiles/laptop.yaml", "catalog/apps/jq.yaml"} {
		dir := linked{MapFS: fstest.MapFS{at: &fstest.MapFile{Mode: fs.ModeSymlink}}, targets: map[string]fs.FileMode{at: 0}}
		_, err := files.New(dir, contract.NoLinks, "catalog", "machinefiles", machinefile.Shared, "studio@1a2b3c4").Read(t.Context())
		if err == nil || !strings.Contains(err.Error(), at+": is a link") {
			t.Errorf("a link at %s read with %v, want it refused", at, err)
		}
	}
}

func TestAnEntryThatIsNoYamlIsSkippedWhateverItsType(t *testing.T) {
	for _, at := range []string{"machinefiles/notes", "catalog/apps/readme", "catalog/apps/jq.md"} {
		for _, mode := range []fs.FileMode{fs.ModeSymlink, fs.ModeNamedPipe, fs.ModeDevice, 0} {
			dir := linked{MapFS: fstest.MapFS{at: &fstest.MapFile{Mode: mode}}, targets: map[string]fs.FileMode{at: fs.ModeDir}}
			tree, err := files.New(dir, contract.NoLinks, "catalog", "machinefiles", machinefile.Shared, "studio@1a2b3c4").Read(t.Context())
			if err != nil || len(tree.Files) != 0 {
				t.Errorf("%s of mode %v read %v, %v; want it skipped", at, mode, tree.Files, err)
			}
		}
	}
}
