package git_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/git"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

// value: the shared source over a clone holding files, its repository's
// machines declared at sub, read by the computer named mini; and the
// clone's folder.
func cloned(t *testing.T, held map[string]string, sub string) (src git.Source, dir string) {
	t.Helper()
	cache := t.TempDir()
	dir = filepath.Join(cache, "sources", "studio")
	clone(t, dir, held)
	run := localtest.NewRunner(t)
	run.On("git", in(dir, "status", "--porcelain=v2", "--branch", "--untracked-files=all", "--ignored")...).Reply(status(pinned))
	return git.New(run, local.New(nil, nil, nil, nil).FS, cache, contract.SharedSource{Name: "studio", URL: url, Path: sub}, "mini"), dir
}

func link(t *testing.T, target, at string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := local.New(nil, nil, nil, nil).FS.Symlink(target, at); err != nil {
		t.Fatal(err)
	}
}

type readAs struct {
	path, repo string
	slot       catalog.Slot
}

func readOf(files []machinefile.File) []readAs {
	out := make([]readAs, 0, len(files))
	for _, f := range files {
		out = append(out, readAs{f.Path, f.Repo, f.Slot})
	}
	return out
}

func TestAComputerReadsItsOwnFolderTheKitsAndThePackages(t *testing.T) {
	src, _ := cloned(t, map[string]string{
		"README.md":                                       "the repository's own",
		"machines/mini/host.machine.yaml":                 "its own",
		"machines/mini/development-linux.machine.yaml":    "its VM",
		"machines/macbook/host.machine.yaml":              "another computer's",
		"machines/macbook/development-linux.machine.yaml": "another computer's",
		"kits/essentials.kit.yaml":                        "a kit",
		"kits/README.md":                                  "not read",
		"packages/command-line/zstd.yaml":                 "a package",
		"files/zshrc":                                     "read by the file method alone",
		"catalog/apps/jq.yaml":                            "not read",
		"machinefiles/mini.machine.yaml":                  "not read",
	}, "")
	tree, err := src.Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := []readAs{
		{"catalog/command-line/zstd.yaml", "packages/command-line/zstd.yaml", ""},
		{"machinefiles/development-linux.machine.yaml", "machines/mini/development-linux.machine.yaml", catalog.SlotVM},
		{"machinefiles/essentials.machine.yaml", "kits/essentials.kit.yaml", catalog.SlotKit},
		{"machinefiles/mini.machine.yaml", "machines/mini/host.machine.yaml", catalog.SlotComputer},
	}
	if got := readOf(tree.Files); !slices.Equal(got, want) {
		t.Errorf("read %v, want %v", got, want)
	}
	for _, f := range tree.Files {
		if strings.Contains(string(f.Body), "another") || strings.Contains(string(f.Body), "not read") {
			t.Errorf("%s read %q", f.Repo, f.Body)
		}
	}
}

func TestASubPathIsReadThereAndNamesItsFilesByTheirWholePath(t *testing.T) {
	held := map[string]string{
		"infra/fleet/kits/essentials.kit.yaml": "a kit",
		"kits/elsewhere.kit.yaml":              "not read",
	}
	src, _ := cloned(t, held, "infra/fleet")
	tree, err := src.Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := readOf(tree.Files), []readAs{{"machinefiles/essentials.machine.yaml", "infra/fleet/kits/essentials.kit.yaml", catalog.SlotKit}}; !slices.Equal(got, want) {
		t.Errorf("read %v, want %v", got, want)
	}
	src, _ = cloned(t, held, "infra/none")
	_, err = src.Read(t.Context())
	var gone *contract.UnavailableError
	if !errors.As(err, &gone) || !strings.Contains(err.Error(), `"infra/none" names no folder`) {
		t.Errorf("a path to no folder read with %v, want the source unavailable", err)
	}
}

func TestASubPathThatLeavesTheCloneIsRefusedBeforeAnyRead(t *testing.T) {
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outside, "kits"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "kits", "planted.kit.yaml"), []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, dir := cloned(t, nil, "")
	rel, err := filepath.Rel(dir, outside)
	if err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{"..", "../..", filepath.ToSlash(rel), outside, "/etc", "kits/../..", "./kits", "kits/", "."} {
		src, _ := cloned(t, map[string]string{"kits/essentials.kit.yaml": "a kit"}, sub)
		tree, err := src.Read(t.Context())
		var refused *contract.OutsideError
		if !errors.As(err, &refused) || refused.At != sub {
			t.Errorf("source.path %q read %v with %v; want it refused as outside the clone", sub, readOf(tree.Files), err)
		}
	}
}

