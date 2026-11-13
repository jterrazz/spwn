package catalog_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestTheBuiltInLinuxImageIsNamedStationLinuxAlone(t *testing.T) {
	c, err := catalog.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	if img, ok := c.Image("station-linux"); !ok || img.Machinefile != "station-linux" {
		t.Fatalf("station-linux: %+v %v, want the built-in Linux image", img, ok)
	}
	if img, ok := c.Image("linux"); ok {
		t.Errorf("linux names %s, want no image", img.Name)
	}
}
