package architecture_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
)

// value: the verbs of a call that reads and changes nothing; with every
// watch, the calls this rule sends to each world.
var readVerbs = []string{"diff", "drift", "explain", "list", "log", "matrix", "plan", "read", "sample", "show"}

// value: a world of its own for the answers no seed world carries: two-macs
// with a Machinefile on this Mac, whose plan has steps.
const machinefileWorld = "two-macs+machinefile"

// value: a world of its own for the wiki: one-mac whose Mac's wiki is the
// wiki family's fixture corpus, read at each of its pages and mapped open
// at each, since no seed world carries a wiki.
const wikiWorld = "one-mac+wiki"

// value: a call sent with its input, past the reads sent with an empty one.
type request struct{ call, in string }

// value: the seed of wikiWorld, written under dir, and a wiki.read and a
// wiki.map of each of its pages.
func wikiSeed(t *testing.T, root, dir string) (string, []request) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "specs", "usecases", "worlds", "one-mac.json"))
	if err != nil {
		t.Fatal(err)
	}
	var world map[string]any
	if err = json.Unmarshal(data, &world); err != nil {
		t.Fatal(err)
	}
	corpus, err := os.OpenRoot(filepath.Join(root, "apps", "engine", "wiki", "testdata", "corpus"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = corpus.Close() })
	files := map[string]string{}
	var asks []request
	err = fs.WalkDir(corpus.FS(), ".", func(p string, d fs.DirEntry, walked error) error {
		if walked != nil || d.IsDir() {
			return walked
		}
		body, read := fs.ReadFile(corpus.FS(), p)
		files[p] = string(body)
		if id, page := strings.CutSuffix(p, ".md"); page {
			in, _ := json.Marshal(id)
			asks = append(asks, request{"wiki.read", `{"Page":` + string(in) + `}`}, request{"wiki.map", `{"Open":` + string(in) + `}`})
		}
		return read
	})
	if err != nil || len(asks) == 0 {
		t.Fatalf("the wiki fixture corpus reads no page: %v", err)
	}
	macs, _ := world["macs"].(map[string]any)
	mac, _ := macs["macbook-pro"].(map[string]any)
	if mac == nil {
		t.Fatal("one-mac.json holds no macbook-pro")
	}
	mac["wiki"] = map[string]any{"head": "c1", "commits": map[string]any{"c1": files}}
	out, err := json.Marshal(world)
	if err != nil {
		t.Fatal(err)
	}
	seed := filepath.Join(dir, wikiWorld+".json")
	if err = os.WriteFile(seed, out, 0o600); err != nil {
		t.Fatal(err)
	}
	return seed, asks
}

type served struct {
	world string
	call  string
	out   any
}

// invariant: rule answers-match-schemas: every read of the registry that
// asks no confirmation, sent with an empty input to `guest serve --stdio`
// of the spec build over each world of specs/usecases/worlds/, answers
// frames frame.json admits, and each out is the one its call's schema in
// packages/api says. A full JSON Schema validator reads the schemas, so the
// generator's own reading of them is not what judges them.
func TestTheWorldsAnswersMatchTheirSchemas(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	bin := filepath.Join(t.TempDir(), "station-test")
	build := exec.Command("go", "build", "-tags", "world", "-o", bin, "./apps/engine/cmd/station")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build -tags world: %v\n%s", err, out)
	}
	reads := readCalls(t, root)
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	frame := compiled(t, c, root, "packages/api/schemas/frame.json", "")
	outs := map[string]*jsonschema.Schema{}
	for _, name := range reads {
		outs[name] = compiled(t, c, root, "packages/api/schemas/calls/"+name+".json", "#/properties/out")
	}
	seeds, err := filepath.Glob(filepath.Join(root, "specs", "usecases", "worlds", "*.json"))
	if err != nil || len(seeds) == 0 {
		t.Fatalf("no world under specs/usecases/worlds: %v", err)
	}
	worlds := map[string]string{machinefileWorld: filepath.Join(root, "specs", "usecases", "worlds", "two-macs.json")}
	wiki, asks := wikiSeed(t, root, t.TempDir())
	worlds[wikiWorld] = wiki
	for _, ask := range asks {
		if outs[ask.call] == nil {
			outs[ask.call] = compiled(t, c, root, "packages/api/schemas/calls/"+ask.call+".json", "#/properties/out")
		}
	}
	for _, seed := range seeds {
		worlds[strings.TrimSuffix(filepath.Base(seed), ".json")] = seed
	}
	var (
		mu     sync.Mutex
		got    []served
		counts = map[string]int{}
	)
	t.Run("worlds", func(t *testing.T) {
		for name, seed := range worlds {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				requests := make([]request, 0, len(reads))
				for _, read := range reads {
					requests = append(requests, request{read, "{}"})
				}
				if name == wikiWorld {
					requests = append(requests, asks...)
				}
				lines := serve(t, bin, name, seed, requests)
				for _, line := range lines {
					v, err := jsonschema.UnmarshalJSON(bytes.NewReader(line))
					if err != nil {
						t.Fatalf("an answer is no JSON: %v: %s", err, line)
					}
					if err := frame.Validate(v); err != nil {
						t.Errorf("a frame frame.json refuses: %s\n%v", clipped(line), err)
					}
					f, _ := v.(map[string]any)
					out, ok := f["out"]
					id, _ := f["id"].(string)
					var n int
					if _, err := fmt.Sscan(id, &n); !ok || err != nil || n < 1 || n > len(requests) {
						continue
					}
					name := requests[n-1].call
					if err := outs[name].Validate(out); err != nil {
						t.Errorf("%s: its out is not its schema's\n%v", name, err)
					}
					mu.Lock()
					got = append(got, served{world: t.Name(), call: name, out: out})
					counts[name]++
					mu.Unlock()
				}
			})
		}
	})
	for _, name := range reads {
		t.Logf("%s: %d worlds answered an out", name, counts[name])
	}
	t.Logf("%d outs validated, %d reads over %d worlds", len(got), len(reads), len(worlds))
	if len(got) == 0 {
		t.Fatal("no world answered an out: the rule reads nothing")
	}
	t.Run("bites", func(t *testing.T) {
		for _, s := range got {
			if s.call != "overview.read" || !replaced(s.out, "State", "bogus") {
				continue
			}
			if outs[s.call].Validate(s.out) == nil {
				t.Errorf("%s of %s with a State its enum does not name passed the rule", s.call, s.world)
			}
			return
		}
		t.Error("no overview.read answer carries a State to break")
	})
}

