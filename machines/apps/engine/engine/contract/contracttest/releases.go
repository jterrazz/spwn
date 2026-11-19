package contracttest

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract"
)

type ReleasesSubject struct {
	Releases contract.Releases
	// value: makes p the latest release GitHub gives; Down makes GitHub
	// answer no more.
	Release func(p engine.Tagged)
	Down    func()
}

func Releases(t *testing.T, subject func(t *testing.T) ReleasesSubject) {
	t.Run("no release yet is the zero release, with no error", func(t *testing.T) {
		got, err := subject(t).Releases.Latest(context.Background())
		if err != nil || got.Tag != "" {
			t.Errorf("Latest = %+v, %v; want no release", got, err)
		}
	})
	t.Run("the latest release is the one made last, with its page and assets", func(t *testing.T) {
		s := subject(t)
		s.Release(engine.Tagged{Tag: "v0.21.0", URL: "https://github.com/jterrazz/spwn/releases/tag/v0.21.0", Assets: []string{"station-darwin-arm64"}})
		s.Release(engine.Tagged{Tag: "v0.22.0", URL: "https://github.com/jterrazz/spwn/releases/tag/v0.22.0", Assets: []string{"station-linux-arm64", "station-darwin-arm64"}})
		got, err := s.Releases.Latest(context.Background())
		if err != nil || got.Tag != "v0.22.0" || got.URL != "https://github.com/jterrazz/spwn/releases/tag/v0.22.0" || !slices.Contains(got.Assets, "station-darwin-arm64") {
			t.Errorf("Latest = %+v, %v; want v0.22.0 with its page and both assets", got, err)
		}
	})
	t.Run("a GitHub that does not answer is an error", func(t *testing.T) {
		s := subject(t)
		s.Down()
		if _, err := s.Releases.Latest(context.Background()); err == nil {
			t.Error("Latest answered with GitHub down")
		}
	})
}

func ReleaseCache(t *testing.T, subject func(t *testing.T) contract.ReleaseCache) {
	t.Run("none written reads as none", func(t *testing.T) {
		if _, ok, err := subject(t).Read(context.Background()); ok || err != nil {
			t.Errorf("Read = %v, %v; want none", ok, err)
		}
	})
	t.Run("a write is read back whole", func(t *testing.T) {
		c := subject(t)
		at := time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)
		want := engine.ReleaseKept{Read: at, Latest: &engine.Tagged{Tag: "v0.22.0", URL: "https://github.com/jterrazz/spwn/releases/tag/v0.22.0"}, Refused: &engine.ReleaseRefusal{At: at.Add(time.Hour), Message: "GitHub said 502"}}
		if err := c.Write(context.Background(), engine.ReleaseKept{Read: at.Add(-time.Hour)}); err != nil {
			t.Fatal(err)
		}
		if err := c.Write(context.Background(), want); err != nil {
			t.Fatal(err)
		}
		got, ok, err := c.Read(context.Background())
		if err != nil || !ok || !got.Read.Equal(want.Read) || got.Latest == nil || got.Latest.Tag != "v0.22.0" || got.Latest.URL != want.Latest.URL ||
			got.Refused == nil || !got.Refused.At.Equal(want.Refused.At) || got.Refused.Message != want.Refused.Message {
			t.Errorf("Read = %+v, %v, %v; want %+v", got, ok, err, want)
		}
	})
}
