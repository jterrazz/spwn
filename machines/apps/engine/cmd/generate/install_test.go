package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

func makeVariable(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root(t), "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	found := regexp.MustCompile(`(?m)^` + name + `\s*:=\s*(.+)$`).FindSubmatch(body)
	if found == nil {
		t.Fatalf("Makefile sets no %s", name)
	}
	return strings.TrimSpace(string(found[1]))
}

func TestTheMakefileInstallsWhereTheCatalogSaysStationIsInstalled(t *testing.T) {
	if got, want := makeVariable(t, "BIN_DIR"), "$(HOME)/"+strings.TrimPrefix(catalog.InstallDir, "~/"); got != want {
		t.Errorf("Makefile BIN_DIR = %q, catalog.InstallDir says %q", got, want)
	}
	if got := makeVariable(t, "BINARY"); got != catalog.StationBinary {
		t.Errorf("Makefile BINARY = %q, catalog.StationBinary says %q", got, catalog.StationBinary)
	}
}

func TestARecordTheEngineWritesMatchesItsSchema(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(apiFiles(root(t))[recordSchemaPath], &schema); err != nil {
		t.Fatal(err)
	}
	written, err := json.MarshalIndent(engine.Record{V: engine.RecordVersion, Executable: "/opt/bin/station", Version: "v0.21.0", At: time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(written, &record); err != nil {
		t.Fatal(err)
	}
	for _, problem := range validate(schema, schema, record, "record") {
		t.Error(problem)
	}
	record["v"] = float64(engine.RecordVersion + 1)
	if len(validate(schema, schema, record, "record")) == 0 {
		t.Error("a record of another v is taken for v1: the schema describes v1 alone")
	}
}
