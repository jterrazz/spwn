package wire

import (
	"context"
	"io"
	"io/fs"
	"net/netip"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/ssh"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

// invariant: the one way into a VM: ssh, or the world's guests in a spec
// build.
func (m machine) guestDialer() contract.GuestDialer {
	if m.world != nil {
		return m.world.guests()
	}
	return ssh.New(m.sys.FS, m.sys.Sockets)
}

type engineGuests struct {
	dialer contract.GuestDialer
	vms    *machinescalls.Machinist
}

func (g engineGuests) Dial(ctx context.Context, vm string, at netip.AddrPort) (process.Guest, error) {
	c, err := g.dialer.Dial(ctx, contract.GuestTarget{VM: vm, Address: at, User: g.vms.VMUser(), Key: g.vms.KeyPath(), KnownHosts: g.vms.KnownHostsPath(vm)})
	if err != nil {
		return nil, err
	}
	return engineGuest{c}, nil
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
