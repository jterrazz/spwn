package contract_test

import (
	"errors"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

const header = "machinefile: station.machinefile/v1\n"

func pkg(name, description string) string {
	return "package: station.package/v1\nkind: app\nname: " + name + "\ndescription: " + description + "\nmethods:\n    - { os: any, script: { run: 'true' } }\n"
}

// value: the sources a computer named mini reads: the owner's files, a
// shared source's repository, the built-in packages.
func library(t *testing.T, user, repo map[string]string) (machinefile.Library, error) {
	t.Helper()
	held := fstest.MapFS{}
	for p, body := range repo {
		held[p] = &fstest.MapFile{Data: []byte(body)}
	}
	files, err := contract.Repository{Dir: counted{MapFS: held, opens: new(int)}, At: ".", Computer: "mini", Source: "studio", Root: "studio@1a2b3c4"}.Read()
	if err != nil {
		t.Fatal(err)
	}
	tree := func(kind machinefile.OriginKind, root string, held map[string]string) machinefile.Tree {
		tr := machinefile.Tree{Kind: kind, Root: root}
		for p, body := range held {
			tr.Files = append(tr.Files, machinefile.File{Path: p, Body: []byte(body)})
		}
		return tr
	}
	return machinefile.Load([]machinefile.Tree{
		tree(machinefile.User, "~/.config/station", user),
		{Kind: machinefile.Shared, Root: "studio@1a2b3c4", Files: files},
		tree(machinefile.Builtin, "built-in", map[string]string{"catalog/command-line/fd.yaml": pkg("fd", "the built-in fd"), "catalog/command-line/jq.yaml": pkg("jq", "json")}),
	})
}

// value: a repository held in memory, which counts every entry opened.
type counted struct {
	fstest.MapFS
	opens *int
}

func (c counted) OpenUnfollowed(name string) (fs.File, error) {
	*c.opens++
	return c.Open(name)
}

func TestEachFolderIsListedOnceAndEveryEntryCounted(t *testing.T) {
	held := fstest.MapFS{}
	for i := range 100 {
		held["packages/c"+strconv.Itoa(i)+"/p.yaml"] = &fstest.MapFile{Data: []byte(pkg("p"+strconv.Itoa(i), "one"))}
	}
	opens := 0
	files, err := contract.Repository{Dir: counted{MapFS: held, opens: &opens}, At: ".", Computer: "mini", Source: "studio", Root: "studio@1a2b3c4"}.Read()
	if err != nil || len(files) != 100 {
		t.Fatalf("read %d files with %v", len(files), err)
	}
	if want := 2 + 2*100; opens != want {
		t.Errorf("opened %d entries; want %d, the top, packages/, each category and each file once", opens, want)
	}
}

func TestAWalkPastItsBudgetIsRefusedBeforeItListsTheRest(t *testing.T) {
	held := fstest.MapFS{}
	for i := range 5000 {
		held["packages/c"+strconv.Itoa(i)] = &fstest.MapFile{Mode: fs.ModeDir | 0o755}
	}
	opens := 0
	_, err := contract.Repository{Dir: counted{MapFS: held, opens: &opens}, At: ".", Computer: "mini", Source: "studio", Root: "studio@1a2b3c4"}.Read()
	var file *catalog.FileError
	if !errors.As(err, &file) || !strings.Contains(file.Problem, "more than "+strconv.Itoa(contract.RepositoryEntries)+" entries") {
		t.Errorf("read 5000 empty categories with %v; want them refused past the budget", err)
	}
	if file != nil && file.File != "studio@1a2b3c4/packages" {
		t.Errorf("the refusal names %q; want studio@1a2b3c4/packages, the folder that crossed the bound", file.File)
	}
	if opens != 2 {
		t.Errorf("opened %d entries; want 2, the top and packages/, refused as it is listed", opens)
	}
}

// value: a repository held in memory where the entry at `at` becomes
// `to` the first time it is opened, after the listing that judged it.
type turned struct {
	fstest.MapFS
	at string
	to *fstest.MapFile
}

func (s *turned) OpenUnfollowed(name string) (fs.File, error) {
	if name == s.at && s.to != nil {
		s.MapFS[name], s.to = s.to, nil
	}
	return s.Open(name)
}

func TestAFileTurnedIntoAPipeAfterItsListingIsRefusedByItsHandle(t *testing.T) {
	held := fstest.MapFS{"kits/essentials.kit.yaml": &fstest.MapFile{Data: []byte(header + "name: essentials\n")}}
	pipe := &fstest.MapFile{Mode: fs.ModeNamedPipe | 0o600}
	_, err := contract.Repository{Dir: &turned{MapFS: held, at: "kits/essentials.kit.yaml", to: pipe}, At: ".", Computer: "mini", Source: "studio", Root: "studio@1a2b3c4"}.Read()
	var file *catalog.FileError
	if !errors.As(err, &file) || file.File != "studio@1a2b3c4/kits/essentials.kit.yaml" || !strings.Contains(file.Problem, "no regular file") {
		t.Errorf("read a kit turned into a pipe with %v; want it refused as no regular file", err)
	}
}

func computer(body string) string {
	return header + "name: mini\nmachine: { kind: computer, system: { os: macos } }\n" + body
}

func TestASharedSourcesFilesAreNamedByTheirSlotAndResolvedThroughTheirKits(t *testing.T) {
	lib, err := library(t, nil, map[string]string{
		"machines/mini/host.machine.yaml":              computer("extends: [essentials]\n"),
		"machines/mini/development-linux.machine.yaml": header + "name: development-linux\nextends: [essentials]\nmachine: { kind: workspace, system: { os: ubuntu } }\n",
		"kits/essentials.kit.yaml":                     header + "name: essentials\napps: [fd, zstd]\n",
		"packages/command-line/zstd.yaml":              pkg("zstd", "compression"),
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := machinefile.Resolve(lib, "mini", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	want := []machinefile.Origin{
		{Name: "essentials", Kind: machinefile.Shared, File: "studio@1a2b3c4/kits/essentials.kit.yaml"},
		{Name: "mini", Kind: machinefile.Shared, File: "studio@1a2b3c4/machines/mini/host.machine.yaml"},
	}
	if !slices.Equal(r.Files, want) {
		t.Errorf("mini read %+v, want %+v", r.Files, want)
	}
	if _, err := machinefile.Resolve(lib, "development-linux", machinefile.Machine{}, machinefile.Lock{}); err != nil {
		t.Errorf("the VM's file resolved with %v", err)
	}
	zstd, ok := lib.Catalog.Package("zstd")
	if !ok || zstd.Origin != string(machinefile.Shared) || zstd.File != "studio@1a2b3c4/packages/command-line/zstd.yaml" {
		t.Errorf("the repository's package joined the catalog as %+v, %v", zstd, ok)
	}
}

func TestTheOwnersFileReplacesTheSharedOneAndTheSharedOneTheBuiltIn(t *testing.T) {
	lib, err := library(t, map[string]string{
		"machinefiles/essentials.machine.yaml": header + "name: essentials\napps: [jq]\n",
	}, map[string]string{
		"machines/mini/host.machine.yaml": computer("extends: [essentials]\n"),
		"kits/essentials.kit.yaml":        header + "name: essentials\napps: [fd]\n",
		"packages/command-line/fd.yaml":   pkg("fd", "the shared fd"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := lib.Machinefiles["essentials"]; got.Kind != machinefile.User {
		t.Errorf("essentials read from %s, want the owner's", got.Kind)
	}
	if fd, _ := lib.Catalog.Package("fd"); fd.Package.Description != "the shared fd" {
		t.Errorf("fd read as %q, want the shared one over the built-in", fd.Package.Description)
	}
}

func TestASharedSourcesFileSaysWhatItsSlotAllows(t *testing.T) {
	for _, tc := range []struct {
		how, file, body, says string
		line                  int
	}{
		{"a kit that says its machine", "kits/essentials.kit.yaml", header + "name: essentials\nmachine: { kind: workspace }\n", "a kit composes", 3},
		{"a kit that says how a VM runs", "kits/essentials.kit.yaml", header + "name: essentials\nvm: { image: { from: ubuntu } }\n", "`vm` is a machine file's", 3},
		{"a kit named apart from its file", "kits/essentials.kit.yaml", header + "name: basics\n", `names "basics", its file "essentials"`, 2},
		{"a computer's file that says a workspace", "machines/mini/host.machine.yaml", header + "name: mini\nmachine: { kind: workspace }\n", "it says `machine: { kind: computer }`", 3},
		{"a computer's file named apart from its computer", "machines/mini/host.machine.yaml", header + "name: host\nmachine: { kind: computer }\n", `names "host", its file "mini"`, 2},
		{"a VM's file that says a computer", "machines/mini/api.machine.yaml", header + "name: api\nmachine: { kind: computer }\n", "a VM's file says", 3},
	} {
		t.Run(tc.how, func(t *testing.T) {
			_, err := library(t, nil, map[string]string{tc.file: tc.body})
			var refused *catalog.FileError
			if !errors.As(err, &refused) || refused.File != "studio@1a2b3c4/"+tc.file || refused.Line != tc.line || !strings.Contains(refused.Problem, tc.says) {
				t.Errorf("read with %v; want %s:%d refused, saying %q", err, tc.file, tc.line, tc.says)
			}
		})
	}
}

func TestAComputersChainReadsNoUpgradeLineOfTheSharedSource(t *testing.T) {
	nightly := header + "name: nightly\nrun: { upgrade: echo up }\n"
	for _, tc := range []struct {
		how           string
		user, repo    map[string]string
		resolve, says string
	}{
		{"the computer's own file", nil, map[string]string{"machines/mini/host.machine.yaml": computer("run: { upgrade: echo up }\n")}, "mini", "mini"},
		{"a kit the computer extends", nil, map[string]string{"machines/mini/host.machine.yaml": computer("extends: [nightly]\n"), "kits/nightly.kit.yaml": nightly}, "mini", "nightly"},
		{"a kit a VM extends", nil, map[string]string{"machines/mini/api.machine.yaml": header + "name: api\nextends: [nightly]\nmachine: { kind: workspace }\n", "kits/nightly.kit.yaml": nightly}, "api", ""},
		{"the owner's own file of the computer", map[string]string{"machinefiles/mini.machine.yaml": computer("run: { upgrade: echo up }\n")}, nil, "mini", ""},
	} {
		t.Run(tc.how, func(t *testing.T) {
			lib, err := library(t, tc.user, tc.repo)
			if err != nil {
				t.Fatal(err)
			}
			_, err = machinefile.Resolve(lib, tc.resolve, machinefile.Machine{}, machinefile.Lock{})
			var upgrade *machinefile.UpgradeOnComputerError
			switch {
			case tc.says != "" && (!errors.As(err, &upgrade) || upgrade.Says != tc.says || upgrade.Name != tc.resolve):
				t.Errorf("resolved with %v; want the run.upgrade of %s refused", err, tc.says)
			case tc.says == "" && err != nil:
				t.Errorf("resolved with %v; want no refusal", err)
			}
		})
	}
}
