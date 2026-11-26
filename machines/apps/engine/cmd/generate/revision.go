package main

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

const (
	revisionPath = "packages/api/revision.json"
	viewsSource  = "apps/desktop/src/domain/views.ts"
)

// value: revision.json: the state revision both peers speak, and what an
// older peer of it cannot ignore, each pin as it was first generated at
// that revision.
type revisionRecord struct {
	State int               `json:"state"`
	Pins  map[string]string `json:"pins"`
}

var viewKeys = regexp.MustCompile(`(?s)export const VIEW_KEYS = \[(.*?)\] as const`)

// value: the views the window asks in its one engine.state request, read
// from its source, since an engine refuses the whole request for one view
// it does not know.
func windowViews(root string) []string {
	source, err := os.ReadFile(filepath.Join(root, viewsSource))
	if err != nil {
		panic(err)
	}
	block := viewKeys.FindSubmatch(source)
	if block == nil {
		panic(viewsSource + " declares no VIEW_KEYS")
	}
	all := regexp.MustCompile(`'([^']+)'`).FindAllSubmatch(block[1], -1)
	out := make([]string, 0, len(all))
	for _, quoted := range all {
		out = append(out, string(quoted[1]))
	}
	slices.Sort(out)
	return out
}

func normalized(doc any) map[string]any {
	body, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		panic(err)
	}
	return out
}

func compact(v any) string {
	body, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(body)
}

func holdsEnum(v any) bool {
	switch s := v.(type) {
	case map[string]any:
		if _, ok := s["enum"]; ok {
			return true
		}
		return slices.ContainsFunc(slices.Collect(maps.Values(s)), holdsEnum)
	case []any:
		return slices.ContainsFunc(s, holdsEnum)
	}
	return false
}

// value: the part of a schema an older peer cannot ignore: every required
// property and its shape, and every optional one whose shape holds an
// enum, since a member it does not know is no absence; an optional field
// with no enum is left out, so adding one pins nothing.
func pinOf(schema map[string]any) string {
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return compact(schema)
	}
	required := map[string]bool{}
	if list, ok := schema["required"].([]any); ok {
		for _, r := range list {
			if name, ok := r.(string); ok {
				required[name] = true
			}
		}
	}
	var parts []string
	for _, name := range slices.Sorted(maps.Keys(props)) {
		switch {
		case required[name]:
			parts = append(parts, name+"!"+compact(props[name]))
		case holdsEnum(props[name]):
			parts = append(parts, name+"?"+compact(props[name]))
		}
	}
	return strings.Join(parts, " ")
}

func defsPins(pins map[string]string, doc map[string]any) {
	defs, _ := doc["$defs"].(map[string]any)
	for name, def := range defs {
		if schema, ok := def.(map[string]any); ok {
			pins["type:"+name] = pinOf(schema)
		}
	}
}

// value: every pin of this tree: the window's views, each confirmed call's
// forms, each call's in and out, and every type of the calls, the frame
// and the events.
func pinsOf(root string, d wireDocuments) map[string]string {
	pins := map[string]string{"views": strings.Join(windowViews(root), " ")}
	for name, c := range confirmsTable() {
		forms := make([]string, 0, 1+len(c.Kinds))
		forms = append(forms, c.Form)
		for _, kind := range slices.Sorted(maps.Keys(c.Kinds)) {
			forms = append(forms, kind+"="+c.Kinds[kind].Form)
		}
		pins["confirm:"+name] = strings.Join(forms, " ")
	}
	for name, doc := range d.docs {
		n := normalized(doc)
		pins["call:"+name] = pinOf(n)
		defsPins(pins, n)
	}
	defsPins(pins, normalized(d.frame))
	defsPins(pins, normalized(d.event))
	return pins
}

func recordedRevision(root string) (revisionRecord, bool) {
	body, err := os.ReadFile(filepath.Join(root, revisionPath))
	if err != nil {
		return revisionRecord{}, false
	}
	var r revisionRecord
	if json.Unmarshal(body, &r) != nil {
		return revisionRecord{}, false
	}
	return r, true
}

// invariant: a pin recorded at the revision the engine still speaks keeps
// its recorded value, so a change an older peer cannot ignore is never
// absorbed by a regeneration: the revision test names it until the
// revision is raised, which records every pin anew.
func revisionFile(root string, pins map[string]string) revisionRecord {
	out := revisionRecord{State: engine.StateRevision, Pins: maps.Clone(pins)}
	recorded, ok := recordedRevision(root)
	if !ok || recorded.State != engine.StateRevision {
		return out
	}
	for key := range out.Pins {
		if was, ok := recorded.Pins[key]; ok {
			out.Pins[key] = was
		}
	}
	return out
}

// value: each pin recorded at this revision whose value moved, said as
// what it was and what it is.
func movedPins(recorded, current map[string]string) []string {
	var moved []string
	for _, key := range slices.Sorted(maps.Keys(recorded)) {
		if now, ok := current[key]; ok && now != recorded[key] {
			moved = append(moved, key+"\n  was "+recorded[key]+"\n  is  "+now)
		}
	}
	return moved
}
