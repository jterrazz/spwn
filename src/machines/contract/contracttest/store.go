package contracttest

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/core/store"
	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/contract"
)

func vmRecord(name string) machines.Record {
	return machines.Record{
		Name: name, Image: "linux", Version: "2026.11.01", Lifetime: machines.Permanent, Want: "running",
		Env: map[string]string{"PORT": "40001"}, CPUs: 4, MemoryBytes: 8 << 30, DiskBytes: 50 << 30,
		CreatedAt: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC),
	}
}

func MachineStore(t *testing.T, open func(t *testing.T) contract.MachineStore) {
	ctx := context.Background()

	t.Run("a VM put is got back as it was put, and listed", func(t *testing.T) {
		s := open(t)
		want := vmRecord("api")
		if err := s.Put(ctx, want); err != nil {
			t.Fatalf("Put: %v", err)
		}
		got, err := s.Get(ctx, "api")
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("Get = %+v, %v; want %+v", got, err, want)
		}
		list, err := s.List(ctx)
		if err != nil || len(list) != 1 || list[0].Name != "api" {
			t.Fatalf("List = %+v, %v; want api alone", list, err)
		}
	})

	t.Run("the VMs are listed by name, and putting one again replaces it", func(t *testing.T) {
		s := open(t)
		for _, name := range []string{"linux-2", "api", "linux-1"} {
			if err := s.Put(ctx, vmRecord(name)); err != nil {
				t.Fatalf("Put %s: %v", name, err)
			}
		}
		stopped := vmRecord("api")
		stopped.Want = "stopped"
		if err := s.Put(ctx, stopped); err != nil {
			t.Fatalf("Put again: %v", err)
		}
		list, err := s.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		names := make([]string, 0, len(list))
		for _, r := range list {
			names = append(names, r.Name)
		}
		if !reflect.DeepEqual(names, []string{"api", "linux-1", "linux-2"}) || list[0].Want != "stopped" {
			t.Errorf("List = %v (api wants %q); want sorted, api replaced", names, list[0].Want)
		}
	})

	t.Run("a stopped workspace's agents stay through a put that names none, and an empty list clears them", func(t *testing.T) {
		s := open(t)
		stopped := vmRecord("api")
		stopped.Agents = []string{"docs-bot"}
		if err := s.Put(ctx, stopped); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := s.Put(ctx, vmRecord("api")); err != nil {
			t.Fatalf("Put again: %v", err)
		}
		if got, err := s.Get(ctx, "api"); err != nil || !reflect.DeepEqual(got.Agents, []string{"docs-bot"}) {
			t.Fatalf("Get = %v, %v; want docs-bot kept", got.Agents, err)
		}
		cleared := vmRecord("api")
		cleared.Agents = []string{}
		if err := s.Put(ctx, cleared); err != nil {
			t.Fatalf("Put cleared: %v", err)
		}
		if got, err := s.Get(ctx, "api"); err != nil || len(got.Agents) != 0 {
			t.Errorf("Get = %v, %v; want none", got.Agents, err)
		}
	})

	t.Run("a deleted VM is gone, and deleting it again changes nothing", func(t *testing.T) {
		s := open(t)
		if err := s.Put(ctx, vmRecord("api")); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := s.Delete(ctx, "api"); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := s.Get(ctx, "api"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("Get after Delete = %v; want not found", err)
		}
		if err := s.Delete(ctx, "api"); err != nil {
			t.Errorf("Delete again = %v; want nothing", err)
		}
		if list, _ := s.List(ctx); len(list) != 0 {
			t.Errorf("List = %+v; want empty", list)
		}
	})
}
