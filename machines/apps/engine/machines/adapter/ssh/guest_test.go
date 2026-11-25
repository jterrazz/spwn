package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"io"
	"io/fs"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

// invariant: the guest the adapter is proved against: an x/crypto sshd that
// accepts the test key or the image password, answers a programmed line,
// writes a file a put's shell line names into its own folder, and greets
// every direct-tcpip to GuestPort.
type sshd struct {
	t        *testing.T
	addr     netip.AddrPort
	dir      string
	mu       sync.Mutex
	answers  map[string]contracttest.Answer
	received map[string][]byte
	// value: a line that runs until the client ends it, what it writes
	// first, and what the guest heard of that end, in order: a signal by its
	// name, or the session's close.
	held    string
	heldOut string
	began   chan struct{}
	heard   chan string
}

func newHostKey(t *testing.T) ssh.Signer {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func startSSHD(t *testing.T, host ssh.Signer) *sshd {
	t.Helper()
	s := &sshd{t: t, dir: t.TempDir(), answers: map[string]contracttest.Answer{}, received: map[string][]byte{}}
	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) { return nil, nil },
		PasswordCallback: func(_ ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
			if string(pw) == "admin" {
				return nil, nil
			}
			return nil, errors.New("wrong password")
		},
	}
	cfg.AddHostKey(host)
	l := localtest.Loopback(t)
	s.addr = netip.MustParseAddrPort(l.Addr().String())
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				_, chans, reqs, err := ssh.NewServerConn(conn, cfg)
				if err == nil {
					s.serve(chans, reqs)
				}
			}()
		}
	}()
	return s
}

func (s *sshd) serve(chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request) {
	go ssh.DiscardRequests(reqs)
	for nc := range chans {
		switch nc.ChannelType() {
		case "session":
			go s.session(nc)
		case "direct-tcpip":
			go s.forward(nc)
		default:
			_ = nc.Reject(ssh.UnknownChannelType, "no")
		}
	}
}

func (s *sshd) session(nc ssh.NewChannel) {
	ch, reqs, err := nc.Accept()
	if err != nil {
		return
	}
	defer func() { _ = ch.Close() }()
	for req := range reqs {
		if req.Type != "exec" {
			_ = req.Reply(false, nil)
			continue
		}
		_ = req.Reply(true, nil)
		line := string(req.Payload[4:])
		if line == s.held && line != "" {
			s.hold(ch, reqs)
			return
		}
		code := s.run(line, ch)
		status := make([]byte, 4)
		binary.BigEndian.PutUint32(status, uint32(code)) //nolint:gosec // outside: an exit code is 0-255.
		_, _ = ch.SendRequest("exit-status", false, status)
		return
	}
}

func (s *sshd) hold(ch ssh.Channel, reqs <-chan *ssh.Request) {
	_, _ = ch.Write([]byte(s.heldOut))
	close(s.began)
	for req := range reqs {
		if req.Type != "signal" || len(req.Payload) <= 4 {
			continue
		}
		s.heard <- "signal " + string(req.Payload[4:])
		status := make([]byte, 4)
		binary.BigEndian.PutUint32(status, 143)
		_, _ = ch.SendRequest("exit-status", false, status)
		return
	}
	s.heard <- "closed"
}

var putLine = regexp.MustCompile(`^/bin/cat > '([^']+)' && /bin/chmod ([0-7]+) '([^']+)'$`)

func (s *sshd) run(line string, ch ssh.Channel) int {
	s.mu.Lock()
	a, ok := s.answers[line]
	s.mu.Unlock()
	if ok {
		in, _ := io.ReadAll(ch)
		s.mu.Lock()
		s.received[line] = in
		s.mu.Unlock()
		_, _ = ch.Write([]byte(a.Stdout))
		_, _ = ch.Stderr().Write([]byte(a.Stderr))
		return a.Code
	}
	m := putLine.FindStringSubmatch(line)
	if len(m) != 4 || m[1] != m[3] {
		_, _ = ch.Stderr().Write([]byte(line + ": not found\n"))
		return 127
	}
	data, err := io.ReadAll(ch)
	if err != nil {
		return 1
	}
	mode, err := strconv.ParseUint(m[2], 8, 32)
	if err != nil {
		return 1
	}
	if err := os.WriteFile(filepath.Join(s.dir, m[1]), data, fs.FileMode(mode)); err != nil {
		return 1
	}
	return 0
}

