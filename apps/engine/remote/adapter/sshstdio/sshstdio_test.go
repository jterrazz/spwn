package sshstdio_test

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/sshstdio"
	"github.com/jterrazz/studio-station/apps/engine/remote/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

// value: ssh as a pipe to the far Mac's `station engine attach`, its server
// running in this process; a Mac that is down ends ssh with 255.
type ssh struct {
	far  serve.Calls
	down bool
	// value: the far Mac answers ssh and has no station: its shell ends 127.
	bare bool
	ran  [][]string
}

type process struct {
	in   *io.PipeWriter
	out  *io.PipeReader
	done chan struct{}
	code int
}

func (p *process) Stdin() io.WriteCloser { return p.in }
func (p *process) Stdout() io.Reader     { return p.out }
func (p *process) Kill()                 { _ = p.in.Close() }
func (p *process) Wait() (code int, stderr string, err error) {
	<-p.done
	switch p.code {
	case 0:
	case 127:
		return p.code, "sh: station: not found", nil
	default:
		return p.code, "ssh: connect to host mac-mini port 22: Host is down", nil
	}
	return 0, "", nil
}

func (s *ssh) Spawn(ctx context.Context, c local.Cmd) (local.Process, error) {
	s.ran = append(s.ran, append([]string{c.Name}, c.Args...))
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	p := &process{in: inW, out: outR, done: make(chan struct{})}
	go func() {
		defer close(p.done)
		if s.down || s.bare {
			p.code = 255
			if s.bare {
				p.code = 127
			}
			_ = inR.Close()
			_ = outW.Close()
			return
		}
		_ = serve.Stream(context.WithoutCancel(ctx), inR, outW, s.far, func(err error) *fail.Error {
			var e *fail.Error
			if errors.As(err, &e) {
				return e
			}
			return &fail.Error{Message: err.Error(), Exit: fail.ExitFailure}
		})
		_ = outW.Close()
	}()
	return p, nil
}

func TestSSHToTheFarEngineKeepsTheEndpointContract(t *testing.T) {
	contracttest.Endpoint(t, func(_ *testing.T, far *call.Registry, profile machine.Profile, version string) contracttest.EndpointSubject {
		s := &ssh{far: serve.Local(far, profile, version)}
		return contracttest.EndpointSubject{Dialer: sshstdio.New(s, version), Mac: profile.Ref.Name, Target: profile.Ref, Down: func() { s.down = true }}
	})
}

func TestAnotherMacIsReachedOnlyThroughItsEngineInBatchMode(t *testing.T) {
	s := &ssh{far: serve.Local(contracttest.Far(), contracttest.Mini, contracttest.Version)}
	ep, err := sshstdio.New(s, contracttest.Version).Dial(t.Context(), "mac-mini")
	if err != nil {
		t.Fatal(err)
	}
	_ = ep.Close()
	want := []string{"ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "mac-mini", "--", "sh", "-c"}
	if len(s.ran) != 1 || !slices.Equal(s.ran[0][:len(want)], want) || !strings.HasSuffix(strings.Join(s.ran[0], " "), " station engine attach") {
		t.Errorf("dialing mac-mini ran %q, want one BatchMode ssh to its `station engine attach`", s.ran)
	}
}

func TestAMacThatAnswersWithoutStationIsToldApartFromOneThatDoesNotAnswer(t *testing.T) {
	s := &ssh{far: serve.Local(contracttest.Far(), contracttest.Mini, contracttest.Version), bare: true}
	ep, err := sshstdio.New(s, contracttest.Version).Dial(t.Context(), "mac-mini")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ep.Close() }()
	_, err = ep.Hello(t.Context())
	var e *fail.Error
	if !errors.As(err, &e) || e.Code != call.NoStation || e.Subject != "mac-mini" {
		t.Fatalf("Hello = %v, want mac-mini's missing station told as %s", err, call.NoStation)
	}
}
