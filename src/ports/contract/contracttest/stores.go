package contracttest

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/core/store"
	"github.com/jterrazz/studio-station/src/ports"
	"github.com/jterrazz/studio-station/src/ports/contract"
)

func reservation(vm string, guest, reserved int) ports.Reservation {
	return ports.Reservation{VM: vm, GuestPort: guest, MacPort: reserved, ReservedAt: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)}
}

func ReservationStore(t *testing.T, open func(t *testing.T) contract.ReservationStore) {
	ctx := context.Background()

	t.Run("a reservation put is got back as it was put", func(t *testing.T) {
		s := open(t)
		want := reservation("linux-1", 5173, 40001)
		if err := s.Put(ctx, want); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if got, err := s.Get(ctx, "linux-1", 5173); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("Get = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("reservations are listed by VM then port, and reserving again replaces", func(t *testing.T) {
		s := open(t)
		for _, r := range []ports.Reservation{reservation("linux-1", 7000, 40002), reservation("api", 3000, 40010), reservation("linux-1", 5173, 40001), reservation("linux-1", 7000, 40003)} {
			if err := s.Put(ctx, r); err != nil {
				t.Fatalf("Put: %v", err)
			}
		}
		list, err := s.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		keys := make([]string, 0, len(list))
		for _, r := range list {
			keys = append(keys, r.Key())
		}
		if !reflect.DeepEqual(keys, []string{"api:3000", "linux-1:5173", "linux-1:7000"}) || list[2].MacPort != 40003 {
			t.Errorf("List = %v; want sorted, linux-1:7000 at 40003", list)
		}
	})

	t.Run("a deleted reservation is gone, and deleting it again changes nothing", func(t *testing.T) {
		s := open(t)
		if err := s.Put(ctx, reservation("linux-1", 5173, 40001)); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := s.Delete(ctx, "linux-1", 5173); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := s.Get(ctx, "linux-1", 5173); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Get after Delete = %v; want not found", err)
		}
		if err := s.Delete(ctx, "linux-1", 5173); err != nil {
			t.Errorf("Delete again = %v", err)
		}
	})
}

func DeclarationStore(t *testing.T, open func(t *testing.T) contract.DeclarationStore) {
	ctx := context.Background()

	t.Run("a declaration put is got back, and declaring again replaces its route", func(t *testing.T) {
		s := open(t)
		if err := s.Put(ctx, ports.Declaration{Machine: "infra", Port: 443, Via: "lan"}); err != nil {
			t.Fatalf("Put: %v", err)
		}
		want := ports.Declaration{Machine: "infra", Port: 443, Via: "cloudflare-tunnel"}
		if err := s.Put(ctx, want); err != nil {
			t.Fatalf("Put again: %v", err)
		}
		if got, err := s.Get(ctx, "infra", 443); err != nil || got != want {
			t.Fatalf("Get = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("declarations are listed by machine then port", func(t *testing.T) {
		s := open(t)
		for _, d := range []ports.Declaration{{Machine: "infra", Port: 443, Via: "lan"}, {Machine: "api", Port: 8080, Via: "lan"}, {Machine: "infra", Port: 22, Via: "lan"}} {
			if err := s.Put(ctx, d); err != nil {
				t.Fatalf("Put: %v", err)
			}
		}
		list, err := s.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		keys := make([]string, 0, len(list))
		for _, d := range list {
			keys = append(keys, d.Key())
		}
		if !reflect.DeepEqual(keys, []string{"api:8080", "infra:22", "infra:443"}) {
			t.Errorf("List = %v; want sorted", keys)
		}
	})

	t.Run("a deleted declaration is gone, and deleting it again changes nothing", func(t *testing.T) {
		s := open(t)
		if err := s.Put(ctx, ports.Declaration{Machine: "infra", Port: 443, Via: "lan"}); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := s.Delete(ctx, "infra", 443); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := s.Get(ctx, "infra", 443); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Get after Delete = %v; want not found", err)
		}
		if err := s.Delete(ctx, "infra", 443); err != nil {
			t.Errorf("Delete again = %v", err)
		}
	})
}
