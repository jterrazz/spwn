package catalog_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func TestTheBuiltInImagesAreTheThreeBases(t *testing.T) {
	c, err := catalog.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, kind := range map[string]catalog.MachineKind{"base-ubuntu": catalog.Linux, "base-debian": catalog.Linux, "base-macos": catalog.MacOS} {
		if img, ok := c.Image(name); !ok || img.Machinefile != name || img.Kind != kind {
			t.Errorf("%s: %+v %v, want the built-in %s image made from its own Machinefile", name, img, ok, kind)
		}
	}
	for name, want := range map[string][2]string{"base-ubuntu": {"ubuntu", "24.04"}, "base-debian": {"debian", "13"}, "base-macos": {"macos", ""}} {
		img, _ := c.Image(name)
		if got := [2]string{string(img.Distro), img.DistroVersion}; got != want {
			t.Errorf("%s is %v, want the distro %v, its version its Machinefile's", name, got, want)
		}
	}
	for _, old := range []string{"linux", "station-linux", "macos"} {
		if img, ok := c.Image(old); ok {
			t.Errorf("%s names %s, want no image", old, img.Name)
		}
	}
}

func TestATemplateIsBuiltWithThePlumbingOfItsSystem(t *testing.T) {
	c, err := catalog.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	for os, want := range map[catalog.OS]string{catalog.OSUbuntu: "base-ubuntu", "": "base-ubuntu", catalog.OSDebian: "base-debian", catalog.OSMacOS: ""} {
		img, ok := c.RecipeFor(os)
		if img.Name != want || ok != (want != "") {
			t.Errorf("the recipe for %q is %q %v; want %q", os, img.Name, ok, want)
		}
	}
}
