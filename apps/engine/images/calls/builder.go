package calls

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/images"

	imagescontract "github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type BuilderOptions struct {
	// value: where the release binary is cached, and the build key.
	Cache, Key string
	Version    string
	User       string
	// value: the Mac this service holds the images of.
	Mac string
}

type Builder struct {
	m         Machine
	e         contract.Hypervisor
	builds    BuildVMs
	store     imagescontract.ImageStore
	source    imagescontract.ImageSource
	vms       contract.MachineStore
	templates Templates
	cat       catalog.Catalog
	o         BuilderOptions
}

type Contracts struct {
	Hypervisor contract.Hypervisor
	Builds     BuildVMs
	Store      imagescontract.ImageStore
	Source     imagescontract.ImageSource
	Machines   contract.MachineStore
	Templates  Templates
}

type Templates interface {
	// value: the Machinefile an image is made from, read from every source
	// as plan reads it: images.UnknownError when no source holds it,
	// images.NotTemplateError when it makes no image.
	Template(ctx context.Context, name string) (images.Template, error)
	// value: every Machinefile an image is made from, by name.
	Names(ctx context.Context) ([]string, error)
}

type BuildVMs interface {
	// value: the engine's build VMs: it boots a clone and holds the guest a
	// build speaks to; images/calls never reaches a VM but through it.
	Boot(ctx context.Context, vm string) error
	// value: the booted VM, its guest reached at the address it leased within
	// wait.
	Open(ctx context.Context, vm string, wait time.Duration) (BuildVM, error)
}

type BuildVM interface {
	Put(ctx context.Context, as Login, path string, data []byte, mode uint32) error
	Run(ctx context.Context, as Login, line, stdin string) (Ran, error)
	// invariant: lets go of every session the engine held for the build.
	Close() error
}

type Login struct {
	// value: who a build speaks to its VM as: the base's admin by its
	// password, or the VM's user by the image key.
	User, Key, Password string
}

type Ran struct {
	Code           int
	Stdout, Stderr string
}

func NewBuilder(m Machine, c Contracts, cat catalog.Catalog, o BuilderOptions) *Builder {
	return &Builder{m: m, e: c.Hypervisor, builds: c.Builds, store: c.Store, source: c.Source, vms: c.Machines, templates: c.Templates, cat: cat, o: o}
}

// invariant: an image is one of the catalog's, by its name or one it was
// renamed from, or a template's of a source, named after its
// Machinefile and built with the plumbing of its system.
func (s *Builder) image(ctx context.Context, name string) (catalog.Image, error) {
	if img, ok := s.cat.Image(name); ok {
		return img, nil
	}
	t, err := s.templates.Template(ctx, name)
	if err != nil {
		return catalog.Image{}, err
	}
	img, ok := s.cat.RecipeFor(t.OS)
	if !ok {
		return catalog.Image{Name: name}, &images.NoRecipeError{Image: name}
	}
	img.Name, img.Machinefile, img.RenamedFrom = name, name, nil
	return img, nil
}

// value: every image this Mac may hold: the catalog's, then each
// template a source holds that has a recipe, in name order.
func (s *Builder) all(ctx context.Context) ([]catalog.Image, error) {
	out := slices.Clone(s.cat.Images)
	names, err := s.templates.Names(ctx)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if _, ok := s.cat.Image(name); ok {
			continue
		}
		if img, err := s.image(ctx, name); err == nil {
			out = append(out, img)
		}
	}
	return out, nil
}

func (s *Builder) List(ctx context.Context) ([]images.Report, error) {
	sizes := map[string]int{}
	if disks, err := s.e.Disks(ctx); err == nil {
		for _, d := range disks {
			sizes[d.VM] = int(d.Bytes / 1e9)
		}
	}
	users, err := s.users()
	if err != nil {
		return nil, err
	}
	list, err := s.all(ctx)
	if err != nil {
		return nil, err
	}
	var out []images.Report
	for _, img := range list {
		held, err := s.records(img)
		if err != nil {
			return nil, err
		}
		for i, r := range held {
			if size, ok := sizes[DiskName(r.Name, r.Version)]; ok {
				r.SizeGB = size
			}
			vms := users[r.Name+"@"+r.Version]
			r.Name = img.Name
			out = append(out, images.Report{Record: r, Newest: i == 0, VMs: vms, Mac: s.o.Mac})
		}
	}
	return out, nil
}

func (s *Builder) users() (map[string][]string, error) {
	records, err := s.vms.ListReadable(context.Background())
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, rec := range records {
		key := rec.Image + "@" + rec.Version
		out[key] = append(out[key], rec.Name)
	}
	for _, names := range out {
		slices.Sort(names)
	}
	return out, nil
}

func (s *Builder) CheckDelete(ctx context.Context, image, version string) error {
	img, err := s.image(ctx, image)
	var none *images.NoRecipeError
	if err != nil && !errors.As(err, &none) {
		return err
	}
	all, err := s.records(img)
	if err != nil {
		return err
	}
	if len(all) == 0 {
		return &images.NotBuiltError{Image: image}
	}
	if version != "" && !slices.ContainsFunc(all, func(r images.Record) bool { return r.Version == version }) {
		return &images.UnknownVersionError{Image: image, Version: version}
	}
	return nil
}

func (s *Builder) Delete(ctx context.Context, image, version string) ([]images.Record, error) {
	if err := s.CheckDelete(ctx, image, version); err != nil {
		return nil, err
	}
	if _, err := s.e.Version(ctx); err != nil {
		return nil, &images.NoEngineError{Hypervisor: s.e.Name(), Cause: err}
	}
	img, _ := s.image(ctx, image)
	all, err := s.records(img)
	if err != nil {
		return nil, err
	}
	present, err := s.e.Disks(ctx)
	if err != nil {
		return nil, err
	}
	var removed []images.Record
	for _, r := range all {
		if version != "" && r.Version != version {
			continue
		}
		name := DiskName(r.Name, r.Version)
		if slices.ContainsFunc(present, func(d contract.Disk) bool { return d.VM == name }) {
			if err := s.e.Discard(ctx, s.location(name)); err != nil {
				return removed, err
			}
		}
		if err := s.store.Delete(ctx, r.Name, r.Version); err != nil {
			return removed, err
		}
		removed = append(removed, r)
	}
	return removed, nil
}

func (s *Builder) location(ref string) imagescontract.ImageLocation {
	return imagescontract.ImageLocation{Hypervisor: s.e.Name(), Ref: ref}
}
