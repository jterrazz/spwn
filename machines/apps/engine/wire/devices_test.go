package wire

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	devicesjsonfile "github.com/jterrazz/studio-station/apps/engine/devices/adapter/jsonfile"
	devicescalls "github.com/jterrazz/studio-station/apps/engine/devices/calls"
	devicescontract "github.com/jterrazz/studio-station/apps/engine/devices/contract"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

type piped struct {
	io.Reader
	io.Writer
	closers []io.Closer
}

func (p piped) Close() error {
	for _, c := range p.closers {
		_ = c.Close()
	}
	return nil
}

// value: the peer of an engine whose hello says the book it holds; nil is
// an engine that does not answer.
func peerOf(t *testing.T, held *devices.Head) devicesjsonfile.Peer {
	if held == nil {
		return enginePeer{engineAPI{c: api.NewClient(func(context.Context) (io.ReadWriteCloser, error) { return nil, errors.New("no socket") })}}
	}
	return peerSaying(t, engine.Hello{Book: held.String()})
}

func peerSaying(t *testing.T, said engine.Hello) devicesjsonfile.Peer {
	hello := call.Bind(call.Call[struct{}, struct{}, engine.Hello]{
		Name: engine.CallHello, Where: call.InEngine, Applies: coremachine.Anywhere,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (engine.Hello, error) {
			return said, nil
		},
	}, struct{}{})
	s := serve.Server{Calls: func(name string) (call.Handler, bool) { return hello, name == engine.CallHello }}
	return enginePeer{engineAPI{c: api.NewClient(func(context.Context) (io.ReadWriteCloser, error) {
		inR, inW := io.Pipe()
		outR, outW := io.Pipe()
		go s.Conn(t.Context(), piped{Reader: inR, Writer: outW, closers: []io.Closer{inR, outW}})
		return piped{Reader: outR, Writer: inW, closers: []io.Closer{outR, inW}}, nil
	})}}
}

// value: a login keychain in memory, keeping the book's head.
type memAnchor struct {
	mu   sync.Mutex
	head devices.Head
}

func (a *memAnchor) Read(context.Context) (devices.Head, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.head, nil
}

func (a *memAnchor) Write(_ context.Context, h devices.Head) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.head = h
	return nil
}

type lyingPeer struct{ head devices.Head }

func (p lyingPeer) Held(context.Context) (devices.Head, bool, error) { return p.head, true, nil }

// value: an engine's book that held laptop's sealed entry, then forgot it,
// and the book a process that runs `security` plants after: that entry at
// a later generation, the keychain's head set to it.
type planted struct {
	at     string
	anchor *memAnchor
	key    *machinesfake.DeviceKey
	laptop devices.Device
}

func bookWithLaptop(t *testing.T) planted {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	p := planted{at: filepath.Join(t.TempDir(), "station", "devices.json"), anchor: &memAnchor{}, key: machinesfake.DeviceKeyOf(priv)}
	p.laptop, err = devicescalls.Sealer{Key: p.key, Mac: "mini"}.Seal(t.Context(), devices.Device{Name: "laptop", Key: "ssh-ed25519 AAAAlaptop", Channels: devices.AllChannels(), Added: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (p planted) store(peer devicesjsonfile.Peer) *devicesjsonfile.Store {
	return devicesjsonfile.New(lockable{local.New(nil, nil, nil, nil).FS}, p.at, p.anchor, peer)
}

func (p planted) forged() devices.Book {
	return devices.Book{Generation: 7, Devices: []devices.Device{p.laptop}}
}

func (p planted) plant(t *testing.T) {
	t.Helper()
	forged := p.forged()
	data, err := json.Marshal(struct {
		V int `json:"v"`
		devices.Book
	}{V: devicesjsonfile.Version, Book: forged})
	if err != nil {
		t.Fatal(err)
	}
	if err = local.New(nil, nil, nil, nil).FS.WriteFile(p.at, data, 0o600); err != nil {
		t.Fatal(err)
	}
	_ = p.anchor.Write(t.Context(), forged.Head())
}

func (p planted) addThenForget(t *testing.T, s *devicesjsonfile.Store) {
	t.Helper()
	if err := s.Change(t.Context(), func(b *devices.Book) error { b.Devices = append(b.Devices, p.laptop); return nil }); err != nil {
		t.Fatal(err)
	}
	if err := s.Change(t.Context(), func(b *devices.Book) error { b.Forget("laptop"); return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestAVMIsGivenNoKeyOfABookPlantedUnderTheEngine(t *testing.T) {
	p := bookWithLaptop(t)
	engine := p.store(lyingPeer{p.forged().Head()})
	engine.Sole(t.Context())
	key := &heldKey{}
	key.once.Do(func() { key.key = p.key })
	m := machine{profile: coremachine.Profile{Kind: coremachine.Computer, Ref: coremachine.Ref{Name: "mini"}}, stores: &stores{devices: engine}, key: key}
	if err := engine.Change(t.Context(), func(b *devices.Book) error { b.Devices = append(b.Devices, p.laptop); return nil }); err != nil {
		t.Fatal(err)
	}
	keys := m.pipeKeys()
	if got := keys(); len(got) != 1 || got[0] != p.laptop.Key {
		t.Fatalf("the engine's own book gives %q; want laptop's key", got)
	}
	if err := engine.Change(t.Context(), func(b *devices.Book) error { b.Forget("laptop"); return nil }); err != nil {
		t.Fatal(err)
	}
	p.plant(t)
	if got := keys(); len(got) != 0 {
		t.Fatalf("a book planted under the engine gives %q; want no key", got)
	}
}

type runner struct{ run func(ctx context.Context) }

func (runner) Check() error { return nil }

func (r runner) Run(ctx context.Context, _ func(string)) error {
	r.run(ctx)
	return nil
}

func TestTheEngineRunsWithItsBookSole(t *testing.T) {
	p := bookWithLaptop(t)
	p.addThenForget(t, p.store(nil))
	book := p.store(nil)
	ran := false
	err := soleEngine{book: book, Engine: runner{func(ctx context.Context) {
		ran = true
		p.plant(t)
		if got, head, err := book.Read(ctx); err != nil || got.Anchored(head) {
			t.Errorf("the running engine read %+v, %+v, %v; want the book planted after its start not anchored", got, head, err)
		}
	}}}.Run(t.Context(), func(string) {})
	if err != nil || !ran {
		t.Fatalf("Run = %v, ran %v", err, ran)
	}
}

func TestTheHeadAProcessTakesIsTheOneTheRunningEngineHolds(t *testing.T) {
	if got, ok, err := peerOf(t, nil).Held(t.Context()); err != nil || ok || got != (devices.Head{}) {
		t.Fatalf("no engine answering: %+v, %v, %v; want nothing held and no error", got, ok, err)
	}
	head := devices.Book{Generation: 5, Devices: []devices.Device{{Name: "mini", Key: "ssh-ed25519 AAAAmini"}}}.Head()
	for _, want := range []devices.Head{{}, head} {
		if got, ok, err := peerOf(t, &want).Held(t.Context()); err != nil || !ok || got != want {
			t.Fatalf("an engine holding %+v: %+v, %v, %v; want it held", want, got, ok, err)
		}
	}
	var unreadable *devicescontract.AnchorUnreadableError
	if _, ok, err := peerSaying(t, engine.Hello{Book: head.String(), BookState: devices.Unchecked}).Held(t.Context()); ok || !errors.As(err, &unreadable) {
		t.Fatalf("an engine still waiting for the keychain: %v, %v; want the keychain unreadable here too", ok, err)
	}
}
