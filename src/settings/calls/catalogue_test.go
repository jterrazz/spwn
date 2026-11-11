package calls

import (
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
)

func TestAnEffectNobodyHandlesIsAnErrorNotASilentSuccess(t *testing.T) {
	svc := NewCatalogue(nil, CatalogueOptions{ThisMachine: catalog.Mac})
	if _, err := svc.handler(catalog.Setting{Name: "x", Effect: catalog.Procedure("nope")}); err == nil {
		t.Fatal("an unknown procedure dispatched")
	}
	if _, err := svc.handler(catalog.Setting{Name: "y"}); err == nil {
		t.Fatal("an item with no effect dispatched")
	}
}
