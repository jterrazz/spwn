package fake

import (
	"context"
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/images"
	"github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

var ErrRegistryDown = errors.New("the registry does not answer, as the world scripted it")

type Source struct {
	// invariant: the world's registry: each image's latest published version.
	w *world.World
	// value: the hypervisors the world's images are published for.
	fetchers []string
}

var _ contract.ImageSource = (*Source)(nil)

func SourceOf(w *world.World, fetchers ...string) *Source {
	return &Source{w: w, fetchers: fetchers}
}

func (s *Source) Latest(_ context.Context, name string) (images.Ref, error) {
	s.w.Lock()
	defer s.w.Unlock()
	if s.w.Fails("ImageSource.Latest") {
		return images.Ref{}, ErrRegistryDown
	}
	v, ok := s.w.Registry[name]
	if !ok {
		return images.Ref{}, contract.ErrNotReleased
	}
	return images.Ref{Name: name, Version: v}, nil
}

func (s *Source) Locate(_ context.Context, r images.Ref, hypervisor string) (contract.ImageLocation, error) {
	if !slices.Contains(s.fetchers, hypervisor) {
		return contract.ImageLocation{}, contract.ErrNotForHypervisor
	}
	return contract.ImageLocation{Hypervisor: hypervisor, Ref: "ghcr.io/jterrazz/station-" + r.Name + ":" + r.Version}, nil
}
