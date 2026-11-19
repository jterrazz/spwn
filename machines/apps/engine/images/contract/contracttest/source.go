package contracttest

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/images"
	"github.com/jterrazz/studio-station/apps/engine/images/contract"
)

const Hypervisor = "runner"

type Shared func(t *testing.T, versions ...string) contract.ImageSource

func ImageSource(t *testing.T, share Shared) {
	// value: publish answers a source over a registry where linux is
	// published at each of these versions, and nothing else is.
	t.Run("the latest of an image is its newest released version, a day's second build after its first", func(t *testing.T) {
		s := share(t, "2026.10.20", "2026.11.01.2", "2026.11.01")
		got, err := s.Latest(t.Context(), "linux")
		if err != nil || got != (images.Ref{Name: "linux", Version: "2026.11.01.2"}) {
			t.Fatalf("Latest = %+v, %v; want linux 2026.11.01.2", got, err)
		}
	})

	t.Run("an image never released is said so, never a version", func(t *testing.T) {
		s := share(t, "2026.11.01")
		if got, err := s.Latest(t.Context(), "debian"); !errors.Is(err, contract.ErrNotReleased) {
			t.Fatalf("Latest(debian) = %+v, %v; want ErrNotReleased", got, err)
		}
	})

	t.Run("two versions are fetched from two locations, for the hypervisor asked", func(t *testing.T) {
		s := share(t, "2026.10.20", "2026.11.01")
		older, errOld := s.Locate(t.Context(), images.Ref{Name: "linux", Version: "2026.10.20"}, Hypervisor)
		newer, errNew := s.Locate(t.Context(), images.Ref{Name: "linux", Version: "2026.11.01"}, Hypervisor)
		if errOld != nil || errNew != nil {
			t.Fatalf("Locate: %v, %v", errOld, errNew)
		}
		if older.Hypervisor != Hypervisor || newer.Hypervisor != Hypervisor || older.Ref == "" || older.Ref == newer.Ref {
			t.Fatalf("Locate = %+v and %+v; want two locations for %s", older, newer, Hypervisor)
		}
	})

	t.Run("a hypervisor the image is not released for is told so", func(t *testing.T) {
		s := share(t, "2026.11.01")
		if got, err := s.Locate(t.Context(), images.Ref{Name: "linux", Version: "2026.11.01"}, "hyperv"); !errors.Is(err, contract.ErrNotForHypervisor) {
			t.Fatalf("Locate(hyperv) = %+v, %v; want ErrNotForHypervisor", got, err)
		}
	})
}
