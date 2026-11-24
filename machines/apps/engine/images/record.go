package images

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type Ref struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (r Ref) String() string { return r.Name + "@" + r.Version }

type Record struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// value: the distro the image is, which every record names, and its
	// version, empty while the image's Machinefile says none.
	Distro        catalog.Distro `json:"distro"`
	DistroVersion string         `json:"distro_version,omitempty"`
	Upstream      string         `json:"upstream"`
	Station       string         `json:"station"`
	RecipeSHA     string         `json:"recipe_sha"`
	// value: the Machinefile the image was made from, the hash of its
	// chain and lock as the build read them, and the lock the build VM
	// wrote once its image phase ran: what the image carries.
	Machinefile    string    `json:"machinefile,omitempty"`
	MachinefileSHA string    `json:"machinefile_sha,omitempty"`
	Lock           string    `json:"lock,omitempty"`
	BuiltAt        time.Time `json:"built_at"`
	SizeGB         int       `json:"size_gb"`
	// value: fetched from the registry that releases the image, not built
	// here.
	Fetched bool `json:"fetched,omitempty"`
}

func (r Record) Ref() Ref { return Ref{Name: r.Name, Version: r.Version} }

type Report struct {
	Record Record
	Newest bool
	VMs    []string
	// value: the Mac that holds this version.
	Mac string
	// value: whether the gesture Delete is offered: it is Unused, the one rule.
	Deletable bool
	// value: the image's Machinefile is gone from every source and the
	// catalog: the version is a record and a disk nothing can build again,
	// so it is never the newest, and Delete is offered once no VM uses it.
	Orphan bool
	// value: the build that is making this version, nil once it is built.
	Building *Building
}

func (r Report) Unused() bool {
	// invariant: an image nothing uses is not the newest of its name, no
	// machine was made from it and no build is making it.
	return !r.Newest && len(r.VMs) == 0 && r.Building == nil
}

func (r Report) Judged() Report {
	r.Deletable = r.Unused()
	return r
}

type Update struct {
	Ref Ref
	// value: the version was already the newest here: nothing was fetched.
	Newest bool
}

func CompareVersions(a, b string) int {
	// invariant: versions are dotted numbers (2026.11.01, 2026.11.01.2); a
	// missing part counts as 1, so a day's first build sorts before its second.
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := range max(len(pa), len(pb)) {
		if c := part(pa, i) - part(pb, i); c != 0 {
			return c
		}
	}
	return 0
}

func part(parts []string, i int) int {
	if i >= len(parts) {
		return 1
	}
	n, err := strconv.Atoi(parts[i])
	if err != nil {
		return 0
	}
	return n
}

type UnknownError struct{ Image string }

func (e *UnknownError) Error() string { return "no image \"" + e.Image + "\"" }

// value: a Machinefile no image is made from: a mixin, a computer's, or a
// VM's that says no vm.image.from.

type NotTemplateError struct{ Image string }

func (e *NotTemplateError) Error() string {
	return e.Image + " makes no image: its Machinefile says no vm.image.from"
}
