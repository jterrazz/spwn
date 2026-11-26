package jsonfile_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/adapter/jsonfile"
)

func TestAProcessAtTheLastGenerationTellsTwoBooksOfItApartByTheirDigest(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "devices.json")
	if err := files().MkdirAll(filepath.Dir(at), 0o700); err != nil {
		t.Fatal(err)
	}
	last := devices.Book{Generation: devices.MaxGeneration, Devices: []devices.Device{{Name: "laptop", Key: "ssh-ed25519 AAAAlaptop", Channels: devices.AllChannels()}}}
	first, err := json.Marshal(struct {
		V int `json:"v"`
		devices.Book
	}{V: jsonfile.Version, Book: last})
	if err != nil {
		t.Fatal(err)
	}
	if err = files().WriteFile(at, first, 0o600); err != nil {
		t.Fatal(err)
	}
	k := &keychain{}
	k.set(last.Head())
	reader := openOn(at, false, k)
	if book, head, readErr := reader.Read(t.Context()); readErr != nil || !book.Anchored(head) {
		t.Fatalf("Read = %+v, %+v, %v; want the book at the last generation anchored", book, head, readErr)
	}
	if err = openOn(at, false, k).Change(t.Context(), add("desk")); err != nil {
		t.Fatal(err)
	}
	book, head, err := reader.Read(t.Context())
	if err != nil || !book.Anchored(head) || len(book.Devices) != 2 || head.Generation != devices.MaxGeneration {
		t.Fatalf("Read = %+v, %+v, %v; want the next book, at the same generation, anchored", book, head, err)
	}
	if err = files().WriteFile(at, first, 0o600); err != nil {
		t.Fatal(err)
	}
	if book, head, err = reader.Read(t.Context()); err != nil || book.Anchored(head) {
		t.Fatalf("Read = %+v, %+v, %v; want the first book written back not anchored", book, head, err)
	}
}
