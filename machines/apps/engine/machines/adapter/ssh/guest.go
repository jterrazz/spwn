package ssh

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"net/netip"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const (
	keepalive = 15 * time.Second
	missed    = 3
)

type dialer struct {
	files   local.FS
	sockets local.Sockets
}

func New(files local.FS, sockets local.Sockets) contract.GuestDialer {
	return dialer{files: files, sockets: sockets}
}

type client struct {
	c    *ssh.Client
	done chan struct{}
	once sync.Once
}

// invariant: the VM's host key is checked against the same per-machine
// known_hosts file the CLI's ssh uses, accept-new: an absent entry is appended,
// a different key refuses.
func (d dialer) knownHosts(file string) (ssh.HostKeyCallback, error) {
	if err := d.files.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return nil, err
	}
	if _, err := d.files.Stat(file); errors.Is(err, fs.ErrNotExist) {
		if err := d.files.WriteFile(file, nil, 0o600); err != nil {
			return nil, err
		}
	}
	return knownhosts.New(file)
}

func (d dialer) pin(file, name string) func(ssh.PublicKey) error {
	return func(key ssh.PublicKey) error {
		known, err := d.files.ReadFile(file)
		if err != nil {
			return err
		}
		return d.files.WriteFile(file, append(known, knownhosts.Line([]string{name}, key)+"\n"...), 0o600)
	}
}

// invariant: a known_hosts file that cannot be read refuses at the
// handshake, so a VM that does not answer is told as such first.
func acceptNew[A any](check func(string, A, ssh.PublicKey) error, unreadable error, name string, pin func(ssh.PublicKey) error) func(string, A, ssh.PublicKey) error {
	return func(_ string, remote A, key ssh.PublicKey) error {
		if unreadable != nil {
			return unreadable
		}
		err := check(name+":22", remote, key)
		var unknown *knownhosts.KeyError
		if errors.As(err, &unknown) && len(unknown.Want) == 0 {
			return pin(key)
		}
		return err
	}
}

// invariant: a VM already pinned is asked only for the key types of its pins,
// so a pin OpenSSH made with one type (ed25519) is never refused because Go
// would negotiate another (ecdsa).
func (d dialer) pinnedAlgorithms(file, name string) []string {
	rest, err := d.files.ReadFile(file)
	if err != nil {
		return nil
	}
	var algos []string
	for len(rest) > 0 {
		var hosts []string
		var key ssh.PublicKey
		_, hosts, key, _, rest, err = ssh.ParseKnownHosts(rest)
		if err != nil {
			break
		}
		if !slices.Contains(hosts, name) && !slices.Contains(hosts, knownhosts.Normalize(name+":22")) {
			continue
		}
		switch t := key.Type(); t {
		case ssh.KeyAlgoRSA:
			algos = append(algos, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA)
		default:
			algos = append(algos, t)
		}
	}
	return algos
}

func (d dialer) auth(t contract.GuestTarget) (ssh.AuthMethod, error) {
	if t.Password != "" {
		return ssh.Password(t.Password), nil
	}
	key, err := d.files.ReadFile(t.Key)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

func (d dialer) Dial(ctx context.Context, t contract.GuestTarget) (contract.Guest, error) {
	method, err := d.auth(t)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:    t.User,
		Auth:    []ssh.AuthMethod{method},
		Timeout: 10 * time.Second,
	}
	// invariant: a build VM is a fresh clone of a base image: its host key is new
	// at every clone, so it is neither checked nor kept.
	if t.KnownHosts == "" {
		cfg.HostKeyCallback = ssh.InsecureIgnoreHostKey() //nolint:gosec // outside: the trust of a fresh clone is its image key, as above.
	} else {
		check, unreadable := d.knownHosts(t.KnownHosts)
		cfg.HostKeyCallback = acceptNew(check, unreadable, t.VM, d.pin(t.KnownHosts, t.VM))
		cfg.HostKeyAlgorithms = d.pinnedAlgorithms(t.KnownHosts, t.VM)
	}
	addr := t.Address.String()
	conn, err := d.sockets.DialTCP(ctx, addr, 10*time.Second)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		return nil, err
	}
	cl := &client{c: ssh.NewClient(c, chans, reqs), done: make(chan struct{})}
	go cl.watch()
	go cl.keepalive()
	return cl, nil
}

func (c *client) watch() {
	_ = c.c.Wait()
	c.once.Do(func() { close(c.done) })
}

func (c *client) keepalive() {
	t := time.NewTicker(keepalive)
	defer t.Stop()
	lost := 0
	for {
		select {
		case <-c.done:
			return
		case <-t.C:
			if _, _, err := c.c.SendRequest("keepalive@openssh.com", true, nil); err != nil {
				lost++
				if lost >= missed {
					_ = c.c.Close()
					return
				}
				continue
			}
			lost = 0
		}
	}
}

func (c *client) Exec(ctx context.Context, cmd contract.GuestCommand) (contract.GuestResult, error) {
	session, err := c.c.NewSession()
	if err != nil {
		return contract.GuestResult{}, err
	}
	defer func() { _ = session.Close() }()
	var out, errs bytes.Buffer
	session.Stdin, session.Stdout, session.Stderr = cmd.Stdin, cmd.Stdout, cmd.Stderr
	if cmd.Stdout == nil {
		session.Stdout = &out
	}
	if cmd.Stderr == nil {
		session.Stderr = &errs
	}
	if err := session.Start(cmd.Line); err != nil {
		return contract.GuestResult{}, err
	}
	ended := make(chan error, 1)
	go func() { ended <- session.Wait() }()
	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		_ = session.Close()
		return contract.GuestResult{}, ctx.Err()
	case err := <-ended:
		res := contract.GuestResult{Stdout: out.String(), Stderr: errs.String()}
		var exit *ssh.ExitError
		if errors.As(err, &exit) {
			res.Code = exit.ExitStatus()
			return res, nil
		}
		return res, err
	}
}

func quote(v string) string { return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'" }

func (c *client) Put(ctx context.Context, path string, data []byte, mode fs.FileMode) error {
	line := "/bin/cat > " + quote(path) + " && /bin/chmod " + strconv.FormatUint(uint64(mode.Perm()), 8) + " " + quote(path)
	res, err := c.Exec(ctx, contract.GuestCommand{Line: line, Stdin: bytes.NewReader(data)})
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return errors.New("writing " + path + " exited " + strconv.Itoa(res.Code) + ": " + strings.TrimSpace(res.Stderr))
	}
	return nil
}

func (c *client) Forward(ctx context.Context, ip string, port int) (io.ReadWriteCloser, error) {
	type result struct {
		conn io.ReadWriteCloser
		err  error
	}
	at, err := netip.ParseAddr(ip)
	if err != nil {
		return nil, err
	}
	addr := ip + ":" + strconv.Itoa(port)
	if at.Is6() {
		addr = "[" + ip + "]:" + strconv.Itoa(port)
	}
	got := make(chan result, 1)
	go func() {
		conn, err := c.c.Dial("tcp", addr)
		got <- result{conn, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-got:
		return r.conn, r.err
	}
}

func (c *client) Close() error          { return c.c.Close() }
func (c *client) Done() <-chan struct{} { return c.done }
