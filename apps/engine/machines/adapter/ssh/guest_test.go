package ssh

import (
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
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"

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
		code := s.run(line, ch)
		status := make([]byte, 4)
		binary.BigEndian.PutUint32(status, uint32(code)) //nolint:gosec // outside: an exit code is 0-255.
		_, _ = ch.SendRequest("exit-status", false, status)
		return
	}
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
	if g, err := dial(t).Dial(t.Context(), target(t, again, known)); err == nil {
		_ = g.Close()
		t.Fatal("a VM answering with another host key was let in")
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

func dial(t *testing.T) contract.GuestDialer {
	t.Helper()
	h := local.New(nil, nil, io.Discard, io.Discard)
	return New(h.FS, h.Sockets)
}
