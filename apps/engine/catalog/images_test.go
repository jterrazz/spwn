package catalog_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestTheBuiltInImagesAreNamedBaseUbuntuAndBaseMacOSAlone(t *testing.T) {
	c, err := catalog.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, kind := range map[string]catalog.MachineKind{"base-ubuntu": catalog.Linux, "base-macos": catalog.MacOS} {
		if img, ok := c.Image(name); !ok || img.Machinefile != name || img.Kind != kind {
			t.Errorf("%s: %+v %v, want the built-in %s image made from its own Machinefile", name, img, ok, kind)
		}
	}
	for _, old := range []string{"linux", "station-linux", "macos"} {
		if img, ok := c.Image(old); ok {
			t.Errorf("%s names %s, want no image", old, img.Name)
		}
	}
}