func (s *sshd) forward(nc ssh.NewChannel) {
	var target struct {
		Host       string
		Port       uint32
		OriginHost string
		OriginPort uint32
	}
	if err := ssh.Unmarshal(nc.ExtraData(), &target); err != nil || target.Port != contracttest.GuestPort {
		_ = nc.Reject(ssh.ConnectionFailed, "nothing listens")
		return
	}
	ch, reqs, err := nc.Accept()
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)
	defer func() { _ = ch.Close() }()
	_, _ = ch.Write([]byte(contracttest.Greeting + "\n"))
	_, _ = io.ReadAll(ch)
}

func clientKey(t *testing.T) string {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(key, "")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(file, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

func target(t *testing.T, s *sshd, known string) contract.GuestTarget {
	return contract.GuestTarget{VM: "dev", Address: s.addr, User: "dev", Key: clientKey(t), KnownHosts: known}
}

func TestTheSSHGuestKeepsTheContract(t *testing.T) {
	contracttest.Guest(t, func(t *testing.T) contracttest.GuestSubject {
		s := startSSHD(t, newHostKey(t))
		known := filepath.Join(t.TempDir(), "known_hosts", "dev")
		return contracttest.GuestSubject{
			Dialer: dial(t),
			Target: func(t *testing.T) contract.GuestTarget { return target(t, s, known) },
			Program: func(line string, a contracttest.Answer) {
				s.mu.Lock()
				defer s.mu.Unlock()
				s.answers[line] = a
			},
			Received: func(line string) []byte {
				s.mu.Lock()
				defer s.mu.Unlock()
				return s.received[line]
			},
			Read: func(t *testing.T, path string) ([]byte, fs.FileMode) {
				file := filepath.Join(s.dir, path)
				data, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				entries, err := os.ReadDir(filepath.Dir(file))
				if err != nil {
					t.Fatal(err)
				}
				for _, e := range entries {
					if e.Name() == filepath.Base(file) {
						info, err := e.Info()
						if err != nil {
							t.Fatal(err)
						}
						return data, info.Mode()
					}
				}
				t.Fatalf("%s is not in its folder", path)
				return nil, 0
			},
			Path: "station",
		}
	})
}

func TestAVMWhoseHostKeyChangedIsRefused(t *testing.T) {
	known := filepath.Join(t.TempDir(), "known_hosts", "dev")
	first := startSSHD(t, newHostKey(t))
	g, err := dial(t).Dial(t.Context(), target(t, first, known))
	if err != nil {
		t.Fatalf("the first sight of the VM was refused: %v", err)
	}
	_ = g.Close()
	again := startSSHD(t, newHostKey(t))
	g, err = dial(t).Dial(t.Context(), target(t, again, known))
	if err == nil {
		_ = g.Close()
		t.Fatal("a VM answering with another host key was let in")
	}
	if !errors.Is(err, contract.ErrHostKeyChanged) {
		t.Errorf("a VM answering with another host key came back %v, want ErrHostKeyChanged", err)
	}
}

func TestABuildVMIsReachedByItsImagePassword(t *testing.T) {
	s := startSSHD(t, newHostKey(t))
	s.answers["true"] = contracttest.Answer{}
	g, err := dial(t).Dial(t.Context(), contract.GuestTarget{VM: "build", Address: s.addr, User: "admin", Password: "admin"})
	if err != nil {
		t.Fatalf("the image password was refused: %v", err)
	}
	defer func() { _ = g.Close() }()
	if res, err := g.Exec(t.Context(), contract.GuestCommand{Line: "true"}); err != nil || res.Code != 0 {
		t.Errorf("a command over the password session came back %+v (%v)", res, err)
	}
}

func TestAGuestThatTakesNoneOfWhatIsOfferedIsAToldRefusal(t *testing.T) {
	s := startSSHD(t, newHostKey(t))
	_, err := dial(t).Dial(t.Context(), contract.GuestTarget{VM: "build", Address: s.addr, User: "admin", Password: "not-admin"})
	if !errors.Is(err, contract.ErrLoginRefused) {
		t.Fatalf("a refused password came back %v, want ErrLoginRefused", err)
	}
}

func TestACommandEndedByItsContextIsToldToStopBeforeItsSessionCloses(t *testing.T) {
	// invariant: with no terminal, a guest's command outlives its closed
	// session (the probe of 2026-11-14 left a login polling), so the
	// session's end is a SIGTERM first.
	s := startSSHD(t, newHostKey(t))
	s.held, s.began, s.heard = "wait-for-approval", make(chan struct{}), make(chan string, 2)
	g, err := dial(t).Dial(t.Context(), target(t, s, filepath.Join(t.TempDir(), "known_hosts", "dev")))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = g.Close() }()
	ctx, cancel := context.WithCancel(t.Context())
	ran := make(chan error, 1)
	go func() {
		_, err := g.Exec(ctx, contract.GuestCommand{Line: s.held, Stdout: io.Discard, Stderr: io.Discard})
		ran <- err
	}()
	select {
	case <-s.began:
	case <-time.NewTimer(10 * time.Second).C:
		t.Fatal("the command never began")
	}
	cancel()
	select {
	case heard := <-s.heard:
		if heard != "signal TERM" {
			t.Errorf("the guest heard %q first; want the SIGTERM before the session closes", heard)
		}
	case <-time.NewTimer(10 * time.Second).C:
		t.Fatal("the guest heard nothing of the end")
	}
	select {
	case err := <-ran:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Exec = %v; want the context's end", err)
		}
	case <-time.NewTimer(10 * time.Second).C:
		t.Fatal("Exec never returned")
	}
}

