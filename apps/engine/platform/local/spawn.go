package local

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
)

type Spawner interface {
	// value: a tool kept running, its stdin and stdout the caller's to speak over.
	Spawn(ctx context.Context, c Cmd) (Process, error)
}

type Process interface {
	Stdin() io.WriteCloser
	Stdout() io.Reader
	// invariant: called once its stdout has ended: the tool's exit code and
	// its stderr kept whole, for the error that names why it ended.
	Wait() (code int, stderr string, err error)
	Kill()
}

type spawned struct {
	in     io.WriteCloser
	out    io.Reader
	cmd    *exec.Cmd
	stderr *bytes.Buffer
	cancel context.CancelFunc
}

func (r *runner) Spawn(ctx context.Context, c Cmd) (Process, error) {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	_, cmd, stop, err := r.command(ctx, c)
	if err != nil {
		cancel()
		return nil, err
	}
	s := &spawned{cmd: cmd, stderr: &bytes.Buffer{}, cancel: func() { stop(); cancel() }}
	cmd.Stderr = s.stderr
	if s.in, err = cmd.StdinPipe(); err != nil {
		s.cancel()
		return nil, err
	}
	if s.out, err = cmd.StdoutPipe(); err != nil {
		s.cancel()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		s.cancel()
		return nil, err
	}
	return s, nil
}

func (s *spawned) Stdin() io.WriteCloser { return s.in }
func (s *spawned) Stdout() io.Reader     { return s.out }
func (s *spawned) Kill()                 { s.cancel() }

func (s *spawned) Wait() (code int, stderr string, err error) {
	err = s.cmd.Wait()
	s.cancel()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), s.stderr.String(), nil
	}
	return 0, s.stderr.String(), err
}
