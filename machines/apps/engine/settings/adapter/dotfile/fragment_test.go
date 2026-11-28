package dotfile_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/settings/adapter/dotfile"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract/contracttest"
)

type disk struct{ local.FS }

func (disk) Lock(p string) (func(), error) { return local.Exclusive(context.Background(), p) }

// value: a home of the test's own, a fragment of Machines' and a setting
// that writes it into one file of that home in the mode given.
type bench struct {
	t       *testing.T
	home    string
	ours    fstest.MapFS
	applier *dotfile.Applier
	setting catalog.Setting
	file    string
	record  string
	files   disk
}

func newBench(t *testing.T, mode catalog.WriteMode, fragment, to string) *bench {
	t.Helper()
	home := t.TempDir()
	from := "pkg/fragment.json"
	if mode == catalog.WriteBlock {
		from = "pkg/fragment.conf"
	}
	ours := fstest.MapFS{from: {Data: []byte(fragment)}}
	files := disk{local.New([]string{"HOME=" + home}, nil, io.Discard, io.Discard).FS}
	record := filepath.Join(home, ".local", "state", "station", "merged.json")
	a := dotfile.New(files, silent{}, dotfile.Options{Home: home, Ours: ours, Record: record})
	it := catalog.Setting{Name: "fmt", Effect: catalog.Dotfiles{Mode: mode, Files: []catalog.Copy{{From: from, To: "~/" + to}}}}
	return &bench{t: t, home: home, ours: ours, applier: a, setting: it, file: filepath.Join(home, filepath.FromSlash(to)), record: record, files: files}
}

func (b *bench) write(text string, perm fs.FileMode) {
	b.t.Helper()
	if err := os.MkdirAll(filepath.Dir(b.file), 0o750); err != nil {
		b.t.Fatal(err)
	}
	if err := os.WriteFile(b.file, []byte(text), perm); err != nil {
		b.t.Fatal(err)
	}
	if err := b.files.Chmod(b.file, perm); err != nil {
		b.t.Fatal(err)
	}
}

func (b *bench) read() string {
	b.t.Helper()
	data, err := os.ReadFile(b.file)
	if err != nil {
		b.t.Fatal(err)
	}
	return string(data)
}

func (b *bench) apply() []string {
	b.t.Helper()
	change, err := b.applier.Apply(b.t.Context(), b.setting, nil)
	if err != nil {
		b.t.Fatalf("Apply: %v", err)
	}
	return said(change.Steps)
}

func (b *bench) off() []string {
	b.t.Helper()
	change, err := b.applier.Undo(b.t.Context(), b.setting)
	if err != nil {
		b.t.Fatalf("Undo: %v", err)
	}
	return said(change.Steps)
}

func said[S ~[]E, E any](steps S) []string {
	out := make([]string, 0, len(steps))
	for _, s := range steps {
		v := reflect.ValueOf(s)
		out = append(out, v.FieldByName("Label").String()+": "+v.FieldByName("Detail").String())
	}
	return out
}

func TestApplierMerging(t *testing.T) {
	contracttest.Applier(t, func(t *testing.T) contracttest.Opened {
		b := newBench(t, catalog.WriteMerge, `{"tabWidth": 4}`, ".config/fmt/settings.json")
		return contracttest.Opened{Applier: b.applier, Package: b.setting, Edit: func(*testing.T) { b.write("{\n  \"tabWidth\": 2\n}\n", 0o600) }}
	})
}

func TestApplierBlocking(t *testing.T) {
	contracttest.Applier(t, func(t *testing.T) contracttest.Opened {
		b := newBench(t, catalog.WriteBlock, "set -g mouse on\n", ".tmux.conf")
		return contracttest.Opened{Applier: b.applier, Package: b.setting, Edit: func(*testing.T) {
			b.write(strings.Replace(b.read(), "mouse on", "mouse off", 1), 0o600)
		}}
	})
}

const owners = `{
  "theme": "dark",
  "editor": {
    "tabSize": 2
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "command": "mine"
      }
    ]
  }
}
`

const hook = `{
  "editor": {"tabSize": 4, "wrap": true},
  "hooks": {"PreToolUse": [{"matcher": "Bash", "command": "rtk"}], "Stop": [{"command": "rtk"}]}
}`

