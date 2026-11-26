package catalog_test

import (
	"regexp"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestEveryPaneACatalogEntryNamesIsOneTheWindowMayOpen(t *testing.T) {
	listed := map[string]bool{}
	for _, p := range catalog.Panes {
		listed[p.URL] = true
		if p.Object == "" {
			t.Errorf("pane %s has no words for the button that opens it", p.URL)
		}
	}
	for _, c := range catalog.CheckedSettings {
		if c.Pane != "" && !listed[c.Pane] {
			t.Errorf("%s names pane %s, which the window may not open", c.Name, c.Pane)
		}
	}
}

func TestAPaneIsAnIdOfSystemSettingsWithAtMostAnAnchor(t *testing.T) {
	shape := regexp.MustCompile(`^x-apple\.systempreferences:[A-Za-z0-9.-]+(\?[A-Za-z_]+)?$`)
	for _, p := range catalog.Panes {
		if !shape.MatchString(p.URL) || len(p.URL) > 256 {
			t.Errorf("pane %q is not a System Settings id with an optional anchor", p.URL)
		}
	}
}
