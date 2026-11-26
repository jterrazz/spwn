package jsonfile_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
)

func garble(t *testing.T, at string) {
	t.Helper()
	if err := files().MkdirAll(filepath.Dir(at), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := files().WriteFile(at, []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestABookTheEngineCannotReadReadsEmptyAndIsRepairedByAddingItsDevicesAgain(t *testing.T) {
	for name, unreadable := range map[string]func(*testing.T, *jsonfile.Store, string){
		"at its start": func(t *testing.T, engine *jsonfile.Store, at string) {
			garble(t, at)
			engine.Sole(t.Context())
		},
		"since its start": func(t *testing.T, engine *jsonfile.Store, at string) {
			engine.Sole(t.Context())
			if err := engine.Change(t.Context(), add("laptop")); err != nil {
				t.Fatal(err)
			}
			garble(t, at)
		},
	} {
		t.Run(name, func(t *testing.T) {
			at := filepath.Join(t.TempDir(), "station", "devices.json")
			k := &keychain{}
			engine := openOn(at, false, k)
			unreadable(t, engine, at)
			if _, standing, _ := engine.Held(t.Context()); standing != devices.Changed {
				t.Fatalf("held %q; want the unreadable book said changed outside Spwn", standing)
			}
			book, _, err := engine.Read(t.Context())
			if _, found := book.Find("laptop"); err != nil || found || len(book.Devices) != 0 {
				t.Fatalf("Read = %+v, %v; want an empty book, so forget finds no laptop", book, err)
			}
			if err = engine.Change(t.Context(), add("desk")); err != nil {
				t.Fatalf("add desk = %v; want it to start an empty book", err)
			}
			book, head, err := engine.Read(t.Context())
			if err != nil || !book.Anchored(head) || len(book.Devices) != 1 || book.Devices[0].Name != "desk" || k.head != head {
				t.Fatalf("Read = %+v, %+v, %v; want desk alone, anchored in the keychain", book, head, err)
			}
			if _, standing, _ := engine.Held(t.Context()); standing != devices.Kept {
				t.Fatalf("held %q after the add; want the book the engine wrote", standing)
			}
		})
	}
}

func TestABookUnreadableWhileTheEngineAwaitsItsKeychainIsRefusedToAddAndForget(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	k := &keychain{}
	forgottenLaptop(t, at, k)
	head := k.head
	k.locked = errors.New("the keychain is locked")
	engine := openOn(at, false, k)
	engine.Sole(t.Context())
	garble(t, at)
	var unreadable *store.UnreadableError
	if _, _, err := engine.Read(t.Context()); !errors.As(err, &unreadable) {
		t.Fatalf("Read while the engine awaits = %v; want the book refused as unreadable", err)
	}
	forget := func(b *devices.Book) error { b.Forget("laptop"); return nil }
	for name, edit := range map[string]func(*devices.Book) error{"add": add("tablet"), "forget": forget} {
		if err := engine.Change(t.Context(), edit); !errors.As(err, &unreadable) {
			t.Fatalf("%s while the engine awaits = %v; want the book refused as unreadable", name, err)
		}
	}
	if data, err := files().ReadFile(at); err != nil || !bytes.Equal(data, []byte("garbage")) || k.head != head {
		t.Fatalf("the book or the keychain moved (%q, %v, head %+v); want nothing written", data, err, k.head)
	}
}

func TestAnEngineHoldingNoHeadStartsTheOwnersAddFromAnEmptyBookWhateverBookWasPlanted(t *testing.T) {
	locked := errors.New("the keychain is locked")
	for name, unheld := range map[string]func(*testing.T, *jsonfile.Store, string, *keychain){
		"its book unreadable at its start": func(t *testing.T, engine *jsonfile.Store, at string, _ *keychain) {
			garble(t, at)
			engine.Sole(t.Context())
		},
		"its keychain head malformed at its start": func(t *testing.T, engine *jsonfile.Store, _ string, k *keychain) {
			k.locked = &contract.AnchorUnreadableError{Cause: "a head Spwn never writes", Malformed: true}
			engine.Sole(t.Context())
			k.locked = nil
		},
		"its wait met another book": func(t *testing.T, engine *jsonfile.Store, at string, k *keychain) {
			k.locked = locked
			engine.Sole(t.Context())
			if err := files().WriteFile(at, []byte(`{"generation": 9, "devices": []}`), 0o600); err != nil {
				t.Fatal(err)
			}
			k.locked = nil
			trustsNothing(t, engine, devices.Changed)
		},
	} {
		t.Run(name, func(t *testing.T) {
			at := filepath.Join(t.TempDir(), "station", "devices.json")
			k := &keychain{}
			withLaptop, headLaptop := forgottenLaptop(t, at, k)
			engine := openOn(at, false, k)
			unheld(t, engine, at, k)
			plant(t, at, k, withLaptop, headLaptop)
			trustsNothing(t, engine, devices.Changed)
			if err := engine.Change(t.Context(), add("phone")); err != nil {
				t.Fatalf("add phone = %v; want it to start a book", err)
			}
			book, head, err := engine.Read(t.Context())
			if err != nil || !book.Anchored(head) || k.head != head {
				t.Fatalf("Read = %+v, %+v, %v; want the engine's own write, anchored in the keychain", book, head, err)
			}
			if len(book.Devices) != 1 || book.Devices[0].Name != "phone" {
				t.Fatalf("the book after the add holds %+v; want phone alone, nothing of the planted book", book.Devices)
			}
		})
	}
}

func TestABookUnreadableIsRefusedToAProcessThatIsNotTheEngine(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	k := &keychain{}
	engine := openOn(at, false, k)
	engine.Sole(t.Context())
	if err := engine.Change(t.Context(), add("laptop")); err != nil {
		t.Fatal(err)
	}
	garble(t, at)
	cli := jsonfile.New(holding{FS: files()}, at, k, engineOf{engine})
	var unreadable *store.UnreadableError
	if _, _, err := cli.Read(t.Context()); !errors.As(err, &unreadable) {
		t.Fatalf("Read = %v; want the book refused as unreadable", err)
	}
	if err := cli.Change(t.Context(), add("desk")); !errors.As(err, &unreadable) {
		t.Fatalf("Change = %v; want the book refused as unreadable", err)
	}
}
