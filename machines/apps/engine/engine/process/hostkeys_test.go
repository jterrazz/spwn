package process

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	model "github.com/jterrazz/studio-station/apps/engine/machines"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

// value: a Mac whose files are the ones a test writes, by path.
type filedMac struct {
	*fakeMac
	mu    sync.Mutex
	files map[string]string
}

func (m *filedMac) ReadFile(p string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.files[p]; ok {
		return []byte(s), nil
	}
	return nil, errNoFile
}

func (m *filedMac) put(p, s string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[p] = s
}

// value: a guest's input that keeps every line written to it.
type heardStream struct {
	mu    sync.Mutex
	lines []string
}

func (*heardStream) Read([]byte) (int, error) { return 0, errors.New("nothing said") }
func (*heardStream) Close() error             { return nil }
func (s *heardStream) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lines = append(s.lines, strings.TrimSpace(string(p)))
	return len(p), nil
}

func (s *heardStream) heard() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.lines...)
}

var errNoFile = errors.New("no such file")

const deviceKeyLine = "ecdsa-sha2-nistp256 AAAAE2FIXTURE station device key"

func keyedEngine(t *testing.T, deviceKey func() string, files map[string]string, book func() []string) (*Engine, *vmState, *filedMac, *heardStream) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	mac := &filedMac{fakeMac: h.mac, files: files}
	d := New(mac, h.hv, nil, h.m, &fakeTailnet{}, memStores(), Options{State: "/s", ThisMachine: catalog.Mac, DeviceKey: deviceKey, BookKeys: book})
	v, _ := d.find("api")
	heard := &heardStream{}
	d.mu.Lock()
	v.in, v.guest = heard, engine.GuestConnected
	d.mu.Unlock()
	return d, v, mac, heard
}

func TestAHostsKeysAreItsDeviceKeyThenItsBooksEachOnceAndNoKeyOfTheMacsOwn(t *testing.T) {
	book := func() []string { return []string{"ssh-ed25519 LAPTOP", "ssh-ed25519 LAPTOP", "ssh-ed25519 PHONE"} }
	d, _, _, _ := keyedEngine(t, func() string { return deviceKeyLine }, map[string]string{
		"/s/device-key.pub":            "ecdsa-sha2-nistp256 AAAAE2WRITTENBYANOTHERPROCESS\n",
		"/home/u/.ssh/id_ed25519.pub":  "ssh-ed25519 PERSONAL me@here\n",
		"/home/u/.ssh/authorized_keys": "ssh-ed25519 FAR far@there\n",
	}, book)
	want := []string{deviceKeyLine, "ssh-ed25519 LAPTOP", "ssh-ed25519 PHONE"}
	if got := d.hostKeys(); !slices.Equal(got, want) {
		t.Fatalf("keys %q, want %q", got, want)
	}
}

func TestAHostWithNoDeviceKeyAndNoBookTrustsNothing(t *testing.T) {
	d, _, _, _ := keyedEngine(t, func() string { return "" }, map[string]string{
		"/s/device-key.pub":           "ecdsa-sha2-nistp256 AAAAE2WRITTENBYANOTHERPROCESS\n",
		"/home/u/.ssh/id_ed25519.pub": "ssh-ed25519 PERSONAL me@here\n",
	}, nil)
	if got := d.hostKeys(); len(got) != 0 {
		t.Fatalf("keys %q, want none", got)
	}
}

func TestABookChangeReachesEveryConnectedGuestAtOnce(t *testing.T) {
	kept := []string{"ssh-ed25519 LAPTOP"}
	d, v, _, heard := keyedEngine(t, nil, map[string]string{}, func() []string { return kept })
	d.hello(v)
	kept = nil
	d.Rehello()
	said := heard.heard()
	if len(said) != 2 || !strings.Contains(said[0], "LAPTOP") || strings.Contains(said[1], "LAPTOP") {
		t.Fatalf("hellos %q, want the second to drop the forgotten key", said)
	}
}

func TestADeviceKeyMadeAfterTheGuestConnectedReachesItOnTheNextWalk(t *testing.T) {
	var made string
	d, v, mac, heard := keyedEngine(t, func() string { return made }, map[string]string{}, nil)
	d.hello(v)
	mac.put("/s/device-key.pub", "ecdsa-sha2-nistp256 AAAAE2WRITTENBYANOTHERPROCESS\n")
	d.rehello(v)
	made = deviceKeyLine
	d.rehello(v)
	said := heard.heard()
	if len(said) != 2 || strings.Contains(said[0], "AAAAE2") || !strings.Contains(said[1], deviceKeyLine) {
		t.Fatalf("hellos %q, want the file written by another process never sent, and the key once made", said)
	}
}