func TestAMergeKeepsTheOwnersKeysAndOffGivesBackHisBytes(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, hook, ".claude/settings.json")
	b.write(owners, 0o644)
	if got := b.apply(); !reflect.DeepEqual(got, []string{"~/.claude/settings.json: merged", "~/.claude/settings.json /editor/tabSize: yours, kept"}) {
		t.Errorf("apply said %q", got)
	}
	want := `{
  "theme": "dark",
  "editor": {
    "tabSize": 2,
    "wrap": true
  },
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "command": "mine"
      },
      {
        "matcher": "Bash",
        "command": "rtk"
      }
    ],
    "Stop": [
      {
        "command": "rtk"
      }
    ]
  }
}
`
	if got := b.read(); got != want {
		t.Errorf("merged into\n%s\nwant\n%s", got, want)
	}
	if got := b.off(); !reflect.DeepEqual(got, []string{"~/.claude/settings.json: taken back"}) {
		t.Errorf("off said %q", got)
	}
	if got := b.read(); got != owners {
		t.Errorf("off left\n%s\nwant the owner's bytes\n%s", got, owners)
	}
}

func TestAMergeAppliedTwiceWritesNothing(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, hook, ".claude/settings.json")
	b.write(owners, 0o644)
	b.apply()
	first, err := b.files.Stat(b.file)
	if err != nil {
		t.Fatal(err)
	}
	record, _ := os.ReadFile(b.record)
	if got := b.apply(); got[0] != "~/.claude/settings.json: as Machines wrote it: nothing written" {
		t.Errorf("a second apply said %q", got)
	}
	if again, _ := b.files.Stat(b.file); !again.ModTime().Equal(first.ModTime()) {
		t.Errorf("a second apply wrote the file again")
	}
	if after, _ := os.ReadFile(b.record); !bytes.Equal(record, after) {
		t.Errorf("a second apply moved the record:\n%s\nthen\n%s", record, after)
	}
}

func TestAnArrayGainsOnlyTheElementsItLacks(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"allow": ["git", "rtk", "rtk"]}`, ".claude/settings.json")
	b.write("{\n  \"allow\": [\n    \"git\"\n  ]\n}\n", 0o600)
	b.apply()
	if got := b.read(); got != "{\n  \"allow\": [\n    \"git\",\n    \"rtk\"\n  ]\n}\n" {
		t.Errorf("merged into %q", got)
	}
	b.apply()
	b.off()
	if got := b.read(); got != "{\n  \"allow\": [\n    \"git\"\n  ]\n}\n" {
		t.Errorf("off took the owner's element or left Machines': %q", got)
	}
}

func TestOffLeavesAValueChangedByHandAndSaysIt(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"tabWidth": 4, "useTabs": false}`, ".oxfmtrc.json")
	b.write("{\n    \"printWidth\": 100\n}\n", 0o600)
	b.apply()
	b.write(strings.Replace(b.read(), `"tabWidth": 4`, `"tabWidth": 8`, 1), 0o600)
	if got := b.off(); !reflect.DeepEqual(got, []string{"~/.oxfmtrc.json: taken back", "~/.oxfmtrc.json /tabWidth: yours, kept"}) {
		t.Errorf("off said %q", got)
	}
	if got := b.read(); got != "{\n    \"printWidth\": 100,\n    \"tabWidth\": 8\n}\n" {
		t.Errorf("off left %q", got)
	}
}

func TestAValueMachinesWroteFollowsTheFragmentAndOneOfTheOwnersStays(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": 1, "b": 1}`, ".x.json")
	b.write("{\n  \"b\": 0\n}\n", 0o600)
	b.apply()
	b.ours["pkg/fragment.json"].Data = []byte(`{"a": 2, "b": 2}`)
	b.apply()
	if got := b.read(); got != "{\n  \"b\": 0,\n  \"a\": 2\n}\n" {
		t.Errorf("a moved fragment wrote %q", got)
	}
	b.ours["pkg/fragment.json"].Data = []byte(`{"b": 2}`)
	b.apply()
	if got := b.read(); got != "{\n  \"b\": 0\n}\n" {
		t.Errorf("a value the fragment dropped stayed: %q", got)
	}
}

func TestAFileThatIsNotStrictJSONIsRefusedAndLeftAsItIs(t *testing.T) {
	for name, text := range map[string]string{
		"a comment":        "{\n  // mine\n  \"a\": 1\n}\n",
		"a trailing comma": "{\n  \"a\": 1,\n}\n",
		"a key twice":      "{\"a\": 1, \"a\": 2}\n",
		"two values":       "{} {}\n",
		"no object":        "[1]\n",
	} {
		t.Run(name, func(t *testing.T) {
			b := newBench(t, catalog.WriteMerge, `{"b": 2}`, ".config/zed/settings.json")
			b.write(text, 0o600)
			_, err := b.applier.Apply(t.Context(), b.setting, nil)
			if err == nil || !strings.HasPrefix(err.Error(), "~/.config/zed/settings.json ") || !strings.HasSuffix(err.Error(), "Machines writes nothing into it") {
				t.Errorf("Apply: %v; want a refusal naming the file", err)
			}
			if got := b.read(); got != text {
				t.Errorf("the file was rewritten: %q", got)
			}
		})
	}
}

