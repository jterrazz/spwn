package contracttest

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
)

func Store(t *testing.T, open func(t *testing.T) contract.Store) {
	// invariant: open hands an empty store; each case works on one store.
	ctx := context.Background()
	entry := func(name string) devices.Device {
		return devices.Device{Name: name, Key: "ssh-ed25519 AAAA" + name, Channels: []devices.Channel{devices.ChannelPipe}, Added: time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC), Seal: []byte("seal of " + name)}
	}

	t.Run("an empty store reads a book with no entry", func(t *testing.T) {
		got, head, err := open(t).Read(ctx)
		if err != nil || got.Devices == nil || len(got.Devices) != 0 || head != (devices.Head{}) || !got.Anchored(head) {
			t.Fatalf("Read = %+v, %+v, %v; want an empty book, anchored nowhere", got, head, err)
		}
	})

	t.Run("a change is read back whole", func(t *testing.T) {
		s := open(t)
		if err := s.Change(ctx, func(b *devices.Book) error {
			b.Devices = append(b.Devices, entry("macbook-pro"), entry("mini"))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		want := devices.Book{Generation: 1, Devices: []devices.Device{entry("macbook-pro"), entry("mini")}}
		got, head, err := s.Read(ctx)
		if err != nil || !reflect.DeepEqual(got, want) || head != want.Head() {
			t.Fatalf("Read = %+v, %+v, %v; want %+v anchored", got, head, err, want)
		}
	})

	t.Run("a change whose edit fails writes nothing", func(t *testing.T) {
		s := open(t)
		refused := errors.New("refused")
		err := s.Change(ctx, func(b *devices.Book) error {
			b.Devices = append(b.Devices, entry("mini"))
			return refused
		})
		if !errors.Is(err, refused) {
			t.Fatalf("Change = %v; want the edit's error", err)
		}
		if got, head, err := s.Read(ctx); err != nil || len(got.Devices) != 0 || head != (devices.Head{}) {
			t.Fatalf("Read = %+v, %+v, %v; want the book and its head unchanged", got, head, err)
		}
	})

	t.Run("a forget is read back", func(t *testing.T) {
		s := open(t)
		_ = s.Change(ctx, func(b *devices.Book) error { b.Devices = append(b.Devices, entry("mini")); return nil })
		_ = s.Change(ctx, func(b *devices.Book) error { b.Forget("mini"); return nil })
		got, head, err := s.Read(ctx)
		if err != nil || len(got.Devices) != 0 || got.Generation != 2 || !got.Anchored(head) {
			t.Fatalf("Read = %+v, %+v, %v; want the entry gone at generation 2, anchored", got, head, err)
		}
	})

	t.Run("changes at once keep every entry", func(t *testing.T) {
		s := open(t)
		var wg sync.WaitGroup
		for i := range 8 {
			wg.Go(func() {
				if err := s.Change(ctx, func(b *devices.Book) error { b.Devices = append(b.Devices, entry("d"+strconv.Itoa(i))); return nil }); err != nil {
					t.Error(err)
				}
			})
		}
		wg.Wait()
		if got, head, err := s.Read(ctx); err != nil || len(got.Devices) != 8 || got.Generation != 8 || !got.Anchored(head) {
			t.Fatalf("Read = %d entries at %d, %v; want 8 at generation 8, anchored", len(got.Devices), got.Generation, err)
		}
	})

	t.Run("each change starts from the book it raised", func(t *testing.T) {
		s := open(t)
		var seen []int
		for _, name := range []string{"mini", "laptop"} {
			if err := s.Change(ctx, func(b *devices.Book) error {
				seen = append(seen, len(b.Devices))
				b.Devices = append(b.Devices, entry(name))
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(seen, []int{0, 1}) {
			t.Fatalf("the edits saw %v entries; want 0 then 1", seen)
		}
	})
}

func Anchor(t *testing.T, open func(t *testing.T) contract.Anchor) {
	// invariant: open hands an anchor that keeps nothing yet.
	ctx := context.Background()
	head := devices.Book{Generation: 3, Devices: []devices.Device{{Name: "mini", Key: "ssh-ed25519 AAAAmini"}}}.Head()

	t.Run("none kept reads as the zero head", func(t *testing.T) {
		if got, err := open(t).Read(ctx); err != nil || got != (devices.Head{}) {
			t.Fatalf("Read = %+v, %v; want the zero head", got, err)
		}
	})

	t.Run("a head written is read back, the next replacing it", func(t *testing.T) {
		a := open(t)
		if err := a.Write(ctx, head); err != nil {
			t.Fatal(err)
		}
		next := devices.Book{Generation: 4}.Head()
		if err := a.Write(ctx, next); err != nil {
			t.Fatal(err)
		}
		if got, err := a.Read(ctx); err != nil || got != next {
			t.Fatalf("Read = %+v, %v; want %+v", got, err, next)
		}
	})
}
