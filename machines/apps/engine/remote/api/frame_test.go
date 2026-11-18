package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
)

type described struct {
	Properties map[string]json.RawMessage `json:"properties"`
	Required   []string                   `json:"required"`
}

func frameSchema(t *testing.T) map[string]described {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(here), "..", "..", "..", "..", "packages", "api", "schemas", "frame.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Defs map[string]described `json:"$defs"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Defs
}

func keysOf(t reflect.Type) (all, required []string) {
	for f := range t.Fields() {
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		all = append(all, name)
		if !strings.Contains(opts, "omitempty") {
			required = append(required, name)
		}
	}
	slices.Sort(all)
	slices.Sort(required)
	return all, required
}

// invariant: packages/api/schemas/frame.json, which cmd/generate writes from
// its own copy of these shapes, says the keys the frame and its parts carry
// and which of them are always there.
func TestTheFrameSchemaIsTheFrame(t *testing.T) {
	defs := frameSchema(t)
	for name, typ := range map[string]reflect.Type{"Target": reflect.TypeFor[Target](), "Hello": reflect.TypeFor[Hello](), "Progress": reflect.TypeFor[Progress]()} {
		all, required := keysOf(typ)
		got := slices.Sorted(func(yield func(string) bool) {
			for k := range defs[name].Properties {
				if !yield(k) {
					return
				}
			}
		})
		if !slices.Equal(got, all) || !slices.Equal(slices.Sorted(slices.Values(defs[name].Required)), required) {
			t.Errorf("frame.json's %s has the keys %v (required %v), %v has %v (required %v): run make generate", name, got, defs[name].Required, typ, all, required)
		}
	}
	all, required := keysOf(reflect.TypeFor[Frame]())
	var carried []string
	for name, d := range defs {
		if !strings.HasSuffix(name, "Frame") {
			continue
		}
		for k := range d.Properties {
			carried = append(carried, k)
		}
		for _, k := range d.Required {
			if !slices.Contains(all, k) {
				t.Errorf("frame.json's %s requires %s, a key Frame does not carry", name, k)
			}
		}
		for _, k := range required {
			if !slices.Contains(d.Required, k) {
				t.Errorf("frame.json's %s does not require %s, which every Frame carries", name, k)
			}
		}
	}
	slices.Sort(carried)
	if carried = slices.Compact(carried); !slices.Equal(carried, all) {
		t.Errorf("frame.json's frames carry %v, Frame carries %v", carried, all)
	}
}
