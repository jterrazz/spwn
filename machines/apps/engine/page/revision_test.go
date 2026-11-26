package page

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

func TestTheMenuBarDrawsTheRevisionOfItsTreesEngine(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "menubar", "sources", "station-kit", "page", "Page.swift"))
	if err != nil {
		t.Fatal(err)
	}
	found := regexp.MustCompile(`public let pageRevision = (\d+)`).FindSubmatch(source)
	if found == nil {
		t.Fatal("Page.swift declares no pageRevision")
	}
	if drawn, _ := strconv.Atoi(string(found[1])); drawn != Revision {
		t.Fatalf("the menu bar draws station.page/v1 revision %d, this engine speaks %d: raise both together", drawn, Revision)
	}
}