func TestALinkOnTheWayToAFileIsRefusedWhereverItPoints(t *testing.T) {
	outside := t.TempDir()
	for _, p := range []string{"host.machine.yaml", "planted.kit.yaml", "category/planted.yaml", "mini/host.machine.yaml", "kits/planted.kit.yaml"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(outside, p)), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(outside, p), []byte("outside"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		how, link, to, sub string
	}{
		{"a kit that links outside", "kits/planted.kit.yaml", filepath.Join(outside, "planted.kit.yaml"), ""},
		{"a kit that links inside the clone", "kits/twin.kit.yaml", "essentials.kit.yaml", ""},
		{"the computer's own file", "machines/mini/host.machine.yaml", filepath.Join(outside, "host.machine.yaml"), ""},
		{"the computer's folder", "machines/mini", filepath.Join(outside, "mini"), ""},
		{"the machines folder", "machines", outside, ""},
		{"a category of packages", "packages/category", filepath.Join(outside, "category"), ""},
		{"the packages folder", "packages", outside, ""},
		{"a folder of the sub-path", "infra", outside, "infra"},
	} {
		t.Run(tc.how, func(t *testing.T) {
			src, dir := cloned(t, map[string]string{"kits/essentials.kit.yaml": "a kit"}, tc.sub)
			link(t, tc.to, filepath.Join(dir, filepath.FromSlash(tc.link)))
			tree, err := src.Read(t.Context())
			var refused *contract.OutsideError
			if !errors.As(err, &refused) || refused.At != tc.link {
				t.Errorf("read %v with %v; want %s refused as a link", readOf(tree.Files), err, tc.link)
			}
		})
	}
}

func TestAFileOverTheBoundIsRefusedUnread(t *testing.T) {
	src, _ := cloned(t, map[string]string{"kits/huge.kit.yaml": strings.Repeat("x", contract.RepositoryFileSize+1)}, "")
	_, err := src.Read(t.Context())
	var file *catalog.FileError
	if !errors.As(err, &file) || file.File != "studio@1a2b3c4/kits/huge.kit.yaml" || !strings.Contains(file.Problem, "bytes") {
		t.Errorf("read a file past the bound with %v", err)
	}
}

func TestARepositoryPastTheFilesBoundIsRefused(t *testing.T) {
	held := map[string]string{}
	for i := range contract.RepositoryFiles + 1 {
		held["packages/many/p"+strconv.Itoa(i)+".yaml"] = "x"
	}
	src, _ := cloned(t, held, "")
	_, err := src.Read(t.Context())
	var file *catalog.FileError
	if !errors.As(err, &file) || !strings.Contains(file.Problem, "more than "+strconv.Itoa(contract.RepositoryFiles)) {
		t.Errorf("read %d files with %v", contract.RepositoryFiles+1, err)
	}
}

func TestAFileOutOfItsSlotIsRefusedAtItsPath(t *testing.T) {
	for _, tc := range []struct {
		held map[string]string
		file string
	}{
		{map[string]string{"kits/essentials.yaml": "x"}, "studio@1a2b3c4/kits/essentials.yaml"},
		{map[string]string{"kits/essentials.machine.yaml": "x"}, "studio@1a2b3c4/kits/essentials.machine.yaml"},
		{map[string]string{"machines/mini/notes.yaml": "x"}, "studio@1a2b3c4/machines/mini/notes.yaml"},
		{map[string]string{"packages/zstd.yaml": "x"}, "studio@1a2b3c4/packages/zstd.yaml"},
		{map[string]string{"kits/dev.kit.yaml": "x", "machines/mini/dev.machine.yaml": "y"}, "studio@1a2b3c4/kits/dev.kit.yaml"},
		{map[string]string{"machines/mini/host.machine.yaml": "x", "machines/mini/mini.machine.yaml": "y"}, "studio@1a2b3c4/machines/mini/mini.machine.yaml"},
	} {
		src, _ := cloned(t, tc.held, "")
		_, err := src.Read(t.Context())
		var file *catalog.FileError
		if !errors.As(err, &file) || file.File != tc.file {
			t.Errorf("%v read with %v; want %s refused", tc.held, err, tc.file)
		}
	}
}
