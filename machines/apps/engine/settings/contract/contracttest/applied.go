package contracttest

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

func applied(setting, digest string) settings.Applied {
	return settings.Applied{Setting: setting, Digest: digest, At: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)}
}

func AppliedStore(t *testing.T, open func(t *testing.T) contract.AppliedStore) {
	ctx := context.Background()

	t.Run("a setting applied is got back, and applying it again replaces it", func(t *testing.T) {
		s := open(t)
		if err := s.Put(ctx, applied("starship", "sha256:old")); err != nil {
			t.Fatalf("Put: %v", err)
		}
		want := applied("starship", "sha256:new")
		if err := s.Put(ctx, want); err != nil {
			t.Fatalf("Put again: %v", err)
		}
		if got, err := s.Get(ctx, "starship"); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("Get = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("the settings applied are listed by name", func(t *testing.T) {
		s := open(t)
		for _, name := range []string{"zed", "git-aliases", "starship"} {
			if err := s.Put(ctx, applied(name, "sha256:"+name)); err != nil {
				t.Fatalf("Put: %v", err)
			}
		}
		list, err := s.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		names := make([]string, 0, len(list))
		for _, a := range list {
			names = append(names, a.Setting)
		}
		if !reflect.DeepEqual(names, []string{"git-aliases", "starship", "zed"}) {
			t.Errorf("List = %v; want by name", names)
		}
	})

	t.Run("a deleted setting is gone, and deleting it again changes nothing", func(t *testing.T) {
		s := open(t)
		if err := s.Put(ctx, applied("starship", "sha256:a")); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := s.Delete(ctx, "starship"); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := s.Get(ctx, "starship"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Get after Delete = %v; want not found", err)
		}
		if err := s.Delete(ctx, "starship"); err != nil {
			t.Errorf("Delete again = %v", err)
		}
	})
}
