package wire

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestEveryRouteTheOwnerCanNameIsOneStationVerifies(t *testing.T) {
	named := make([]string, len(catalog.Routes))
	for i, r := range catalog.Routes {
		named[i] = r.Name
	}
	if wired := (machine{}).routes().Names(); !slices.Equal(wired, named) {
		t.Fatalf("--via names %v, station verifies %v", named, wired)
	}
}
