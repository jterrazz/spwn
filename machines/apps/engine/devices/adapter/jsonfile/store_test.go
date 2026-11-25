package jsonfile_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the machine's files with the hold wire gives the book; torn, when
// set, fails every Replace as a write cut short does.
type holding struct {
	local.FS
	torn bool
}

func (h holding) Hold(ctx context.Context, path string) (func(), error) {
	return local.Exclusive(ctx, path)
}

func (h holding) Replace(path string, data []byte, perm fs.FileMode) error {
	if h.torn {
		return errors.New("the disk is full")
	}
	return h.FS.Replace(path, data, perm)
}

func open(at string, torn bool) *jsonfile.Store {
	return jsonfile.New(holding{FS: local.New(nil, nil, nil, nil).FS, torn: torn}, at)
}

func TestStore(t *testing.T) {
	contracttest.Store(t, func(t *testing.T) contract.Store {
		return open(filepath.Join(t.TempDir(), "station", "devices.json"), false)
	})
}

func add(name string) func(*devices.Book) error {
	return func(b *devices.Book) error {
		b.Devices = append(b.Devices, devices.Device{Name: name, Key: "ssh-ed25519 AAAA" + name, Channels: devices.AllChannels()})
		return nil
	}
}

func TestTheBookIsAFileOfMode0600(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	if err := open(at, false).Change(t.Context(), add("mini")); err != nil {
		t.Fatal(err)
	}
	info, err := local.New(nil, nil, nil, nil).FS.Stat(at)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("stat = %v, %v; want mode 0600", info, err)
	}
}

func TestAWriteThatFailsLeavesTheBookItReplaces(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	if err := open(at, false).Change(t.Context(), add("mini")); err != nil {
		t.Fatal(err)
	}
	var unwritable *store.UnwritableError
	if err := open(at, true).Change(t.Context(), add("macbook-pro")); !errors.As(err, &unwritable) {
		t.Fatalf("Change = %v; want the write refused", err)
	}
	got, err := open(at, false).Read(t.Context())
	if err != nil || len(got.Devices) != 1 || got.Devices[0].Name != "mini" {
		t.Fatalf("Read = %+v, %v; want the book before the failed write", got, err)
	}
}

func TestABookOfANewerSpwnIsRefused(t *testing.T) {
	at := filepath.Join(t.TempDir(), "devices.json")
	if err := os.WriteFile(at, []byte(`{"v": 2, "devices": []}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var newer *store.NewerError
	if _, err := open(at, false).Read(t.Context()); !errors.As(err, &newer) {
		t.Fatalf("Read = %v; want a newer book refused", err)
	}
}
