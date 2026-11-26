package jsonfile_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
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

// value: a login keychain in memory: the head it keeps, how often it was
// read, and a refusal every write meets when set.
type keychain struct {
	mu      sync.Mutex
	head    devices.Head
	reads   int
	refused error
	locked  error
}

func (k *keychain) Read(context.Context) (devices.Head, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reads++
	if k.locked != nil {
		return devices.Head{}, k.locked
	}
	return k.head, nil
}

func (k *keychain) Write(_ context.Context, h devices.Head) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.refused != nil {
		return k.refused
	}
	k.head = h
	return nil
}

func (k *keychain) set(h devices.Head) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.head = h
}

func openOn(at string, torn bool, k *keychain) *jsonfile.Store {
	return jsonfile.New(holding{FS: local.New(nil, nil, nil, nil).FS, torn: torn}, at, k, nil)
}

// value: this Mac's running engine as another process asks it: the head
// its store holds.
type engineOf struct{ s *jsonfile.Store }

func (e engineOf) Held(ctx context.Context) (devices.Head, bool, error) {
	h, _, ok := e.s.Held(ctx)
	return h, ok, nil
}

func TestABookAndAnAnchorMovedForwardWhileTheEngineRunsAreNotTheAnchoredOnes(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	k := &keychain{}
	engine := openOn(at, false, k)
	engine.Sole()
	if err := engine.Change(t.Context(), add("laptop")); err != nil {
		t.Fatal(err)
	}
	book, _, err := engine.Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	laptop := book.Devices[0]
	if err = engine.Change(t.Context(), add("desk")); err != nil {
		t.Fatal(err)
	}
	if err = engine.Change(t.Context(), func(b *devices.Book) error { b.Forget("laptop"); return nil }); err != nil {
		t.Fatal(err)
	}
	held, _, _ := engine.Held(t.Context())
	forged := devices.Book{Generation: held.Generation + 5, Devices: []devices.Device{laptop, {Name: "desk", Key: "ssh-ed25519 AAAAdesk", Channels: devices.AllChannels()}}}
	data, err := json.Marshal(struct {
		V int `json:"v"`
		devices.Book
	}{V: jsonfile.Version, Book: forged})
	if err != nil {
		t.Fatal(err)
	}
	if err = files().WriteFile(at, data, 0o600); err != nil {
		t.Fatal(err)
	}
	k.set(forged.Head())
	cli := jsonfile.New(holding{FS: files()}, at, k, engineOf{engine})
	for name, s := range map[string]*jsonfile.Store{"the engine": engine, "another process": cli} {
		book, head, readErr := s.Read(t.Context())
		if readErr != nil || book.Generation != forged.Generation || book.Anchored(head) || head != held {
			t.Errorf("%s read %+v, %+v, %v; want the book at %d not anchored, the engine's head %+v kept", name, book, head, readErr, forged.Generation, held)
		}
		if got := devices.Check(book, head, "mini", nil); !got.Stale || len(got.Trusted) != 0 {
			t.Errorf("%s checked %+v; want nothing of the book trusted", name, got)
		}
	}
	if _, changed, ok := engine.Held(t.Context()); !changed || !ok {
		t.Errorf("the engine's hold says changed %v, ok %v; want the book said changed outside Spwn", changed, ok)
	}
	if err = engine.Change(t.Context(), add("tablet")); err != nil {
		t.Fatal(err)
	}
	if book, head, err := engine.Read(t.Context()); err != nil || !book.Anchored(head) || len(book.Devices) != 1 || book.Devices[0].Name != "tablet" || head.Generation != held.Generation+1 {
		t.Errorf("Read = %+v, %+v, %v; want a new book of tablet alone, one past the engine's own", book, head, err)
	}
}

func open(at string, torn bool) *jsonfile.Store { return openOn(at, torn, &keychain{}) }

func TestStore(t *testing.T) {
	contracttest.Store(t, func(t *testing.T) contract.Store {
		return open(filepath.Join(t.TempDir(), "station", "devices.json"), false)
	})
}

func files() local.FS { return local.New(nil, nil, nil, nil).FS }

