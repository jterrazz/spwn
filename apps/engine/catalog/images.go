package catalog

import (
	"embed"
	"slices"
)

type ImageDefaults struct {
	CPUs                   int
	MemoryBytes, DiskBytes int64
}

type Image struct {
	// value: Recipe, Cleanup and Firstboot empty: station builds no such
	// image; it is only fetched where it is published. Machinefile, the
	// built-in template it is made from, its name the image's.
	Name, Base, Recipe, Bake, Cleanup, Firstboot string
	Machinefile                                  string
	// value: the names the image was known by, read as its own for one
	// release.
	RenamedFrom []string
	Defaults    ImageDefaults
	Kind        MachineKind
}

//go:embed images
var Recipes embed.FS

// invariant: memory counts in binary gigabytes, as macOS and tart count RAM; a
// disk in decimal ones, as tart's --disk-size does.
const (
	gigabyte = 1000 * 1000 * 1000
	gibibyte = 1 << 30
)

// value: the images station knows; an image is no package of the catalog's
// files, and reaches its readers through Catalog.Images.
var builtinImages = []Image{
	{
		Name:        "station-linux",
		Machinefile: "station-linux",
		RenamedFrom: []string{"linux"},
		Recipe:      "images/linux.sh",
		Bake:        "images/linux-bake.sh",
		Cleanup:     "images/linux-cleanup.sh",
		Firstboot:   "images/linux-firstboot.sh",
		Defaults:    ImageDefaults{CPUs: 4, MemoryBytes: 4 * gibibyte, DiskBytes: 20 * gigabyte},
		Kind:        Linux,
	},
	{
		Name:     "macos",
		Defaults: ImageDefaults{CPUs: 4, MemoryBytes: 8 * gibibyte, DiskBytes: 80 * gigabyte},
		Kind:     MacOS,
	},
}

func (c Catalog) Image(name string) (Image, bool) {
	// value: the image a name stands for, by its name or one it was renamed
	// from.
	for _, img := range c.Images {
		if img.Name == name || slices.Contains(img.RenamedFrom, name) {
			return img, true
		}
	}
	return Image{}, false
}

func (c Catalog) RecipeFor(os OS) (Image, bool) {
	// value: the base plumbing a template of this system is built with: the
	// recipe, cleanup and first boot of the built-in image of its kind; only
	// ubuntu has one (a template that says no os is the base's).
	for _, img := range c.Images {
		if img.Recipe != "" && img.Kind == Linux && (os == OSUbuntu || os == "") {
			return img, true
		}
	}
	return Image{}, false
}

func (c Catalog) FetchedOnly(name string) bool {
	for _, img := range c.Images {
		if img.Name == name {
			return img.Recipe == ""
		}
	}
	return false
}
