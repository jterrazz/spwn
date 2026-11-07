package tmux

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/agents/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const (
	session = "main"           // value: the session the dotfiles' jc opens, the one the owner attaches to.
	buffer  = "station-agents" // value: the paste buffer a text is loaded into, deleted once pasted.
	budget  = 5 * time.Second
)

type Runner interface {
	Output(ctx context.Context, c local.Cmd) (local.Output, error)
}

type Session struct{ run Runner }

var _ contract.Session = (*Session)(nil)

func New(run Runner) *Session { return &Session{run: run} }

func (s *Session) tmux(ctx context.Context, stdin string, args ...string) (local.Output, error) {
	out, err := s.run.Output(ctx, local.Cmd{Name: "tmux", Args: args, Stdin: stdin, Timeout: budget})
	if err != nil {
		return out, err
	}
	if out.ExitCode != 0 {
		return out, errors.New("tmux " + args[0] + ": " + strings.TrimSpace(out.Stderr))
	}
	return out, nil
}

func (s *Session) Open(ctx context.Context, folder string, argv []string) error {
	running, err := s.run.Output(ctx, local.Cmd{Name: "tmux", Args: []string{"has-session", "-t", session}, Timeout: budget})
	if err != nil {
		return err
	}
	args := []string{"new-session", "-d", "-s", session}
	if running.ExitCode == 0 {
		args = []string{"new-window", "-t", session}
	}
	if folder != "" {
		args = append(args, "-c", folder)
	}
	_, err = s.tmux(ctx, "", append(args, argv...)...)
	return err
}

func (s *Session) Send(ctx context.Context, text string) error {
	if _, err := s.tmux(ctx, text, "load-buffer", "-b", buffer, "-"); err != nil {
		return err
	}
	if _, err := s.tmux(ctx, "", "paste-buffer", "-d", "-b", buffer, "-t", session); err != nil {
		return err
	}
	_, err := s.tmux(ctx, "", "send-keys", "-t", session, "Enter")
	return err
}