func TestACommandEndedByItsContextAnswersTheOutputItWroteBefore(t *testing.T) {
	s := startSSHD(t, newHostKey(t))
	s.held, s.heldOut, s.began, s.heard = "lockdown", "ssh-ed25519 AAAA printed\n", make(chan struct{}), make(chan string, 2)
	g, err := dial(t).Dial(t.Context(), target(t, s, filepath.Join(t.TempDir(), "known_hosts", "dev")))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = g.Close() }()
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-s.began
		cancel()
	}()
	res, err := g.Exec(ctx, contract.GuestCommand{Line: s.held})
	if !errors.Is(err, context.Canceled) || res.Stdout != s.heldOut {
		t.Errorf("Exec = %q, %v; want the line it printed and the context's end", res.Stdout, err)
	}
}

func dial(t *testing.T) contract.GuestDialer {
	t.Helper()
	h := local.New(nil, nil, io.Discard, io.Discard)
	return New(h.FS, h.Sockets, "darwin")
}

func dialRefused(t *testing.T, goos string, err error) error {
	t.Helper()
	h := local.New(nil, nil, io.Discard, io.Discard)
	_, got := New(h.FS, localtest.Refusing(h.Sockets, err), goos).Dial(context.Background(), contract.GuestTarget{
		VM: "linux-2", Address: netip.MustParseAddrPort("192.168.64.3:22"), User: "admin", Password: "admin",
	})
	if got == nil {
		t.Fatal("Dial answered; want its refusal")
	}
	return got
}

func TestADialTheMacRefusesForLocalNetworkSaysSo(t *testing.T) {
	err := dialRefused(t, "darwin", localtest.Unreachable())
	if !errors.As(err, new(*contract.NoRouteError)) {
		t.Fatalf("Dial = %v; want the no-route refusal", err)
	}
	if strings.Contains(err.Error(), "approved") || !strings.Contains(err.Error(), "no route to host") {
		t.Errorf("Dial = %v; want the dial's own error alone, naming no cause", err)
	}
}

func TestAnUnreachableRouteOffAMacIsNoLocalNetworkBlock(t *testing.T) {
	if err := dialRefused(t, "linux", localtest.Unreachable()); errors.As(err, new(*contract.NoRouteError)) {
		t.Fatalf("Dial on linux = %v; want a plain refusal", err)
	}
}

func TestARefusedDialOnAMacIsNoLocalNetworkBlock(t *testing.T) {
	if err := dialRefused(t, "darwin", localtest.Refused()); errors.As(err, new(*contract.NoRouteError)) {
		t.Fatalf("Dial = %v; want a plain refusal", err)
	}
}

