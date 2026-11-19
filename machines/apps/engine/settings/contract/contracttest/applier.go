package contracttest

import (
	"context"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

type Opened struct {
	Applier contract.Applier
	Package catalog.Setting
	// value: the owner edits what the setting wrote, by hand.
	Edit func(t *testing.T)
}

func Applier(t *testing.T, open func(t *testing.T) Opened) {
	ctx := context.Background()

	t.Run("a setting never applied reads absent, and reading it twice changes nothing", func(t *testing.T) {
		o := open(t)
		first, err := o.Applier.Observe(ctx, o.Package)
		if err != nil {
			t.Fatalf("Observe: %v", err)
		}
		second, _ := o.Applier.Observe(ctx, o.Package)
		if first.Present() || first.Digest != second.Digest {
			t.Fatalf("Observe = %+v then %+v; want absent twice", first, second)
		}
	})

	t.Run("once applied it reads present, exactly as station writes it", func(t *testing.T) {
		o := open(t)
		if _, err := o.Applier.Apply(ctx, o.Package, nil); err != nil {
			t.Fatalf("Apply: %v", err)
		}
		now, err := o.Applier.Observe(ctx, o.Package)
		if err != nil || !now.Present() || now.Diff != "" {
			t.Fatalf("Observe = %+v, %v; want present with no diff", now, err)
		}
		again, _ := o.Applier.Apply(ctx, o.Package, nil)
		after, _ := o.Applier.Observe(ctx, o.Package)
		if after.Digest != now.Digest {
			t.Fatalf("applying again moved the fingerprint: %s, then %s (%+v)", now.Digest, after.Digest, again)
		}
	})

	t.Run("an edit by hand moves the fingerprint", func(t *testing.T) {
		o := open(t)
		if o.Edit == nil {
			t.Skip("nothing of this setting can be edited by hand")
		}
		if _, err := o.Applier.Apply(ctx, o.Package, nil); err != nil {
			t.Fatalf("Apply: %v", err)
		}
		before, _ := o.Applier.Observe(ctx, o.Package)
		o.Edit(t)
		after, _ := o.Applier.Observe(ctx, o.Package)
		if after.Digest == before.Digest {
			t.Fatalf("an edit kept the fingerprint %s", after.Digest)
		}
	})

	t.Run("undone, it reads absent again", func(t *testing.T) {
		o := open(t)
		if _, err := o.Applier.Apply(ctx, o.Package, nil); err != nil {
			t.Fatalf("Apply: %v", err)
		}
		if _, err := o.Applier.Undo(ctx, o.Package); err != nil {
			if errors.Is(err, contract.ErrNoUndo) {
				return
			}
			t.Fatalf("Undo: %v", err)
		}
		if now, _ := o.Applier.Observe(ctx, o.Package); now.Present() {
			t.Fatalf("Observe = %+v after Undo; want absent", now)
		}
	})

	t.Run("an edit by hand survives Undo", func(t *testing.T) {
		o := open(t)
		if o.Edit == nil {
			t.Skip("nothing of this setting can be edited by hand")
		}
		if _, err := o.Applier.Apply(ctx, o.Package, nil); err != nil {
			t.Fatalf("Apply: %v", err)
		}
		o.Edit(t)
		edited, _ := o.Applier.Observe(ctx, o.Package)
		if _, err := o.Applier.Undo(ctx, o.Package); err != nil && !errors.Is(err, contract.ErrNoUndo) {
			t.Fatalf("Undo: %v", err)
		}
		if now, _ := o.Applier.Observe(ctx, o.Package); now.Digest != edited.Digest {
			t.Fatalf("Undo took the owner's edit: %s, then %s", edited.Digest, now.Digest)
		}
	})
}