func TestAMissingFileIsMadeWithItsFoldersAndOffTakesThemBack(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": {"b": [1]}}`, ".config/new/deep/x.json")
	if err := os.MkdirAll(filepath.Join(b.home, ".config"), 0o750); err != nil {
		t.Fatal(err)
	}
	b.apply()
	if got := b.read(); got != "{\n  \"a\": {\n    \"b\": [\n      1\n    ]\n  }\n}\n" {
		t.Errorf("made %q", got)
	}
	if info, _ := b.files.Stat(b.file); info.Mode().Perm() != 0o600 {
		t.Errorf("made %v, want 0600", info.Mode().Perm())
	}
	if got := b.off(); !reflect.DeepEqual(got, []string{"~/.config/new/deep/x.json: removed"}) {
		t.Errorf("off said %q", got)
	}
	if _, err := b.files.Stat(filepath.Join(b.home, ".config", "new")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a folder Machines made stayed: %v", err)
	}
	if _, err := b.files.Stat(filepath.Join(b.home, ".config")); err != nil {
		t.Errorf("the owner's folder went: %v", err)
	}
	if _, err := b.files.Stat(b.record); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a record of nothing stayed: %v", err)
	}
}

func TestAFileKeepsItsModeAndItsLink(t *testing.T) {
	b := newBench(t, catalog.WriteMerge, `{"a": 1}`, ".x.json")
	target := filepath.Join(b.home, "dotfiles", "x.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := b.files.Chmod(target, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := b.files.Symlink("dotfiles/x.json", b.file); err != nil {
		t.Fatal(err)
	}
	b.apply()
	if to, err := b.files.Readlink(b.file); err != nil || to != "dotfiles/x.json" {
		t.Errorf("the link is now %q, %v", to, err)
	}
	info, _ := b.files.Stat(target)
	if data, _ := os.ReadFile(target); string(data) != "{\n  \"a\": 1\n}\n" || info.Mode().Perm() != 0o644 {
		t.Errorf("the linked file holds %q in %v", data, info.Mode().Perm())
	}
}

func TestTheBoundsAreHeld(t *testing.T) {
	deep := strings.Repeat(`{"a":`, 33) + "1" + strings.Repeat("}", 33)
	for name, c := range map[string]struct{ fragment, file, want string }{
		"a file past 1 MiB":         {`{"a": 1}`, `{"b": "` + strings.Repeat("x", 1<<20) + `"}`, "past 1 MiB"},
		"a fragment past 256 KiB":   {`{"b": "` + strings.Repeat("x", 256<<10) + `"}`, "{}", "past 256 KiB"},
		"a file deeper than 32":     {`{"a": 1}`, deep, "nests deeper than 32"},
		"a fragment deeper than 32": {deep, "{}", "nests deeper than 32"},
	} {
		t.Run(name, func(t *testing.T) {
			b := newBench(t, catalog.WriteMerge, c.fragment, ".x.json")
			b.write(c.file, 0o600)
			if _, err := b.applier.Apply(t.Context(), b.setting, nil); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("Apply: %v; want %q", err, c.want)
			}
			if b.read() != c.file {
				t.Errorf("the file was rewritten")
			}
		})
	}
}

func TestABlockIsWrittenOnceAndTakenBackToTheOwnersBytes(t *testing.T) {
	b := newBench(t, catalog.WriteBlock, "set -g mouse on\n", ".tmux.conf")
	b.write("set -g base-index 1\n", 0o644)
	b.apply()
	if got := b.read(); got != "set -g base-index 1\n# >>> station:fmt >>>\nset -g mouse on\n# <<< station:fmt <<<\n" {
		t.Errorf("wrote %q", got)
	}
	if got := b.apply(); got[0] != "~/.tmux.conf [station:fmt]: as Machines wrote it: nothing written" {
		t.Errorf("a second apply said %q", got)
	}
	b.off()
	if got := b.read(); got != "set -g base-index 1\n" {
		t.Errorf("off left %q", got)
	}
}

