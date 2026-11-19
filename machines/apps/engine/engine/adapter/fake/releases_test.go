package fake_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestReleases(t *testing.T) {
	contracttest.Releases(t, func(*testing.T) contracttest.ReleasesSubject {
		w := &world.World{}
		return contracttest.ReleasesSubject{
			Releases: fake.ReleasesOf(w),
			Release: func(p engine.Tagged) {
				w.Releases = append(w.Releases, world.Release{Tag: p.Tag, URL: p.URL, Assets: p.Assets})
			},
			Down: func() { w.Script = append(w.Script, world.Fault{Call: "Releases.Latest"}) },
		}
	})
}

func TestReleaseCache(t *testing.T) {
	contracttest.ReleaseCache(t, func(*testing.T) contract.ReleaseCache {
		return fake.ReleaseCacheOf(&world.World{Macs: map[string]*world.Mac{"macbook-pro": {}}}, "macbook-pro")
	})
}
