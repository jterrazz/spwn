package wire

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/ssh"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// invariant: the one way into a VM: ssh, or the world's guests in a spec
// build.
func (m machine) guestDialer() contract.GuestDialer {
	if m.world != nil {
		return m.world.guests()
	}
	return ssh.New(m.sys.FS, m.sys.Sockets, m.sys.Env.GOOS())
}

type engineGuests struct {
	dialer contract.GuestDialer
	vms    *machinescalls.Machinist
	files  local.FS
}

func (g engineGuests) Dial(ctx context.Context, vm string, at netip.AddrPort) (process.Guest, error) {
	c, err := g.dial(ctx, g.vms.GuestTarget(vm, at))
	if err != nil {
		return nil, err
	}
	return engineGuest{c}, nil
}

// value: how long a lockdown's script may run in its guest.
const lockdownWait = 3 * time.Minute

// invariant: a dial the VM's first login answers (machinescalls.FirstLoginOf)
// runs its lockdown as the image's own account, then dials by the key
// again; a key dial to a VM whose image carries a lockdown is kept only
// once its marker holds, the lockdown run again over it when it does not.
// A lockdown that is refused, cannot run or fails is a
// contract.LockdownError, the guest probed for a login a start could run it
// again by; so is a host key other than the one pinned, on such a VM, which
// no start cures. What could not be read for a lockdown is its own error, and
// a dial the guest did not answer stays the dial's own.
func (g engineGuests) dial(ctx context.Context, t contract.GuestTarget) (contract.Guest, error) {
	c, err := g.dialer.Dial(ctx, t)
	if errors.Is(err, contract.ErrHostKeyChanged) {
		if locks, read := g.vms.Locks(ctx, t.VM); read == nil && locks {
			return nil, &contract.LockdownError{Cause: "it answers with a host key other than the one its lockdown printed", Gone: true}
		}
	}
	if err != nil {
		first, ok, why := g.vms.FirstLoginOf(ctx, t, err)
		if why != nil {
			return nil, why
		}
		if !ok {
			return nil, err
		}
		if ran := g.firstLogin(ctx, first, t); ran != nil {
			return nil, ran
		}
		if c, err = g.dialer.Dial(ctx, t); err != nil {
			return nil, g.failed(ctx, t, first, errors.New("its lockdown ended, and its key is refused: "+err.Error()))
		}
	}
	if held := g.locked(ctx, c, t); held != nil {
		_ = c.Close()
		return nil, held
	}
	return c, nil
}

func (g engineGuests) firstLogin(ctx context.Context, l machinescalls.Lockdown, t contract.GuestTarget) error {
	c, err := g.dialer.Dial(ctx, l.Login)
	if errors.Is(err, contract.ErrLoginRefused) {
		return &contract.LockdownError{Cause: "it takes neither station's key nor its image's login " + l.Login.User, Gone: true}
	}
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	if ran := g.lockdown(ctx, c, l, l.Login); ran != nil {
		return g.failed(ctx, t, l, ran)
	}
	return nil
}

// value: a lockdown's failure, Gone when the guest now refuses both station's
// key and its image's login, or answers either with another host key.
func (g engineGuests) failed(ctx context.Context, t contract.GuestTarget, l machinescalls.Lockdown, cause error) error {
	gone := true
	for _, by := range []contract.GuestTarget{t, l.Login} {
		c, err := g.dialer.Dial(ctx, by)
		if err == nil {
			_ = c.Close()
		}
		if !errors.Is(err, contract.ErrLoginRefused) && !errors.Is(err, contract.ErrHostKeyChanged) {
			gone = false
		}
	}
	why := cause.Error()
	if gone {
		why += "; it now takes neither station's key nor its image's login"
	}
	return &contract.LockdownError{Cause: why, Gone: gone}
}

func (g engineGuests) locked(ctx context.Context, c contract.Guest, t contract.GuestTarget) error {
	if locks, err := g.vms.Locks(ctx, t.VM); err != nil || !locks {
		return err
	}
	res, err := c.Exec(ctx, contract.GuestCommand{Line: catalog.LockedLine})
	if err != nil {
		return err
	}
	if res.Code == 0 {
		return nil
	}
	l, _, err := g.vms.LockdownOf(ctx, t)
	if err != nil {
		return err
	}
	if ran := g.lockdown(ctx, c, l, t); ran != nil {
		return &contract.LockdownError{Cause: "its lockdown is unfinished, and running it again failed: " + ran.Error()}
	}
	return nil
}

