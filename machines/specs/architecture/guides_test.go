package architecture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/guide"
	"github.com/jterrazz/studio-station/apps/engine/engine/feeds"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

// invariant: rule guides: the act families the calls of calls.json name as
// their guide are the families feeds projects, one each.
func guidedFamilies(calls []byte) ([]string, error) {
	var c struct {
		Calls []struct {
			Guide string `json:"guide"`
		} `json:"calls"`
	}
	if err := json.Unmarshal(calls, &c); err != nil {
		return nil, err
	}
	var out []string
	for _, e := range c.Calls {
		if e.Guide != "" {
			out = append(out, e.Guide)
		}
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

func TestEveryGuidedCallHasItsProjection(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "apps", "engine", "remote", "api", "calls.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := guidedFamilies(data)
	if err != nil {
		t.Fatal(err)
	}
	projected := feeds.GuidedFamilies()
	slices.Sort(projected)
	for _, family := range projected {
		if _, ok := feeds.Guided(family); !ok {
			t.Errorf("guides: feeds names %s guided and projects it by nothing", family)
		}
	}
	if !slices.Equal(got, projected) {
		t.Errorf("guides: the calls of calls.json guide %v, feeds projects %v", got, projected)
	}
	mutant, err := guidedFamilies([]byte(`{"calls":[{"name":"machines.create-workspace"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if slices.Equal(mutant, projected) {
		t.Error("guides: a create that names no guide passes the rule")
	}
}

// value: the quoted names of a TypeScript array or union, in order.
func quoted(src string) []string {
	var out []string
	for _, m := range regexp.MustCompile(`'([a-z0-9-]+)'`).FindAllStringSubmatch(src, -1) {
		out = append(out, m[1])
	}
	return out
}

func tsBlock(t *testing.T, file, pattern string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), file))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(pattern).FindSubmatch(data)
	if m == nil {
		t.Fatalf("guides: %s holds no %s", file, pattern)
	}
	return quoted(string(m[1]))
}

func TestTheGuideWordsAreTheComponentsWords(t *testing.T) {
	// invariant: the window's drawings and the engine's pictures are one
	// list, the generated API's union among them, and so are the rail's
	// statuses: a picture one side adds alone turns this red.
	drawn := tsBlock(t, "apps/desktop/src/presentation/ui/illustrations/illustration.tsx", `(?s)ILLUSTRATIONS = \[(.*?)\] as const`)
	generated := tsBlock(t, "packages/api/index.d.ts", `export type CoreGuidePicture = ([^;]*);`)
	engine := guide.Picture("").Enum()
	slices.Sort(drawn)
	slices.Sort(generated)
	slices.Sort(engine)
	if !slices.Equal(drawn, engine) || !slices.Equal(generated, engine) {
		t.Errorf("guides: the engine names %v, the window draws %v, the API types %v", engine, drawn, generated)
	}
	statuses := tsBlock(t, "apps/desktop/src/presentation/ui/guide-rail.tsx", `export type GuideStepStatus = ([^;]*);`)
	own := guide.Status("").Enum()
	slices.Sort(statuses)
	slices.Sort(own)
	if !slices.Equal(statuses, own) {
		t.Errorf("guides: the engine's statuses are %v, the rail's %v", own, statuses)
	}
	if mutant := append(slices.Clone(engine), "fetch-bytes"); slices.Equal(drawn, mutant) {
		t.Error("guides: a picture the window does not draw passes the rule")
	}
}

func TestAGuideGestureCarriesNoCommand(t *testing.T) {
	// invariant: a Guide's gesture is a call and its args, or a page: the
	// window runs no command, so the type has no field to carry one.
	for _, name := range []string{"Command", "Cmd", "Shell", "Line"} {
		if _, ok := reflect.TypeFor[guide.Gesture]().FieldByName(name); ok {
			t.Errorf("guides: guide.Gesture carries a %s", name)
		}
	}
	vm := machines.VM{Name: "linux-2", State: machines.Stopped, Birth: &machines.Birth{Failed: true, Step: machines.BornBooting}}
	g := machines.CreateGuide(guide.Act{ID: "machines.create:linux-2", Family: machines.GuideCreate, Subject: "linux-2", Rail: machines.CreateRail(nil), Step: "booting", Ended: guide.EndedFailed}, &vm)
	inputs, err := callInputs(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, gesture := range []*guide.Gesture{g.View.Action, g.View.Quiet} {
		if gesture == nil {
			t.Fatal("guides: a create failed at boot offers no repair")
		}
		in, ok := inputs[gesture.Call]
		if !ok {
			t.Errorf("guides: %s is no call of calls.json", gesture.Call)
		}
		for arg := range gesture.Args {
			if _, ok := in.fields[arg]; !ok {
				t.Errorf("guides: %s's arg %s is no field of its input", gesture.Call, arg)
			}
		}
	}
}

func TestEveryStepOfACreateIsReachedInAStory(t *testing.T) {
	// invariant: a rail step exists only where a call reports it: each step
	// of a create's own rail stands reached (done, now, waiting or failed)
	// in some story of engine.guides.
	root := filepath.Join(repoRoot(t), "specs", "usecases", "engine", "guides")
	var told strings.Builder
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".spec.yaml") {
			data, _ := os.ReadFile(path)
			told.Write(data)
		}
		return nil
	})
	for _, r := range machines.CreateRail(nil) {
		reached := regexp.MustCompile(`"id":"` + regexp.QuoteMeta(r.ID) + `","label":"` + regexp.QuoteMeta(r.Label) + `","status":"(done|now|waiting|failed)"`)
		if !reached.MatchString(told.String()) {
			t.Errorf("guides: no story of engine.guides reaches the create's step %s", r.ID)
		}
	}
}
