package jsonfile_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the machine's files with the hold wire gives the book; asked, when
// set, hears each hold before it waits.
type holding struct {
	local.FS
	asked chan<- struct{}
}

func (h holding) Hold(ctx context.Context, path string) (func(), error) {
	if h.asked != nil {
		h.asked <- struct{}{}
	}
	return local.Exclusive(ctx, path)
}

func openBook(at string, asked chan<- struct{}) *jsonfile.Book {
	return jsonfile.NewBook(holding{FS: local.New(nil, nil, nil, nil).FS, asked: asked}, at)
}

func TestBookStore(t *testing.T) {
	contracttest.BookStore(t, func(t *testing.T) contract.BookStore {
		return openBook(filepath.Join(t.TempDir(), "station", "vault.json"), nil)
	})
}

func TestTheBookIsAFileOfMode0600(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "vault.json")
	if err := openBook(at, nil).Change(t.Context(), func(b *accounts.Book) error { b.Next(); return nil }); err != nil {
		t.Fatal(err)
	}
	info, err := local.New(nil, nil, nil, nil).FS.Stat(at)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("stat = %v, %v; want mode 0600", info, err)
	}
}

func TestABookOfANewerStationIsRefused(t *testing.T) {
	at := filepath.Join(t.TempDir(), "vault.json")
	if err := os.WriteFile(at, []byte(`{"v": 2, "counter": 1, "entries": []}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var newer *store.NewerError
	if _, err := openBook(at, nil).Read(t.Context()); !errors.As(err, &newer) {
		t.Fatalf("Read = %v; want a newer book refused", err)
	}
	if err := openBook(at, nil).Migrate(t.Context()); !errors.As(err, &newer) {
		t.Fatalf("Migrate = %v; want a newer book refused", err)
	}
}

func TestASecondChangeWaitsForTheFirstToWrite(t *testing.T) {
	// invariant: two books over one file, as two calls of the engine hold;
	// the second change reads the book only once the first has written it.
	at := filepath.Join(t.TempDir(), "station", "vault.json")
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	inside, release, asked := make(chan struct{}), make(chan struct{}), make(chan struct{}, 1)
	first := make(chan error, 1)
	go func() {
		first <- openBook(at, nil).Change(ctx, func(b *accounts.Book) error {
			b.Entries = append(b.Entries, accounts.VaultEntry{ID: b.Next()})
			close(inside)
			select {
			case <-release:
			case <-ctx.Done():
			}
			return nil
		})
	}()
	select {
	case <-inside:
	case <-ctx.Done():
		t.Fatal("the first change never ran")
	}
	second, read := make(chan error, 1), make(chan struct{}, 1)
	go func() {
		second <- openBook(at, asked).Change(ctx, func(b *accounts.Book) error {
			read <- struct{}{}
			b.Entries = append(b.Entries, accounts.VaultEntry{ID: b.Next()})
			return nil
		})
	}()
	select {
	case <-asked:
	case <-read:
		t.Error("the second change read the book while the first held it")
	case <-ctx.Done():
		t.Fatal("the second change never started")
	}
	close(release)
	for _, done := range []chan error{first, second} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal("a change never ended")
		}
	}
	got, err := openBook(at, nil).Read(t.Context())
	if err != nil || len(got.Entries) != 2 || got.Counter != 2 {
		t.Fatalf("Read = %+v, %v; want both changes kept", got, err)
	}
}