func TestABookWrittenBackFromBeforeAForgetIsNotTheAnchoredOne(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	k := &keychain{}
	if err := openOn(at, false, k).Change(t.Context(), add("laptop")); err != nil {
		t.Fatal(err)
	}
	before, err := files().ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if err = openOn(at, false, k).Change(t.Context(), func(b *devices.Book) error { b.Forget("laptop"); return nil }); err != nil {
		t.Fatal(err)
	}
	if err = files().WriteFile(at, before, 0o600); err != nil {
		t.Fatal(err)
	}
	book, head, err := openOn(at, false, k).Read(t.Context())
	if err != nil || len(book.Devices) != 1 || book.Anchored(head) {
		t.Fatalf("Read = %+v, %+v, %v; want the copy read and not anchored", book, head, err)
	}
	if got := devices.Check(book, head, "mini", nil); !got.Stale || len(got.Trusted) != 0 || len(got.Broken) != 1 {
		t.Fatalf("checked %+v; want every entry of the copy broken", got)
	}
	if err = openOn(at, false, k).Change(t.Context(), add("tablet")); err != nil {
		t.Fatal(err)
	}
	if book, head, err = openOn(at, false, k).Read(t.Context()); err != nil || !book.Anchored(head) || len(book.Devices) != 1 || book.Devices[0].Name != "tablet" {
		t.Fatalf("Read = %+v, %v; want a new book of tablet alone, anchored", book, err)
	}
}

func TestTheAnchorIsWrittenBeforeTheFile(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	k := &keychain{}
	if err := openOn(at, false, k).Change(t.Context(), add("mini")); err != nil {
		t.Fatal(err)
	}
	k.refused = errors.New("the keychain is locked")
	if err := openOn(at, false, k).Change(t.Context(), add("laptop")); !errors.Is(err, k.refused) {
		t.Fatalf("Change = %v; want the keychain's refusal", err)
	}
	book, head, err := openOn(at, false, k).Read(t.Context())
	if err != nil || !book.Anchored(head) || len(book.Devices) != 1 {
		t.Fatalf("Read = %+v, %v; want the book before, still anchored", book, err)
	}
	k.refused = nil
	if err = openOn(at, true, k).Change(t.Context(), add("laptop")); err == nil {
		t.Fatal("a torn write answered no error")
	}
	if book, head, err = openOn(at, false, k).Read(t.Context()); err != nil || book.Anchored(head) {
		t.Fatalf("Read = %+v, %+v, %v; want a write cut after the anchor to leave a book anchored nowhere", book, head, err)
	}
}

func TestAProcessReadsTheAnchorOnlyWhenTheBooksGenerationMoves(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	k := &keychain{}
	writer, reader := openOn(at, false, k), openOn(at, false, k)
	if err := writer.Change(t.Context(), add("mini")); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, _, err := reader.Read(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Change(t.Context(), add("laptop")); err != nil {
		t.Fatal(err)
	}
	reads := k.reads
	for range 3 {
		if _, _, err := writer.Read(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	book, head, err := reader.Read(t.Context())
	if err != nil || !book.Anchored(head) || len(book.Devices) != 2 {
		t.Fatalf("Read = %+v, %v; want the writer's book, anchored", book, err)
	}
	if k.reads != reads+1 {
		t.Fatalf("%d reads of the anchor after the change; want one, the reader's", k.reads-reads)
	}
}

func TestAnAnchorRolledBackBelowWhatThisProcessWroteIsNotBelieved(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	k := &keychain{}
	s := openOn(at, false, k)
	if err := s.Change(t.Context(), add("laptop")); err != nil {
		t.Fatal(err)
	}
	before, err := files().ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	old := k.head
	if err = s.Change(t.Context(), func(b *devices.Book) error { b.Forget("laptop"); return nil }); err != nil {
		t.Fatal(err)
	}
	if err = files().WriteFile(at, before, 0o600); err != nil {
		t.Fatal(err)
	}
	k.set(old)
	if book, head, err := s.Read(t.Context()); err != nil || book.Anchored(head) {
		t.Fatalf("Read = %+v, %+v, %v; want the book and the keychain rolled back together refused by the process that wrote past them", book, head, err)
	}
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
	got, _, err := open(at, false).Read(t.Context())
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
	if _, _, err := open(at, false).Read(t.Context()); !errors.As(err, &newer) {
		t.Fatalf("Read = %v; want a newer book refused", err)
	}
}

func TestAnAnchorThatCannotBeReadAnswersNoBook(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	if err := openOn(at, false, &keychain{}).Change(t.Context(), add("laptop")); err != nil {
		t.Fatal(err)
	}
	locked := errors.New("the keychain is locked")
	book, head, err := openOn(at, false, &keychain{locked: locked}).Read(t.Context())
	if !errors.Is(err, locked) || len(book.Devices) != 0 || head != (devices.Head{}) {
		t.Fatalf("Read = %+v, %+v, %v; want the keychain's refusal and no book", book, head, err)
	}
}
