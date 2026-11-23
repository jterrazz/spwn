package catalog_test

import (
	"io/fs"
	"path"
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
	for name, want := range map[string][2]string{"base-ubuntu": {"ubuntu", "24.04"}, "base-debian": {"debian", "13"}, "base-macos": {"macos", "26.6.2"}} {
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

func TestAFetchOnlyBaseIsPinnedByDigest(t *testing.T) {
	c, err := catalog.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	fetched := 0
	for _, img := range c.Images {
		if !c.FetchedOnly(img.Name) {
			continue
		}
		fetched++
		file := path.Join("machines", img.Machinefile+catalog.MachinefileSuffix)
		body, err := fs.ReadFile(catalog.BuiltinFiles(), file)
		if err != nil {
			t.Fatal(err)
		}
		mf, err := catalog.DecodeMachinefile(file, body)
		if err != nil {
			t.Fatal(err)
		}
		if from := mf.VM.Image.From; !catalog.PinnedByDigest(from) {
			t.Errorf("%s is fetched from %q, want its upstream pinned by digest", img.Name, from)
		}
		if img.DistroVersion == "" {
			t.Errorf("%s says no system.version, want the pinned image's version", img.Name)
		}
	}
	if fetched == 0 {
		t.Error("no built-in image is fetched only, want base-macos")
	}
}

func TestADigestPinsAndATagDoesNot(t *testing.T) {
	digest := "sha256:65c3133a633fda8753cf1f71f2d55727792d534a2125b0032466563dcc8e58fd"
	for ref, want := range map[string]bool{
		"ghcr.io/cirruslabs/macos-tahoe-base@" + digest:                                                                    true,
		"ghcr.io/cirruslabs/macos-tahoe-base:latest":                                                                       false,
		"ghcr.io/cirruslabs/macos-tahoe-base@sha256:65c3":                                                                  false,
		"ghcr.io/cirruslabs/macos-tahoe-base@sha256:" + "65C3133A633FDA8753CF1F71F2D55727792D534A2125B0032466563DCC8E58FD": false,
	} {
		if got := catalog.PinnedByDigest(ref); got != want {
			t.Errorf("PinnedByDigest(%q) = %v, want %v", ref, got, want)
		}
	}
}