func readCalls(t *testing.T, root string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "apps", "engine", "remote", "api", "calls.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c call.Catalogue
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range c.Calls {
		_, verb, _ := strings.Cut(e.Name, ".")
		if !e.Own && e.Confirm == "" && (e.Watch || slices.Contains(readVerbs, verb)) {
			out = append(out, e.Name)
		}
	}
	return out
}

func compiled(t *testing.T, c *jsonschema.Compiler, root, path, pointer string) *jsonschema.Schema {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	id, _ := doc.(map[string]any)["$id"].(string)
	if err = c.AddResource(id, doc); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	s, err := c.Compile(id + pointer)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return s
}

// value: the answer lines of one session of `guest serve --stdio` over a
// fresh copy of a world: a hello, then each request by its index, from 1.
func serve(t *testing.T, bin, world, seed string, reads []request) [][]byte {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"bin", "root"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	env := []string{
		"HOME=" + dir, "PATH=" + filepath.Join(dir, "bin"), "NO_COLOR=1", "COLUMNS=80", "TZ=UTC",
		"STATION_GITHUB_API_URL=http://127.0.0.1:9", "STATION_MACHINE_KIND=mac",
		"STATION_KERNEL_STUB=" + filepath.Join(dir, "kernel"), "STATION_POWER_STUB=" + filepath.Join(dir, "power"),
		"STATION_SYSTEM_DIRS=", "STATION_SYSTEM_ROOT=" + filepath.Join(dir, "root"),
		"STATION_WORLD=" + filepath.Join(dir, "world.json"), "STATION_WORLD_SEED=" + seed, "STATION_TERMINAL=spec-terminal",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if world == machinefileWorld {
		edit := exec.CommandContext(ctx, bin, "world", "edit", "macbook-pro", "machinefiles/macbook-pro.machine.yaml")
		edit.Dir, edit.Env = dir, env
		edit.Stdin = strings.NewReader("machinefile: station.machinefile/v1\nname: macbook-pro\nmachine: { kind: computer, system: { os: macos } }\napps: [jq, node, ghostty]\n")
		if out, err := edit.CombinedOutput(); err != nil {
			t.Fatalf("world edit: %v\n%s", err, out)
		}
	}
	cmd := exec.CommandContext(ctx, bin, "guest", "serve", "--stdio")
	cmd.Dir, cmd.Env = dir, env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var frames bytes.Buffer
	frames.WriteString(`{"v":2,"id":"0","hello":{"version":"dev","calls":null}}` + "\n")
	for i, r := range reads {
		fmt.Fprintf(&frames, `{"v":2,"id":"%d","call":%q,"in":%s}`+"\n", i+1, r.call, r.in)
	}
	if _, err := io.Copy(in, &frames); err != nil {
		t.Fatal(err)
	}
	var lines [][]byte
	ended := map[string]bool{}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(nil, 64<<20)
	for len(ended) <= len(reads) && scanner.Scan() {
		line := slices.Clone(scanner.Bytes())
		lines = append(lines, line)
		var f struct {
			ID    string          `json:"id"`
			Hello json.RawMessage `json:"hello"`
			Out   json.RawMessage `json:"out"`
			Error json.RawMessage `json:"error"`
		}
		if json.Unmarshal(line, &f) == nil && (f.Hello != nil || f.Out != nil || f.Error != nil) {
			ended[f.ID] = true
		}
	}
	_ = in.Close()
	_ = cmd.Wait()
	if len(ended) <= len(reads) {
		t.Fatalf("%d of %d requests answered before the server ended: %v\n%s", len(ended), len(reads)+1, scanner.Err(), stderr.String())
	}
	return lines
}

func replaced(v any, key, value string) bool {
	switch v := v.(type) {
	case map[string]any:
		if _, ok := v[key].(string); ok {
			v[key] = value
			return true
		}
		for _, k := range slices.Sorted(maps.Keys(v)) {
			if replaced(v[k], key, value) {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if replaced(item, key, value) {
				return true
			}
		}
	}
	return false
}

func clipped(line []byte) string {
	if len(line) > 300 {
		return string(line[:300]) + "..."
	}
	return string(line)
}
