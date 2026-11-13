package contracttest

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
)

func Vault(t *testing.T, subject func(t *testing.T) contract.Vault) {
	ctx := context.Background()
	const name = "operator:example:token"
	secret := []byte(`s3 cr"et\with quotes`)

	t.Run("a secret put is got back byte for byte, and its name is listed", func(t *testing.T) {
		v := subject(t)
		if err := v.Put(ctx, name, secret); err != nil {
			t.Fatalf("Put: %v", err)
		}
		got, err := v.Get(ctx, name)
		if err != nil || !bytes.Equal(got, secret) {
			t.Fatalf("Get = %q, %v; want %q", got, err, secret)
		}
		names, err := v.Names(ctx)
		if err != nil || !slices.Contains(names, name) {
			t.Fatalf("Names = %v, %v; want %s listed", names, err, name)
		}
	})

	t.Run("putting a name again replaces its secret", func(t *testing.T) {
		v := subject(t)
		if err := v.Put(ctx, name, []byte("first")); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := v.Put(ctx, name, []byte("second")); err != nil {
			t.Fatalf("Put again: %v", err)
		}
		if got, err := v.Get(ctx, name); err != nil || string(got) != "second" {
			t.Fatalf("Get = %q, %v; want the second secret", got, err)
		}
	})

	t.Run("a forgotten secret is no longer kept, and forgetting it again changes nothing", func(t *testing.T) {
		v := subject(t)
		if err := v.Put(ctx, name, secret); err != nil {
			t.Fatalf("Put: %v", err)
		}
		if err := v.Forget(ctx, name); err != nil {
			t.Fatalf("Forget: %v", err)
		}
		if _, err := v.Get(ctx, name); !errors.Is(err, accounts.ErrNotKept) {
			t.Fatalf("Get after Forget = %v; want ErrNotKept", err)
		}
		if err := v.Forget(ctx, name); !errors.Is(err, accounts.ErrNotKept) {
			t.Fatalf("Forget again = %v; want ErrNotKept", err)
		}
		if names, _ := v.Names(ctx); slices.Contains(names, name) {
			t.Fatalf("Names = %v; the forgotten name is still listed", names)
		}
	})

	t.Run("a name never kept answers not kept", func(t *testing.T) {
		if _, err := subject(t).Get(ctx, "session:never"); !errors.Is(err, accounts.ErrNotKept) {
			t.Fatalf("Get = %v; want ErrNotKept", err)
		}
	})
}
