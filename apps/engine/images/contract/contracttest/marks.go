package contracttest

import (
	"context"
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/images"
	"github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

func BuildMarks(t *testing.T, open func(t *testing.T) contract.BuildMarks) {
	ctx := context.Background()
	owner := &machines.Process{PID: 4242, Start: "1762250400.120000"}

	t.Run("a mark is read back as it was made", func(t *testing.T) {
		b := open(t)
		want := images.Making{Image: "linux", Version: "2026.11.01", Step: images.BuildPackages, Owner: owner}
		if err := b.Mark(ctx, want); err != nil {
			t.Fatalf("Mark: %v", err)
		}
		if got, err := b.Marked(ctx); err != nil || !reflect.DeepEqual(got, []images.Making{want}) {
			t.Fatalf("Marked = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("a second mark of an image replaces its first, another image's stays", func(t *testing.T) {
		b := open(t)
		for _, m := range []images.Making{
			{Image: "linux", Version: "2026.11.01", Step: images.BuildBaseSystem, Owner: owner},
			{Image: "macos", Version: "1", Step: images.BuildSeal, Owner: owner},
			{Image: "linux", Version: "2026.11.01", Step: images.BuildSettings, Owner: owner},
		} {
			if err := b.Mark(ctx, m); err != nil {
				t.Fatalf("Mark %s: %v", m.Image, err)
			}
		}
		got, err := b.Marked(ctx)
		if err != nil || len(got) != 2 {
			t.Fatalf("Marked = %+v, %v; want two marks", got, err)
		}
		for _, m := range got {
			if m.Image == "linux" && m.Step != images.BuildSettings {
				t.Fatalf("linux is at %s, want its second mark", m.Step)
			}
		}
	})

	t.Run("a cleared image has no mark, and clearing one never marked is no error", func(t *testing.T) {
		b := open(t)
		if err := b.Mark(ctx, images.Making{Image: "linux", Version: "1", Step: images.BuildSeal, Owner: owner}); err != nil {
			t.Fatalf("Mark: %v", err)
		}
		for _, image := range []string{"linux", "linux", "never-marked"} {
			if err := b.Clear(ctx, image); err != nil {
				t.Fatalf("Clear %s: %v", image, err)
			}
		}
		if got, err := b.Marked(ctx); err != nil || len(got) != 0 {
			t.Fatalf("Marked = %+v, %v; want none", got, err)
		}
	})
}
