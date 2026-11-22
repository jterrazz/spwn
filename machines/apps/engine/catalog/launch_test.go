package catalog_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestTheCatalogNamesTheBundleIdentifiersTheAppsShipWith(t *testing.T) {
	apps := filepath.Join("..", "..")
	var window struct{ Identifier, ProductName string }
	body, err := os.ReadFile(filepath.Join(apps, "desktop", "src-tauri", "tauri.conf.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(body, &window); err != nil {
		t.Fatal(err)
	}
	if window.Identifier != catalog.WindowBundle {
		t.Errorf("the window ships as %q, the catalog names %q", window.Identifier, catalog.WindowBundle)
	}
	if window.ProductName != catalog.WindowName {
		t.Errorf("the window is named %q, the catalog %q", window.ProductName, catalog.WindowName)
	}
	info, err := os.ReadFile(filepath.Join(apps, "menubar", "resources", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`<key>CFBundleIdentifier</key>\s*<string>([^<]+)</string>`).FindSubmatch(info)
	if len(m) < 2 || string(m[1]) != catalog.MenuBarBundle {
		t.Errorf("the menu bar ships as %q, the catalog names %q", m, catalog.MenuBarBundle)
	}
}
