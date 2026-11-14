package catalog

import (
	"embed"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
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
	Defaults                                     ImageDefaults
	Kind                                         MachineKind
}

//go:embed images
var Recipes embed.FS

// invariant: memory counts in binary gigabytes, as macOS and tart count RAM; a
// disk in decimal ones, as tart's --disk-size does.
const (
	gigabyte = 1000 * 1000 * 1000
	gibibyte = 1 << 30
)

// value: the built-in image a workspace is made of when none is named,
// and its Machinefile's name.
const (
	LinuxImage = "station-linux"
)

// value: the images station knows; an image is no package of the catalog's
// files, and reaches its readers through Catalog.Images.
var builtinImages = []Image{
	{
		Name:        LinuxImage,
		Machinefile: LinuxImage,
		Recipe:      "images/linux.sh",
		Bake:        "images/linux-bake.sh",
		Cleanup:     "images/linux-cleanup.sh",
		Firstboot:   "images/linux-firstboot.sh",
		Kind:        Linux,
	},
	{
		Name:     "macos",
		Defaults: ImageDefaults{CPUs: 4, MemoryBytes: 8 * gibibyte, DiskBytes: 80 * gigabyte},
		Kind:     MacOS,
	},
}

func sizedImages(files fs.FS) ([]Image, error) {
	// invariant: a built-in image made from a Machinefile is sized by that
	// file alone, its vm's resources; an image only fetched carries its own.
	out := slices.Clone(builtinImages)
	for i, img := range out {
		if img.Machinefile == "" {
			continue
		}
		file := path.Join("machines", img.Machinefile+MachinefileSuffix)
		body, err := fs.ReadFile(files, file)
		if err != nil {
			return nil, err
		}
		mf, err := DecodeMachinefile(file, body)
		if err != nil {
			return nil, err
		}
		if mf.VM == nil || mf.VM.Resources == nil {
			return nil, &FileError{File: file, Problem: "says no vm resources: a built-in image is sized by its Machinefile"}
		}
		r := mf.VM.Resources
		out[i].Defaults = ImageDefaults{CPUs: r.CPUs, MemoryBytes: r.Memory.bytes(1024), DiskBytes: r.Disk.bytes(1000)}
	}
	return out, nil
}

// value: the bytes of a size its file's check let through, memory counted
// in base 1024 and a disk in base 1000; none is zero.
func (s Size) bytes(base int64) int64 {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(string(s[:len(s)-1]), 10, 64)
	if err != nil {
		return 0
	}
	for range strings.IndexByte("KMGT", s[len(s)-1]) + 1 {
		n *= base
	}
	return n
}

func (c Catalog) Image(name string) (Image, bool) {
	for _, img := range c.Images {
		if img.Name == name {
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
