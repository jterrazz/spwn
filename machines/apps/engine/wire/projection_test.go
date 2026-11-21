package wire

import (
	"encoding/json"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine/store"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func TestAPageIsServedInItsDocumentTheMenuBarDecodes(t *testing.T) {
	served := projection(func(*store.Snapshot, string) page.Page { return page.Page{ID: "menubar"} })(&store.Snapshot{}, "")
	raw, err := json.Marshal(served)
	if err != nil {
		t.Fatal(err)
	}
	got, err := page.Decode(raw)
	if err != nil || got.ID != "menubar" {
		t.Fatalf("page.Decode(%s) = %v, %v", raw, got.ID, err)
	}
}
