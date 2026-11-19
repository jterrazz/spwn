package localtest

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func Unreachable() error {
	// value: a TCP dial's failure as macOS answers it while the Mac has not
	// approved the process for Local Network: the route refused, EHOSTUNREACH.
	return &net.OpError{Op: "dial", Net: "tcp", Err: local.ErrUnreachableRoute}
}

func Refused() error {
	// value: a TCP dial's failure where nothing listens.
	return &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connect: connection refused")}
}

type refusing struct {
	local.Sockets
	err error
}

func (r refusing) DialTCP(context.Context, string, time.Duration) (net.Conn, error) {
	return nil, r.err
}

func Refusing(s local.Sockets, err error) local.Sockets {
	// value: sockets whose every TCP dial fails with err, the rest of them
	// those of s.
	return refusing{Sockets: s, err: err}
}
