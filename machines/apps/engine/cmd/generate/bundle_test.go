package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"os"
	"path"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: a plist's top-level keys, a string's value or an array's strings.
func plistValues(t *testing.T, body []byte) map[string][]string {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(body))
	values, key, depth := map[string][]string{}, "", 0
	for {
		tok, err := d.Token()
		if err != nil {
			return values
		}
		switch e := tok.(type) {
		case xml.StartElement:
			depth++
			if e.Name.Local != "key" && e.Name.Local != "string" {
				continue
			}
			var text string
			if err := d.DecodeElement(&text, &e); err != nil {
				t.Fatal(err)
			}
			depth--
			if e.Name.Local == "key" && depth == 2 {
				key = text
			} else if e.Name.Local == "string" {
				values[key] = append(values[key], text)
			}
		case xml.EndElement:
			depth--
		}
	}
}

func TestTheBundledJobRunsTheBundlesEngineUnderTheCatalogsLabel(t *testing.T) {
	for label, args := range map[string][]string{catalog.BundledEngineLabel: catalog.EngineArgs} {
		body, err := os.ReadFile(filepath.Join(root(t), bundleAgentsDir, path.Base(catalog.BundledPlist(label))))
		if err != nil {
			t.Fatal(err)
		}
		values := plistValues(t, body)
		if got := values["Label"]; !slices.Equal(got, []string{label}) {
			t.Errorf("%s: Label = %q", label, got)
		}
		if got := values["BundleProgram"]; !slices.Equal(got, []string{catalog.BundledEnginePath}) {
			t.Errorf("%s: BundleProgram = %q, want %q", label, got, catalog.BundledEnginePath)
		}
		if got, want := values["ProgramArguments"], append([]string{catalog.BundledEngine}, args...); !slices.Equal(got, want) {
			t.Errorf("%s: ProgramArguments = %q, want %q", label, got, want)
		}
		for _, absent := range []string{"EnvironmentVariables", "StandardOutPath", "AssociatedBundleIdentifiers"} {
			if _, ok := values[absent]; ok || bytes.Contains(body, []byte("<key>"+absent+"</key>")) {
				t.Errorf("%s: a sealed bundle's job names %s", label, absent)
			}
		}
	}
}

func TestTheWindowsBundleHoldsTheEngineItsOneJobAndTheMenuBarWhereTheCatalogSays(t *testing.T) {
	body, err := os.ReadFile(filepath.Join(root(t), "apps", "desktop", "src-tauri", "tauri.conf.json"))
	if err != nil {
		t.Fatal(err)
	}
	var conf struct {
		Identifier string `json:"identifier"`
		Bundle     struct {
			MacOS struct {
				Files map[string]string `json:"files"`
			} `json:"macOS"`
		} `json:"bundle"`
	}
	if err := json.Unmarshal(body, &conf); err != nil {
		t.Fatal(err)
	}
	if conf.Identifier != catalog.WindowBundle {
		t.Errorf("tauri.conf.json identifier = %q, catalog.WindowBundle says %q", conf.Identifier, catalog.WindowBundle)
	}
	contents := func(p string) string { return path.Clean(p[len("Contents/"):]) }
	want := map[string]string{
		contents(catalog.BundledEnginePath):  "../../../.artifacts/window/engine/station",
		contents(catalog.BundledMenuBarPath): "../../../.artifacts/menubar/Station.app",
	}
	want[contents(catalog.BundledPlist(catalog.BundledEngineLabel))] = path.Join(path.Base(bundleAgentsDir), path.Base(catalog.BundledPlist(catalog.BundledEngineLabel)))
	if got := conf.Bundle.MacOS.Files; len(got) != len(want) {
		t.Errorf("bundle.macOS.files = %q, want %q", got, want)
	}
	for to, from := range want {
		if got := conf.Bundle.MacOS.Files[to]; got != from {
			t.Errorf("bundle.macOS.files[%q] = %q, want %q", to, got, from)
		}
	}
	if got, want := makeVariable(t, "WINDOW_ENGINE"), "$(WINDOW_APP)/"+catalog.BundledEnginePath; got != want {
		t.Errorf("Makefile WINDOW_ENGINE = %q, want %q", got, want)
	}
	if got, want := makeVariable(t, "WINDOW_MENUBAR"), "$(WINDOW_APP)/"+catalog.BundledMenuBarPath; got != want {
		t.Errorf("Makefile WINDOW_MENUBAR = %q, want %q", got, want)
	}
}