// invariant: a lockdown makes the guest's host keys anew and prints its
// ed25519 one as soon as it is made, which replaces every key the VM's
// known hosts held, whether the lockdown ends 0, fails, or its session
// errors or ends past lockdownWait once that line came; its script runs
// lockdownWait at most.
func (g engineGuests) lockdown(ctx context.Context, c contract.Guest, l machinescalls.Lockdown, t contract.GuestTarget) error {
	made, err := c.Exec(ctx, contract.GuestCommand{Line: catalog.LockdownFolder})
	if err != nil {
		return err
	}
	dir, err := l.Folder(made)
	if err != nil {
		return err
	}
	if put := c.Put(ctx, dir+"/station", l.Binary, 0o755); put != nil {
		return put
	}
	if put := c.Put(ctx, dir+"/lockdown.sh", l.Script, 0o600); put != nil {
		return put
	}
	run, cancel := context.WithTimeout(ctx, lockdownWait)
	defer cancel()
	res, err := c.Exec(run, contract.GuestCommand{Line: catalog.LockdownLine(dir), Stdin: strings.NewReader(l.Values)})
	var pinned error
	if strings.TrimSpace(res.Stdout) != "" {
		pinned = ssh.Repin(g.files, t.KnownHosts, t.VM, res.Stdout)
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return errors.Join(errors.New("its lockdown did not finish in "+strconv.Itoa(int(lockdownWait.Minutes()))+" min"), pinned)
	case err != nil:
		return errors.Join(err, pinned)
	case pinned != nil:
		return errors.Join(l.Failed(res.Code, res.Stderr), pinned)
	case strings.TrimSpace(res.Stdout) == "" && res.Code == 0:
		return errors.New("its lockdown printed no host key")
	}
	return l.Failed(res.Code, res.Stderr)
}

type engineGuest struct{ contract.Guest }

func (g engineGuest) Exec(ctx context.Context, line, stdin string) (engine.GuestRan, error) {
	res, err := g.Guest.Exec(ctx, contract.GuestCommand{Line: line, Stdin: strings.NewReader(stdin)})
	return engine.GuestRan{Code: res.Code, Stdout: res.Stdout, Stderr: res.Stderr}, err
}

func (g engineGuest) Forward(ctx context.Context, ip string, port int) (process.Stream, error) {
	c, err := g.Guest.Forward(ctx, ip, port)
	if c == nil {
		return nil, err
	}
	return c, err
}

// invariant: the session runs until its command, the connection or ctx
// ends; its output then ends too, so the engine's reader stops, and Wait
// answers the command's exit code.
func (g engineGuest) Attach(ctx context.Context, line string) (process.Session, error) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	s := &session{out: outR, in: inW, ended: make(chan struct{})}
	go func() {
		res, err := g.Guest.Exec(ctx, contract.GuestCommand{Line: line, Stdin: inR, Stdout: outW, Stderr: io.Discard})
		s.code, s.err = res.Code, err
		close(s.ended)
		_ = outW.Close()
		_ = inR.Close()
	}()
	go func() {
		select {
		case <-ctx.Done():
		case <-g.Done():
		case <-s.ended:
		}
		_ = outW.Close()
	}()
	return s, nil
}

type session struct {
	out   *io.PipeReader
	in    *io.PipeWriter
	ended chan struct{}
	code  int
	err   error
}

func (s *session) Read(p []byte) (int, error)  { return s.out.Read(p) }
func (s *session) Write(p []byte) (int, error) { return s.in.Write(p) }
func (s *session) Close() error                { return s.in.Close() }

func (s *session) Wait() (int, error) {
	<-s.ended
	return s.code, s.err
}

type buildGuests struct{ dialer contract.GuestDialer }

func (g buildGuests) Dial(ctx context.Context, vm string, at netip.AddrPort, as process.BuildLogin) (process.BuildGuest, error) {
	c, err := g.dialer.Dial(ctx, contract.GuestTarget{VM: vm, Address: at, User: as.User, Key: as.Key, Password: as.Password})
	if err != nil {
		return nil, err
	}
	return buildGuest{engineGuest{c}}, nil
}

type buildGuest struct{ engineGuest }

func (g buildGuest) Put(ctx context.Context, path string, data []byte, mode uint32) error {
	return g.engineGuest.Put(ctx, path, data, fs.FileMode(mode))
}
