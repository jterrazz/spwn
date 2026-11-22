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

func keyedEngine(t *testing.T, deviceKey string, files map[string]string) (*Engine, *vmState, *filedMac, *heardStream) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	mac := &filedMac{fakeMac: h.mac, files: files}
	d := New(mac, h.hv, nil, h.m, &fakeTailnet{}, memStores(), Options{State: "/s", ThisMachine: catalog.Mac, DeviceKey: deviceKey})
	v, _ := d.find("api")
	heard := &heardStream{}
	d.mu.Lock()
	v.in, v.guest = heard, engine.GuestConnected
	d.mu.Unlock()
	return d, v, mac, heard
}

func TestAHostsKeysAreItsDeviceKeyFirstThenItsPersonalKeyThenItsAuthorizedKeysEachOnce(t *testing.T) {
	d, _, _, _ := keyedEngine(t, "/s/device-key.pub", map[string]string{
		"/s/device-key.pub":            deviceKeyLine + "\n",
		"/home/u/.ssh/id_ed25519.pub":  "ssh-ed25519 PERSONAL me@here\n",
		"/home/u/.ssh/authorized_keys": "# a comment\nssh-ed25519 PERSONAL me@here\nssh-ed25519 FAR far@there\n\n",
	})
	want := []string{deviceKeyLine, "ssh-ed25519 PERSONAL me@here", "ssh-ed25519 FAR far@there"}
	if got := d.hostKeys(); !slices.Equal(got, want) {
		t.Fatalf("keys %q, want %q", got, want)
	}
}

func TestAHostWithNoDeviceKeyFileSendsTheKeysItHadBefore(t *testing.T) {
	d, _, _, _ := keyedEngine(t, "", map[string]string{"/home/u/.ssh/id_ed25519.pub": "ssh-ed25519 PERSONAL me@here\n"})
	if got, want := d.hostKeys(), []string{"ssh-ed25519 PERSONAL me@here"}; !slices.Equal(got, want) {
		t.Fatalf("keys %q, want %q", got, want)
	}
}

func TestADeviceKeyWrittenAfterTheGuestConnectedReachesItOnTheNextWalk(t *testing.T) {
	d, v, mac, heard := keyedEngine(t, "/s/device-key.pub", map[string]string{})
	d.hello(v)
	mac.put("/s/device-key.pub", deviceKeyLine+"\n")
	d.rehello(v)
	said := heard.heard()
	if len(said) != 2 || strings.Contains(said[0], deviceKeyLine) || !strings.Contains(said[1], deviceKeyLine) {
		t.Fatalf("hellos %q, want the second alone to carry the device key", said)
	}
}
