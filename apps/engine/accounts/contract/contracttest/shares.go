package contracttest

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
)

func share(name string) accounts.Share {
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	return accounts.Share{Name: name, At: at, Until: at.Add(accounts.DefaultShareTerm)}
}

func ShareStore(t *testing.T, open func(t *testing.T, vms ...string) contract.ShareStore) {
	// invariant: open hands a store whose machines already hold a record for
	// each VM named, and none other.
	ctx := context.Background()

	t.Run("a VM's shares put are got back and listed under it", func(t *testing.T) {
		s := open(t, "api", "linux-1")
		want := []accounts.Share{share("session:a"), share("operator:claude:oauth-token")}
		if err := s.Put(ctx, "api", want); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if got, err := s.Get(ctx, "api"); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("Get = %+v, %v; want %+v", got, err, want)
		}
		all, err := s.List(ctx)
		if err != nil || !reflect.DeepEqual(all, map[string][]accounts.Share{"api": want}) {
			t.Fatalf("List = %+v, %v; want api's shares alone", all, err)
		}
	})

	t.Run("putting none takes every share back, and deleting again changes nothing", func(t *testing.T) {
		s := open(t, "api")
		if err := s.Put(ctx, "api", []accounts.Share{share("session:a")}); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := s.Put(ctx, "api", nil); err != nil {
			t.Fatalf("Put none: %v", err)
		}
		if got, err := s.Get(ctx, "api"); err != nil || len(got) != 0 {
			t.Errorf("Get = %+v, %v; want none", got, err)
		}
		if err := s.Delete(ctx, "api"); err != nil {
			t.Errorf("Delete = %v; want nothing", err)
		}
		if all, _ := s.List(ctx); len(all) != 0 {
			t.Errorf("List = %+v; want empty", all)
		}
	})

	t.Run("a VM with no record cannot be handed a share", func(t *testing.T) {
		if err := open(t).Put(ctx, "gone", []accounts.Share{share("session:a")}); err == nil {
			t.Error("Put on a VM with no record succeeded")
		}
	})
}
