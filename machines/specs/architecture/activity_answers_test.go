package architecture_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

// invariant: rule activity-answers-match-schemas: the activity.read answers
// the window's recordings carry, a graph and a history among them, and an
// engine.activity answer of a series the engine kept, its minutes' gauges
// and a gap included, are each the one its call's schema in packages/api
// says.
func TestTheActivityAnswersMatchTheirSchemas(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	read := compiled(t, c, root, "packages/api/schemas/calls/activity.read.json", "#/properties/out")
	kept := compiled(t, c, root, "packages/api/schemas/calls/engine.activity.json", "#/properties/out")

	files, gerr := filepath.Glob(filepath.Join(root, "apps", "desktop", "src", "fixtures", "worlds", "*", "activity.read.json"))
	if gerr != nil || len(files) == 0 {
		t.Fatalf("no recording of activity.read: %v", gerr)
	}
	var outs, rated, history int
	for _, file := range files {
		data, rerr := os.ReadFile(file)
		if rerr != nil {
			t.Fatal(rerr)
		}
		var rec struct {
			Exchanges []struct {
				Answers []map[string]json.RawMessage `json:"answers"`
			} `json:"exchanges"`
		}
		if err := json.Unmarshal(data, &rec); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for _, x := range rec.Exchanges {
			for _, a := range x.Answers {
				out, ok := a["out"]
				if !ok {
					continue
				}
				v, err := jsonschema.UnmarshalJSON(bytes.NewReader(out))
				if err != nil {
					t.Fatalf("%s: %v", file, err)
				}
				if err := read.Validate(v); err != nil {
					t.Errorf("%s: an activity.read out is not its schema's\n%v", file, err)
				}
				outs++
				if bytes.Contains(out, []byte(`"rated"`)) {
					rated++
				}
				if bytes.Contains(out, []byte(`"history"`)) {
					history++
				}
			}
		}
	}
	if outs == 0 || rated == 0 || history == 0 {
		t.Errorf("the recordings hold %d activity.read outs, %d with a rated point, %d with a history; want some of each", outs, rated, history)
	}

	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	h := activity.NewHistory(from)
	x := activity.Sample{At: from, CPUs: 4, MemTotal: 64 << 30, Load: [3]float64{1.5, 1, 0.5}, DiskFree: 400 << 30, Battery: &activity.Battery{Percent: 80}}
	for i := range 90 {
		x.At = from.Add(time.Duration(i) * time.Minute)
		x.CPUTotal, x.CPUBusy, x.MemUsed = x.CPUTotal+24_000, x.CPUBusy+6_000, x.MemUsed+1<<20
		if i == 60 {
			x.CPUTotal, x.CPUBusy, x.Boot = 1_000, 100, x.At
		}
		h.Add("macbook-pro", activity.Reading{Sample: x})
	}
	now := x.At
	part, ok := h.Part("macbook-pro", activity.Want{Graph: time.Hour, History: true}, now)
	if !ok {
		t.Fatal("the history keeps nothing of the machine it was given")
	}
	data, err := json.Marshal(engine.ActivityOut{Since: from, Kept: &engine.Kept{Last: part.Last, Series: part.Series}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"gauges"`) {
		t.Fatalf("the engine.activity answer carries no gauges: %s", clipped(data))
	}
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if err = kept.Validate(v); err != nil {
		t.Errorf("an engine.activity out is not its schema's\n%v", err)
	}
}
