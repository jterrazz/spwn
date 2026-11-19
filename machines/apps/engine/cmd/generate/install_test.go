package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
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
