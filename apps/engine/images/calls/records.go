package calls

import (
	"context"
	"slices"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/catalog"

	"github.com/jterrazz/studio-station/apps/engine/images"
)

const (
	imagePrefix = "station-image-"
	buildPrefix = "station-build-"
)

func DiskName(image, version string) string { return imagePrefix + image + "-" + version }

// invariant: an image's versions are those held under its name and under
// each name it was renamed from, the newest first.
func (s *Builder) records(img catalog.Image) ([]images.Record, error) {
	var out []images.Record
	for _, name := range append([]string{img.Name}, img.RenamedFrom...) {
		held, err := s.store.List(context.Background(), name)
		if err != nil {
			return nil, err
		}
		out = append(out, held...)
	}
	slices.SortStableFunc(out, func(a, b images.Record) int { return images.CompareVersions(b.Version, a.Version) })
	return out, nil
}

func (s *Builder) writeRecord(r images.Record) error { return s.store.Put(context.Background(), r) }

func (s *Builder) nextVersion(img catalog.Image) (string, error) {
	existing, err := s.records(img)
	if err != nil {
		return "", err
	}
	taken := make(map[string]bool, len(existing))
	for _, r := range existing {
		taken[r.Version] = true
	}
	day := s.m.Now().UTC().Format("2006.01.02")
	if !taken[day] {
		return day, nil
	}
	for n := 2; ; n++ {
		if candidate := day + "." + strconv.Itoa(n); !taken[candidate] {
			return candidate, nil
		}
	}
}
