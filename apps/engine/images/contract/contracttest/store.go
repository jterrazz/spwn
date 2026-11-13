package contracttest

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/images"
	"github.com/jterrazz/studio-station/apps/engine/images/contract"
)

func built(version string) images.Record {
	return images.Record{
		Name: "linux", Version: version, Base: "ubuntu:24.04", Station: "1.4.0", RecipeSHA: "abc123",
		BuiltAt: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC), SizeGB: 12,
	}
}

func ImageStore(t *testing.T, open func(t *testing.T) contract.ImageStore) {
	ctx := context.Background()

	t.Run("a version put is got back as it was put", func(t *testing.T) {
		s := open(t)
		want := built("2026.11.01")
		if err := s.Put(ctx, want); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if got, err := s.Get(ctx, "linux", "2026.11.01"); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("Get = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("an image's versions are listed newest first, another image's apart", func(t *testing.T) {
		s := open(t)
		for _, v := range []string{"2026.10.30", "2026.11.01.2", "2026.11.01"} {
			if err := s.Put(ctx, built(v)); err != nil {
				t.Fatalf("Put %s: %v", v, err)
			}
		}
		other := built("2026.11.02")
		other.Name = "macos"
		if err := s.Put(ctx, other); err != nil {
			t.Fatalf("Put macos: %v", err)
		}
		list, err := s.List(ctx, "linux")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		versions := make([]string, 0, len(list))
		for _, r := range list {
			versions = append(versions, r.Version)
		}
		if !reflect.DeepEqual(versions, []string{"2026.11.01.2", "2026.11.01", "2026.10.30"}) {
			t.Errorf("List = %v; want the newest first, linux alone", versions)
		}
	})

	t.Run("a deleted version is gone, and deleting it again changes nothing", func(t *testing.T) {
		s := open(t)
		if err := s.Put(ctx, built("2026.11.01")); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := s.Delete(ctx, "linux", "2026.11.01"); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := s.Get(ctx, "linux", "2026.11.01"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Get after Delete = %v; want not found", err)
		}
		if err := s.Delete(ctx, "linux", "2026.11.01"); err != nil {
			t.Errorf("Delete again = %v; want nothing", err)
		}
		if list, _ := s.List(ctx, "linux"); len(list) != 0 {
			t.Errorf("List = %+v; want empty", list)
		}
	})
}
