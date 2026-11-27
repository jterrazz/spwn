package files_test

import (
	"context"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

// value: the longest one input may take to decode and resolve; past it the
// read is no longer total, as a loop or an exponential parse would be.
const inputBudget = 10 * time.Second

type read struct {
	file    catalog.Machinefile
	decoded string
	overlay string
	loaded  string
	steps   []machinefile.Step
	refused string
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// invariant: what a Machinefile's body becomes, from its decode to its
// resolution against the built-in catalog, as the owner's file named
// `fuzzed`, and as its overlay.
func readMachinefile(builtin machinefile.Tree, body []byte) read {
	var r read
	var err error
	r.file, err = catalog.DecodeMachinefile("~/.config/station/machinefiles/fuzzed"+catalog.MachinefileSuffix, body)
	r.decoded = errText(err)
	_, err = catalog.DecodeMachinefile("~/.config/station/machinefiles/fuzzed"+catalog.OverlaySuffix, body)
	r.overlay = errText(err)
	user := machinefile.Tree{Kind: machinefile.User, Root: "~/.config/station", Files: []machinefile.File{{Path: "machinefiles/fuzzed" + catalog.MachinefileSuffix, Body: body}}}
	lib, err := machinefile.Load([]machinefile.Tree{user, builtin})
	r.loaded = errText(err)
	if err != nil {
		return r
	}
	resolved, err := machinefile.Resolve(lib, "fuzzed", machinefile.Machine{}, machinefile.Lock{})
	r.steps, r.refused = resolved.Steps, errText(err)
	return r
}

func FuzzDecodeMachinefile(f *testing.F) {
	builtin, err := files.New(files.Embedded{FS: catalog.BuiltinFiles()}, contract.NoLinks, "packages", "machines", machinefile.Builtin, catalog.BuiltinOrigin).Read(f.Context())
	if err != nil {
		f.Fatal(err)
	}
	for _, file := range builtin.Files {
		if path.Dir(file.Path) == "machinefiles" {
			f.Add(file.Body)
		}
	}
	for _, body := range []string{
		"machinefile: station.machinefile/v1\nname: fuzzed\nmachine: { kind: computer, system: { os: ubuntu } }\napps: [foo]\n",
		"machinefile: station.machinefile/v1\nname: fuzzed\nmachine: { kind: workspace, system: { os: ubuntu, version: '24.04' } }\napps: [vite, expo]\n",
		"machinefile: station.machinefile/v1\nname: fuzzed\ndescription: d\nmachine: { kind: computer, system: { os: macos } }\napps: [homebrew, jq, ghostty]\nsettings:\n    hushlogin: {}\n",
		"machinefile: station.machinefile/v1\nname: fuzzed\nextends: [base-debian]\nprefer: [cask]\nports: { open: 3000-3999, to: private, audiences: { 5432: local, 3001: public } }\nrun: { boot: 'true' }\n",
		"machinefile: station.machinefile/v1\noverrides: fuzzed\nports: { except: [3306] }\n",
		"machinefile: station.machinefile/v1\nname: fuzzed\nmachine: &m { kind: computer, system: { os: macos } }\napps: *m\n",
		"machinefile: station.machinefile/v1\nname: fuzzed\napps: [[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[x]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]\n",
		"a: &a [x, x, x, x, x, x, x, x, x]\nb: &b [*a, *a, *a, *a, *a, *a, *a, *a, *a]\nc: &c [*b, *b, *b, *b, *b, *b, *b, *b, *b]\nd: [*c, *c, *c, *c, *c, *c, *c, *c, *c]\n",
		"---\n",
		"--- !!binary\n",
		"",
	} {
		f.Add([]byte(body))
	}
	f.Fuzz(func(t *testing.T, body []byte) {
		var first, second read
		ctx, cancel := context.WithTimeout(t.Context(), inputBudget)
		defer cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			first, second = readMachinefile(builtin, body), readMachinefile(builtin, body)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatalf("reading %q took more than %v", body, inputBudget)
		}
		if !reflect.DeepEqual(first, second) {
			t.Errorf("reading %q twice answered\n%+v\nthen\n%+v", body, first, second)
		}
	})
}
