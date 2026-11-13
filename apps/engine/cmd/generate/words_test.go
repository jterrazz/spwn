package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

func TestEveryValueOfAWordedEnumHasItsWordAndATone(t *testing.T) {
	tones := verdict.Tone("").Enum()
	for _, w := range wordedEnums() {
		name := wordsName(w.of)
		values := w.of.Enum()
		for _, v := range values {
			word, ok := w.words[v]
			switch {
			case !ok:
				t.Errorf("%s: %q has no word", name, v)
			case word.Word == "":
				t.Errorf("%s: %q has an empty word", name, v)
			case !slices.Contains(tones, string(word.Tone)):
				t.Errorf("%s: %q has the tone %q, not one of %v", name, v, word.Tone, tones)
			}
		}
		for v := range w.words {
			if !slices.Contains(values, v) {
				t.Errorf("%s: %q is worded, but is no value of the enum", name, v)
			}
		}
	}
}

func TestEveryWordedEnumIsNamedOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, name := range wordsNames() {
		if seen[name] {
			t.Errorf("%s is named twice", name)
		}
		seen[name] = true
	}
}

func TestNoWordedEnumAdmitsAnEmptyValueOnTheWire(t *testing.T) {
	r := &reflector{}
	for _, w := range wordedEnums() {
		got := r.zeroable(reflect.TypeOf(w.of), map[string]any{"enum": w.of.Enum()})
		if values, _ := got["enum"].([]string); slices.Contains(values, "") {
			t.Errorf("%s admits \"\" on the wire: %v", wordsName(w.of), values)
		}
	}
}

func TestAMachinesKindIsClosedInTheSchemasTheWindowReads(t *testing.T) {
	kinds := catalog.MachineKind("").Enum()
	files := apiFiles(root(t))
	for file, at := range map[string]string{
		"machines.list.json": "machines.VM",
		"overview.read.json": "overview.Machine",
	} {
		var doc struct {
			Defs map[string]struct {
				Properties map[string]struct {
					Enum []string `json:"enum"`
				} `json:"properties"`
			} `json:"$defs"`
		}
		if err := json.Unmarshal(files[filepath.Join(apiCallsDir, file)], &doc); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		got := doc.Defs[at].Properties["Kind"].Enum
		if !slices.Equal(got, kinds) {
			t.Errorf("%s: %s.Kind is [%s]; want exactly [%s], with no \"\"", file, at, strings.Join(got, ", "), strings.Join(kinds, ", "))
		}
	}
}