func TestAVMRepinnedToTheHostKeyItsLockdownPrintedIsLetInByThatKeyAlone(t *testing.T) {
	known := filepath.Join(t.TempDir(), "known_hosts", "dev")
	first := startSSHD(t, newHostKey(t))
	g, err := dial(t).Dial(t.Context(), target(t, first, known))
	if err != nil {
		t.Fatalf("the first sight of the VM was refused: %v", err)
	}
	_ = g.Close()
	made := newHostKey(t)
	h := local.New(nil, nil, io.Discard, io.Discard)
	if pinned := Repin(h.FS, known, "dev", string(ssh.MarshalAuthorizedKey(made.PublicKey()))); pinned != nil {
		t.Fatalf("the printed host key was not pinned: %v", pinned)
	}
	again := startSSHD(t, made)
	g, err = dial(t).Dial(t.Context(), target(t, again, known))
	if err != nil {
		t.Fatalf("the VM answering with the host key its lockdown printed was refused: %v", err)
	}
	_ = g.Close()
	if g, err := dial(t).Dial(t.Context(), target(t, first, known)); err == nil {
		_ = g.Close()
		t.Fatal("the VM's host key before its lockdown is still let in")
	}
	if Repin(h.FS, known, "dev", "not a key") == nil {
		t.Fatal("a printed line that is no host key was pinned")
	}
}

func TestARepinReplacesTheVMsLinesAndKeepsEveryOtherHosts(t *testing.T) {
	known := filepath.Join(t.TempDir(), "known_hosts")
	old, other, made := newHostKey(t).PublicKey(), newHostKey(t).PublicKey(), newHostKey(t).PublicKey()
	others := knownhosts.Line([]string{"build"}, other) + "\n" + "# a note the owner wrote\n" + knownhosts.Line([]string{"[dev]:2222"}, other) + "\n"
	before := knownhosts.Line([]string{"dev"}, old) + "\n" + others + knownhosts.Line([]string{"dev"}, other) + "\n"
	if err := os.WriteFile(known, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	h := local.New(nil, nil, io.Discard, io.Discard)
	if err := Repin(h.FS, known, "dev", string(ssh.MarshalAuthorizedKey(made))); err != nil {
		t.Fatalf("the printed host key was not pinned: %v", err)
	}
	after, err := os.ReadFile(known)
	if err != nil {
		t.Fatal(err)
	}
	if want := others + knownhosts.Line([]string{"dev"}, made) + "\n"; string(after) != want {
		t.Fatalf("known hosts after the repin:\n%s\nwant every other host's line kept and dev's one key last:\n%s", after, want)
	}
}

func TestARepinTakesOneLineHoldingOneKeyAndNothingElse(t *testing.T) {
	key := strings.TrimSuffix(string(ssh.MarshalAuthorizedKey(newHostKey(t).PublicKey())), "\n")
	h := local.New(nil, nil, io.Discard, io.Discard)
	for name, printed := range map[string]string{
		"two keys":           key + "\n" + key + "\n",
		"a key then a line":  key + "\nsomething else\n",
		"a blank line first": "\n" + key + "\n",
		"a blank line after": key + "\n\n",
		"no key":             "not a key\n",
		"nothing":            "\n",
	} {
		known := filepath.Join(t.TempDir(), "known_hosts")
		if err := os.WriteFile(known, []byte("kept\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if Repin(h.FS, known, "dev", printed) == nil {
			t.Errorf("%s: printed %q was pinned", name, printed)
		}
		if got, _ := os.ReadFile(known); string(got) != "kept\n" {
			t.Errorf("%s: a refused print changed known hosts to %q", name, got)
		}
	}
	for _, printed := range []string{key + "\n", key} {
		if err := Repin(h.FS, filepath.Join(t.TempDir(), "known_hosts"), "dev", printed); err != nil {
			t.Errorf("one key line %q was refused: %v", printed, err)
		}
	}
}

func TestARepinTakesTheLineARealLockdownPrintedBeforeItWasKilled(t *testing.T) {
	// value: what macos-firstlogin.sh prints on the pinned macos-tahoe-base
	// (macOS 26.6.2, 25G83) when it is killed right after its print.
	printed := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILfppDqoZeh8VT6wKD6cvMPGhBzgZ2SaaTVigCblfOlJ root@Manageds-Virtual-Machine.local\n"
	known := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(known, []byte("dev ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMYRnPRb0woo5J/IbiUEUkSJ8kxETUFsKOWJfNno6nc5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Repin(local.New(nil, nil, io.Discard, io.Discard).FS, known, "dev", printed); err != nil {
		t.Fatalf("the real print was refused: %v", err)
	}
	if got, _ := os.ReadFile(known); string(got) != "dev ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILfppDqoZeh8VT6wKD6cvMPGhBzgZ2SaaTVigCblfOlJ\n" {
		t.Errorf("known hosts hold %q; want the printed key alone", got)
	}
}
