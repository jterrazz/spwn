package catalog_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
)

func TestEveryAppOfTheCatalogueReadsBackAsItWasSent(t *testing.T) {
	c, err := catalog.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range c.Apps {
		raw, err := json.Marshal(tool)
		if err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
		var back catalog.App
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
		if !reflect.DeepEqual(back, tool) {
			t.Errorf("%s came back as %+v, want %+v", tool.Name, back.Installs, tool.Installs)
		}
	}
}
