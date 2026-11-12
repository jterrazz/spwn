package calls

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jterrazz/studio-station/src/images"
	imagescontract "github.com/jterrazz/studio-station/src/images/contract"
)

// value: bare, every image this Mac holds a version of; named, those.
func (s *Builder) updating(ctx context.Context, names []string) ([]string, error) {
	if len(names) > 0 {
		var out []string
		for _, name := range names {
			img, err := s.image(ctx, name)
			var none *images.NoRecipeError
			if err != nil && !errors.As(err, &none) {
				return nil, err
			}
			if !slices.Contains(out, img.Name) {
				out = append(out, img.Name)
			}
		}
		return out, nil
	}
	list, err := s.all(ctx)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, img := range list {
		held, err := s.records(img)
		if err != nil {
			return nil, err
		}
		if len(held) > 0 {
			out = append(out, img.Name)
		}
	}
	return out, nil
}

func (s *Builder) Update(ctx context.Context, names []string, progress func(images.Step)) ([]images.Update, error) {
	wanted, err := s.updating(ctx, names)
	if err != nil || len(wanted) == 0 {
		return nil, err
	}
	if _, err := s.e.Version(ctx); err != nil {
		return nil, &images.NoEngineError{Hypervisor: s.e.Name(), Cause: err}
	}
	if progress == nil {
		progress = func(images.Step) {}
	}
	var out []images.Update
	for _, name := range wanted {
		u, err := s.updateOne(ctx, name, progress)
		if err != nil {
			return out, err
		}
		out = append(out, u)
	}
	return out, nil
}

// invariant: an update fetches only a version newer than every one held;
// it lands whole as an image disk or not at all, so a failed fetch leaves
// the image as it was.
func (s *Builder) updateOne(ctx context.Context, name string, progress func(images.Step)) (images.Update, error) {
	img, _ := s.image(ctx, name)
	latest, err := s.source.Latest(ctx, name)
	// invariant: an image renamed is asked of its registry by each name
	// it was known by too, for one release.
	for _, was := range img.RenamedFrom {
		if !errors.Is(err, imagescontract.ErrNotReleased) {
			break
		}
		latest, err = s.source.Latest(ctx, was)
	}
	switch {
	case errors.Is(err, imagescontract.ErrNotReleased):
		return images.Update{}, &images.NotReleasedError{Image: name, Buildable: img.Recipe != ""}
	case err != nil:
		return images.Update{}, &images.RegistryError{Image: name, Cause: err}
	}
	held, err := s.records(img)
	if err != nil {
		return images.Update{}, err
	}
	if len(held) > 0 && images.CompareVersions(held[0].Version, latest.Version) >= 0 {
		return images.Update{Ref: held[0].Ref(), Newest: true}, nil
	}
	at, err := s.source.Locate(ctx, latest, s.e.Name())
	if errors.Is(err, imagescontract.ErrNotForHypervisor) {
		return images.Update{}, &images.NotReleasedError{Image: name}
	}
	if err != nil {
		return images.Update{}, &images.RegistryError{Image: name, Cause: err}
	}
	failed := func(step string, cause error) (images.Update, error) {
		return images.Update{}, &images.FetchFailedError{Image: name, Version: latest.Version, Step: step, Cause: cause}
	}
	progress(images.Step{Label: "fetching " + name + " " + latest.Version})
	if err := s.e.Fetch(ctx, at, nil); err != nil {
		return failed("fetching", err)
	}
	disk := DiskName(name, latest.Version)
	progress(images.Step{Label: "recording " + name + " " + latest.Version})
	if err := s.e.Clone(ctx, at, disk); err != nil {
		_ = s.e.Discard(context.WithoutCancel(ctx), at)
		return failed("keeping the fetched image", err)
	}
	_ = s.e.Discard(ctx, at)
	rec := images.Record{Name: name, Version: latest.Version, BuiltAt: s.m.Now().UTC().Truncate(time.Second), Fetched: true}
	rec.Base = img.Base
	if disks, err := s.e.Disks(ctx); err == nil {
		for _, d := range disks {
			if d.VM == disk {
				rec.SizeGB = int(d.Bytes / 1e9)
			}
		}
	}
	if err := s.writeRecord(rec); err != nil {
		_ = s.e.Delete(context.WithoutCancel(ctx), disk)
		return failed("writing the image record", err)
	}
	return images.Update{Ref: rec.Ref()}, nil
}
