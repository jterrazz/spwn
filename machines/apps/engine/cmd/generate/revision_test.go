package main

import (
	"maps"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

func TestAChangeAnOlderPeerCannotIgnoreRaisesTheStateRevision(t *testing.T) {
	recorded, ok := recordedRevision(root(t))
	if !ok {
		t.Fatalf("%s does not read: run make generate and commit the result", revisionPath)
	}
	if recorded.State != engine.StateRevision {
		t.Fatalf("%s records revision %d, the engine speaks %d: run make generate and commit the result", revisionPath, recorded.State, engine.StateRevision)
	}
	if moved := movedPins(recorded.Pins, pinsOf(root(t), wireDocs(root(t)))); len(moved) > 0 {
		t.Fatalf("this changes what an older peer understands: raise the revision (engine.StateRevision and the window's STATE_REVISION, then make generate), or show it is additive (a new field left optional, a new call or type), as docs/reference/api.md#what-raises-the-revision says:\n%s", strings.Join(moved, "\n"))
	}
}

func TestAPinMovesForWhatAnOlderPeerReadsAsThereAndOnlyForThat(t *testing.T) {
	base := map[string]any{
		"properties": map[string]any{
			"name":  map[string]any{"type": "string"},
			"state": map[string]any{"enum": []any{"on", "off"}},
		},
		"required": []any{"name"},
	}
	baseProps, _ := base["properties"].(map[string]any)
	with := func(edit func(props map[string]any, required []any) []any) map[string]any {
		props := maps.Clone(baseProps)
		return map[string]any{"properties": props, "required": edit(props, []any{"name"})}
	}

	cases := []struct {
		change string
		schema map[string]any
		moves  bool
	}{
		{"an optional field added", with(func(p map[string]any, r []any) []any { p["note"] = map[string]any{"type": "string"}; return r }), false},
		{"a required field added", with(func(p map[string]any, r []any) []any {
			p["pane"] = map[string]any{"type": "string"}
			return append(r, "pane")
		}), true},
		{"a required field made optional", with(func(_ map[string]any, _ []any) []any { return nil }), true},
		{"a member added", with(func(p map[string]any, r []any) []any {
			p["state"] = map[string]any{"enum": []any{"on", "off", "held"}}
			return r
		}), true},
		{"a required field retyped", with(func(p map[string]any, r []any) []any { p["name"] = map[string]any{"type": "integer"}; return r }), true},
	}

	for _, c := range cases {
		if moved := pinOf(c.schema) != pinOf(base); moved != c.moves {
			t.Errorf("%s moved the pin: %v, want %v", c.change, moved, c.moves)
		}
	}
}

func TestATypeOrCallFirstSeenAtARevisionIsPinnedWithoutRaisingIt(t *testing.T) {
	recorded := map[string]string{"type:a": "name!{}", "views": "machines.list"}
	current := map[string]string{"type:a": "name!{}", "type:b": "pane!{}", "views": "machines.list"}

	if moved := movedPins(recorded, current); len(moved) != 0 {
		t.Fatalf("a new type moved %v, want nothing", moved)
	}

	current["views"] = "engine.guides machines.list"
	if moved := movedPins(recorded, current); len(moved) != 1 || !strings.HasPrefix(moved[0], "views") {
		t.Fatalf("a view the window now asks moved %v, want the views", moved)
	}
}
