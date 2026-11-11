package sshstdio

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/remote/api"
	"github.com/jterrazz/studio-station/src/remote/contract"
)

const (
	unreachable = 255
	// value: the far shell found no station to exec.
	missing     = 127
	connectWait = 5
	// value: what the far side is given to end once its stdin is closed.
	closeWait = 2 * time.Second
	// value: the far Mac's own login PATH is not ssh's, so station is looked
	// for where scripts/install.sh puts it.
	attach = `'PATH="$HOME/.local/bin:/usr/local/bin:$PATH" exec station "$@"'`
)

type Dialer struct {
	spawn local.Spawner
	// value: this binary's version, named by the hello of each connection.
	version string
}

func New(spawn local.Spawner, version string) Dialer { return Dialer{spawn: spawn, version: version} }

func Argv(mac string) []string {
	// invariant: the one command line station runs on another Mac: its
	// engine's API over ssh's stdio, in BatchMode, so it never waits on a
	// prompt nobody sees.
	return []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=" + strconv.Itoa(connectWait), mac, "--", "sh", "-c", attach, "station", "engine", "attach"}
}

func (d Dialer) Dial(ctx context.Context, mac string) (contract.Endpoint, error) {
	p, err := d.spawn.Spawn(ctx, local.Cmd{Name: "ssh", Args: Argv(mac)})
	if err != nil {
		return nil, gone(mac, err)
	}
	var once sync.Once
	var ended error
	end := func() error {
		once.Do(func() {
			code, stderr, err := p.Wait()
			switch {
			case err != nil:
				ended = gone(mac, err)
			case code == unreachable:
				ended = gone(mac, errors.New(strings.TrimSpace(stderr)))
			case code == missing:
				e := fail.New(fail.Failed, call.NoStation, mac, nil)
				e.Cause = errors.New(strings.TrimSpace(stderr))
				ended = e
			default:
				e := fail.New(fail.Unreachable, call.Unreachable, mac, map[string]string{"exit": strconv.Itoa(code)})
				e.Cause = errors.New(strings.TrimSpace(stderr))
				ended = e
			}
		})
		return ended
	}
	closer := func() error {
		_ = p.Stdin().Close()
		done := make(chan struct{})
		go func() { _ = end(); close(done) }()
		wait, cancel := context.WithTimeout(context.Background(), closeWait)
		defer cancel()
		select {
		case <-done:
		case <-wait.Done():
			p.Kill()
			<-done
		}
		return nil
	}
	return api.NewConn(p.Stdout(), p.Stdin(), closer, func(error) error { return end() }, d.version), nil
}

func gone(mac string, cause error) error {
	e := fail.New(fail.Unreachable, call.Unreachable, mac, nil)
	e.Cause = cause
	return e
}
