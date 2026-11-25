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
		got, err := open(t).Read(ctx)
		if err != nil || got.Devices == nil || len(got.Devices) != 0 {
			t.Fatalf("Read = %+v, %v; want an empty book", got, err)
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
		want := devices.Book{Devices: []devices.Device{entry("macbook-pro"), entry("mini")}}
		if got, err := s.Read(ctx); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("Read = %+v, %v; want %+v", got, err, want)
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
		if got, err := s.Read(ctx); err != nil || len(got.Devices) != 0 {
			t.Fatalf("Read = %+v, %v; want the book unchanged", got, err)
		}
	})

	t.Run("a forget is read back", func(t *testing.T) {
		s := open(t)
		_ = s.Change(ctx, func(b *devices.Book) error { b.Devices = append(b.Devices, entry("mini")); return nil })
		_ = s.Change(ctx, func(b *devices.Book) error { b.Forget("mini"); return nil })
		if got, err := s.Read(ctx); err != nil || len(got.Devices) != 0 {
			t.Fatalf("Read = %+v, %v; want the entry gone", got, err)
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
		if got, err := s.Read(ctx); err != nil || len(got.Devices) != 8 {
			t.Fatalf("Read = %d entries, %v; want 8", len(got.Devices), err)
		}
	})
}
