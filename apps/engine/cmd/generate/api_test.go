package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestThePackagesAPISchemasAndTypesHaveNotDriftedFromTheirGoTypes(t *testing.T) {
	want := apiFiles(root(t))
	for path, body := range want {
		current, err := os.ReadFile(filepath.Join(root(t), path))
		if err != nil || !bytes.Equal(current, body) {
			t.Errorf("%s is out of sync with the calls' Go types: run make generate and commit the result", path)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root(t), apiCallsDir))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if _, ok := want[filepath.Join(apiCallsDir, e.Name())]; !ok {
			t.Errorf("%s/%s describes no call: delete it, then run make generate", apiCallsDir, e.Name())
		}
	}
}

func TestEveryCallOfTheWireHasItsSchema(t *testing.T) {
	var c struct {
		Calls []struct {
			Name string `json:"name"`
		} `json:"calls"`
	}
	data, err := os.ReadFile(filepath.Join(root(t), callsPath))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	files := apiFiles(root(t))
	for _, e := range c.Calls {
		if _, ok := files[filepath.Join(apiCallsDir, e.Name+".json")]; !ok {
			t.Errorf("%s: in calls.json, with no schema under %s", e.Name, apiCallsDir)
		}
	}
	if len(c.Calls) != len(wireCalls()) {
		t.Errorf("calls.json lists %d calls, the wire serves %d", len(c.Calls), len(wireCalls()))
	}
}

// invariant: what the engine really sent, recorded, is what the schemas
// say: each line of the recorded streams is an answer frame whose out is
// its watch's, and each page fixture is a page.Document.
func TestTheRecordedFixturesMatchTheSchemas(t *testing.T) {
	files := apiFiles(root(t))
	doc := func(path string) map[string]any {
		var d map[string]any
		if err := json.Unmarshal(files[path], &d); err != nil {
			t.Fatal(err)
		}
		return d
	}
	frame := doc(framePath)
	for path, watch := range map[string]string{streamPath: "page", eventsPath: "engine.events"} {
		call := doc(filepath.Join(apiCallsDir, watch+".json"))
		out := object(object(call["properties"])["out"])
		data, err := os.ReadFile(filepath.Join(root(t), path))
		if err != nil {
			t.Fatal(err)
		}
		lines := bufio.NewScanner(bytes.NewReader(data))
		lines.Buffer(nil, 64<<20)
		n := 0
		for lines.Scan() {
			n++
			var v map[string]any
			if err := json.Unmarshal(lines.Bytes(), &v); err != nil {
				t.Fatalf("%s:%d: %v", path, n, err)
			}
			for _, e := range validate(frame, frame, v, "frame") {
				t.Errorf("%s:%d: %s", path, n, e)
			}
			for _, e := range validate(call, out, v["out"], "out") {
				t.Errorf("%s:%d: %s", path, n, e)
			}
		}
		if n == 0 {
			t.Errorf("%s: no frame recorded: the rule reads nothing", path)
		}
	}
	watch := doc(filepath.Join(apiCallsDir, "page.json"))
	pageRef := map[string]any{"$ref": "#/$defs/page.Document"}
	read := 0
	for _, dir := range []string{pagesDir, storiesDir, "apps/engine/ui/testdata/__expected__"} {
		entries, err := os.ReadDir(filepath.Join(root(t), dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(root(t), dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			var v any
			if err := json.Unmarshal(data, &v); err != nil {
				t.Fatalf("%s/%s: %v", dir, e.Name(), err)
			}
			read++
			for _, problem := range validate(watch, pageRef, v, "page") {
				t.Errorf("%s/%s: %s", dir, e.Name(), problem)
			}
		}
	}
	if read == 0 {
		t.Error("no page fixture read: the rule reads nothing")
	}
}

// value: the part of JSON Schema the generator writes: $ref into the
// document's $defs, anyOf, const, enum, type, properties, required,
// additionalProperties, items, minItems and maxItems.
func validate(doc, s map[string]any, v any, at string) []string {
	if ref, ok := s["$ref"].(string); ok {
		def, ok := doc["$defs"].(map[string]any)[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any)
		if !ok {
			return []string{at + ": no definition " + ref}
		}
		return validate(doc, def, v, at)
	}
	if alts, ok := s["anyOf"].([]any); ok {
		var all []string
		for _, a := range alts {
			problems := validate(doc, object(a), v, at)
			if len(problems) == 0 {
				return nil
			}
			all = append(all, problems...)
		}
		return []string{fmt.Sprintf("%s: matches none of its %d forms (%s)", at, len(alts), strings.Join(all, "; "))}
	}
	if c, ok := s["const"]; ok && !reflect.DeepEqual(canonical(c), v) {
		return []string{fmt.Sprintf("%s: %v is not %v", at, v, c)}
	}
	if enum, ok := s["enum"]; ok && !slices.ContainsFunc(list(canonical(enum)), func(e any) bool { return reflect.DeepEqual(e, v) }) {
		return []string{fmt.Sprintf("%s: %v is none of %v", at, v, enum)}
	}
	if types := typesOf(s["type"]); len(types) > 0 && !slices.Contains(types, jsonType(v)) && (jsonType(v) != "integer" || !slices.Contains(types, "number")) {
		return []string{fmt.Sprintf("%s: a %s, not %v", at, jsonType(v), types)}
	}
	var problems []string
	switch v := v.(type) {
	case map[string]any:
		props, _ := s["properties"].(map[string]any)
		for _, k := range typesOf(s["required"]) {
			if _, ok := v[k]; !ok {
				problems = append(problems, at+": no "+k)
			}
		}
		for k, item := range v {
			if p, ok := props[k].(map[string]any); ok {
				problems = append(problems, validate(doc, p, item, at+"."+k)...)
				continue
			}
			switch extra := s["additionalProperties"].(type) {
			case bool:
				if !extra {
					problems = append(problems, at+": the key "+k+" is not in its schema")
				}
			case map[string]any:
				problems = append(problems, validate(doc, extra, item, at+"."+k)...)
			}
		}
	case []any:
		if items, ok := s["items"].(map[string]any); ok {
			for i, item := range v {
				problems = append(problems, validate(doc, items, item, fmt.Sprintf("%s[%d]", at, i))...)
			}
		}
		if n, ok := s["minItems"].(float64); ok && float64(len(v)) < n {
			problems = append(problems, fmt.Sprintf("%s: %d items, fewer than %v", at, len(v), n))
		}
		if n, ok := s["maxItems"].(float64); ok && float64(len(v)) > n {
			problems = append(problems, fmt.Sprintf("%s: %d items, more than %v", at, len(v), n))
		}
	}
	return problems
}

func typesOf(v any) []string {
	switch v := v.(type) {
	case string:
		return []string{v}
	case []any:
		var out []string
		for _, s := range v {
			if name, ok := s.(string); ok {
				out = append(out, name)
			}
		}
		return out
	}
	return nil
}

func jsonType(v any) string {
	switch v := v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		if v == math.Trunc(v) {
			return "integer"
		}
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	}
	return "object"
}
