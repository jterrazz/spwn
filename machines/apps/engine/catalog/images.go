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
	Name, Recipe, Bake, Cleanup, Firstboot string
	// value: the script a clone of an image station only fetches runs once,
	// as UpstreamLogin, before station's key is in it; it puts the key in and
	// leaves no password of the image's that logs in. Empty for an image
	// station builds, whose key is baked.
	FirstLogin  string
	Machinefile string
	Defaults    ImageDefaults
	Kind        MachineKind
	// value: the distro the image is, and its version as its Machinefile's
	// `system.version` says it; empty while that file says none.
	Distro        Distro
	DistroVersion string
}

// value: the distribution a base image is, one base per distro; a
// machine's `system.os` names the same systems, `any` beside them.

type Distro string

const (
	DistroUbuntu Distro = "ubuntu"
	DistroDebian Distro = "debian"
	DistroMacOS  Distro = "macos"
)

func (Distro) Enum() []string {
	return []string{string(DistroUbuntu), string(DistroDebian), string(DistroMacOS)}
}

func (d Distro) Known() bool { return slices.Contains(d.Enum(), string(d)) }

func (d Distro) Word() string {
	switch d {
	case DistroUbuntu:
		return "Ubuntu"
	case DistroDebian:
		return "Debian"
	case DistroMacOS:
		return "macOS"
	}
	return string(d)
}

//go:embed images
var Recipes embed.FS

type ImageLogin struct{ User, Password string }

// value: the account every Cirrus Labs tart image ships with, and its
// password, the default Cirrus Labs publishes for all of them: no secret.
var (
	UpstreamLogin = ImageLogin{User: "admin", Password: "admin"}
)

// value: the built-in image a workspace is made of when none is named,
// and its Machinefile's name; the built-in Debian and macOS images, and
// theirs.
const (
	LinuxImage  = "base-ubuntu"
	DebianImage = "base-debian"
	MacOSImage  = "base-macos"
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
		Distro:      DistroUbuntu,
	},
	{
		Name:        DebianImage,
		Machinefile: DebianImage,
		Recipe:      "images/linux.sh",
		Bake:        "images/linux-bake.sh",
		Cleanup:     "images/linux-cleanup.sh",
		Firstboot:   "images/linux-firstboot.sh",
		Kind:        Linux,
		Distro:      DistroDebian,
	},
	{
		Name:        MacOSImage,
		Machinefile: MacOSImage,
		FirstLogin:  "images/macos-firstlogin.sh",
		Kind:        MacOS,
		Distro:      DistroMacOS,
	},
}

func sizedImages(files fs.FS) ([]Image, error) {
	// invariant: a built-in image is sized by its Machinefile alone, its
	// vm's resources, and its distro's version is that file's.
	out := slices.Clone(builtinImages)
	for i, img := range out {
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
		if mf.Machine != nil && mf.Machine.System != nil {
			out[i].DistroVersion = mf.Machine.System.Version
		}
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
	// recipe, cleanup and first boot of the built-in image of its distro;
	// ubuntu and debian have one, and a template that says no os is
	// ubuntu's.
	distro := Distro(os)
	if os == "" {
		distro = DistroUbuntu
	}
	for _, img := range c.Images {
		if img.Recipe != "" && img.Kind == Linux && img.Distro == distro {
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

func PinnedByDigest(ref string) bool {
	// value: an image reference that names its content by a sha256 digest,
	// never by a tag that may move.
	_, digest, ok := strings.Cut(ref, "@sha256:")
	return ok && len(digest) == 64 && strings.Trim(digest, "0123456789abcdef") == ""
}