func TestABlockEditedByHandIsLeftAndSaid(t *testing.T) {
	b := newBench(t, catalog.WriteBlock, "set -g mouse on\n", ".tmux.conf")
	b.apply()
	edited := strings.Replace(b.read(), "mouse on", "mouse off", 1)
	b.write(edited, 0o600)
	if got := b.off(); !reflect.DeepEqual(got, []string{"~/.tmux.conf [station:fmt]: yours, kept"}) {
		t.Errorf("off said %q", got)
	}
	if b.read() != edited {
		t.Errorf("off took the owner's block")
	}
}

func TestADamagedBlockIsRefused(t *testing.T) {
	b := newBench(t, catalog.WriteBlock, "set -g mouse on\n", ".tmux.conf")
	b.write("# >>> station:fmt >>>\nset -g mouse on\n", 0o600)
	if _, err := b.applier.Apply(t.Context(), b.setting, nil); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Errorf("Apply: %v; want the damaged block refused", err)
	}
}

// value: a random JSON value of a few shapes, its keys drawn so that two
// trees meet on some of them, `/` and `~` among them.
func random(r *rand.Rand, depth int) any {
	switch n := r.IntN(10); {
	case depth > 3 || n < 4:
		switch r.IntN(4) {
		case 0:
			return r.IntN(3)
		case 1:
			return []string{"x", "y", "é"}[r.IntN(3)]
		case 2:
			return r.IntN(2) == 0
		}
		return nil
	case n < 7:
		out := map[string]any{}
		for range r.IntN(4) {
			out[[]string{"a", "b", "c/d", "e~f", "g"}[r.IntN(5)]] = random(r, depth+1)
		}
		return out
	default:
		out := []any{}
		for range r.IntN(3) {
			out = append(out, random(r, depth+1))
		}
		return out
	}
}

func object(r *rand.Rand) map[string]any {
	for {
		if m, ok := random(r, 0).(map[string]any); ok {
			return m
		}
	}
}

// value: whether every value of the owner's is still in the merged tree:
// a key with what it held, an array with its elements first, in order.
func keeps(merged, mine any) bool {
	switch m := mine.(type) {
	case map[string]any:
		got, ok := merged.(map[string]any)
		if !ok {
			return false
		}
		for k, v := range m {
			if !keeps(got[k], v) {
				return false
			}
		}
		return true
	case []any:
		got, ok := merged.([]any)
		return ok && len(got) >= len(m) && reflect.DeepEqual(got[:len(m)], m)
	}
	return reflect.DeepEqual(merged, mine)
}

func TestAnyMergeKeepsTheOwnersTreeIsWrittenOnceAndOffGivesBackHisBytes(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 2026)) //nolint:gosec // value: a seeded generator, so a failing case is found again by its number.
	for i := range 400 {
		indent := []string{"  ", "    ", "\t"}[i%3]
		mine, err := json.MarshalIndent(object(r), "", indent)
		if err != nil {
			t.Fatal(err)
		}
		fragment, _ := json.Marshal(object(r))
		original := string(mine) + "\n"
		b := newBench(t, catalog.WriteMerge, string(fragment), ".x.json")
		b.write(original, 0o600)
		if _, err := b.applier.Apply(t.Context(), b.setting, nil); err != nil {
			if strings.Contains(err.Error(), "writes no value") {
				continue
			}
			t.Fatalf("case %d: Apply: %v", i, err)
		}
		merged := b.read()
		var before, after any
		_ = json.Unmarshal([]byte(original), &before)
		if err := json.Unmarshal([]byte(merged), &after); err != nil || !keeps(after, before) {
			t.Fatalf("case %d: the merge lost the owner's values (%v):\n%s\ninto\n%s", i, err, original, merged)
		}
		b.apply()
		if again := b.read(); again != merged {
			t.Fatalf("case %d: a second apply changed\n%s\ninto\n%s", i, merged, again)
		}
		b.off()
		if got := b.read(); got != original {
			t.Fatalf("case %d: fragment %s: off left\n%s\nnot the owner's\n%s", i, fragment, got, original)
		}
	}
}

func TestTheIndentIsTheFilesOwn(t *testing.T) {
	for _, indent := range []string{"  ", "    ", "\t"} {
		b := newBench(t, catalog.WriteMerge, `{"b": {"c": 1}}`, ".x.json")
		b.write("{\n"+indent+"\"a\": 1\n}", 0o600)
		b.apply()
		want := "{\n" + indent + "\"a\": 1,\n" + indent + "\"b\": {\n" + strings.Repeat(indent, 2) + "\"c\": 1\n" + indent + "}\n}"
		if got := b.read(); got != want {
			t.Errorf("indent %s: wrote %q, want %q", strconv.Quote(indent), got, want)
		}
	}
}
