package catalog_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestTheJobsNameTheBundleIdentifiersTheAppsShipWith(t *testing.T) {
	apps := filepath.Join("..", "..")
	var window struct{ Identifier string }
	body, err := os.ReadFile(filepath.Join(apps, "desktop", "src-tauri", "tauri.conf.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(body, &window); err != nil {
		t.Fatal(err)
	}
	if window.Identifier != catalog.WindowBundle {
		t.Errorf("the window ships as %q, the jobs name %q", window.Identifier, catalog.WindowBundle)
	}
	info, err := os.ReadFile(filepath.Join(apps, "menubar", "resources", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`<key>CFBundleIdentifier</key>\s*<string>([^<]+)</string>`).FindSubmatch(info)
	if len(m) < 2 || string(m[1]) != catalog.MenuBarBundle {
		t.Errorf("the menu bar ships as %q, the jobs name %q", m, catalog.MenuBarBundle)
	}
}
