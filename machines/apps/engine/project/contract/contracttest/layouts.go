package contracttest

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
)

func Layouts(t *testing.T, open func(t *testing.T) (contract.Layouts, contract.Projects, string)) {
	// invariant: open gives Layouts and the Projects of the same disk over a
	// folder of its own, no project in it nor above it, and that folder.
	acted := errors.New("what act returned")

	t.Run("act runs over the user's own folders, its error as it is", func(t *testing.T) {
		l, _, _ := open(t)
		ran := false
		err := l.Within("", func() error { ran = true; return acted })
		if !ran || !errors.Is(err, acted) {
			t.Fatalf("Within(\"\") ran %v, = %v; want act run and its error", ran, err)
		}
	})

	t.Run("act runs within a project made, its error as it is", func(t *testing.T) {
		l, p, folder := open(t)
		made, err := p.Create(context.Background(), folder, "site", catalog.Harnesses()[0], "")
		if err != nil {
			t.Fatal(err)
		}
		ran := false
		err = l.Within(made.Root, func() error { ran = true; return acted })
		if !ran || !errors.Is(err, acted) {
			t.Fatalf("Within(%s) ran %v, = %v; want act run and its error", made.Root, ran, err)
		}
	})
}
