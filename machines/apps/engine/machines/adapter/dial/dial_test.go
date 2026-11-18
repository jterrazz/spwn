package dial_test

import (
	"io"
	"strconv"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/dial"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// invariant: a server that answers every request with 204, on a port of
// this Mac's loopback, closed for the refusing half.
func TestTheDialCheckerKeepsTheCheckRunnerContract(t *testing.T) {
	contracttest.CheckRunner(t, func(t *testing.T) contracttest.CheckRunnerSubject {
		host := local.New(nil, nil, io.Discard, io.Discard)
		l, err := host.Sockets.ListenTCP("127.0.0.1", 0)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			for {
				c, err := l.Accept()
				if err != nil {
					return
				}
				go func() {
					buf := make([]byte, 4096)
					_, _ = c.Read(buf)
					_, _ = c.Write([]byte("HTTP/1.1 204 No Content\r\nConnection: close\r\n\r\n"))
					_ = c.Close()
				}()
			}
		}()
		t.Cleanup(func() { _ = l.Close() })
		at := "127.0.0.1:" + strconv.Itoa(l.Port())
		return contracttest.CheckRunnerSubject{
			Runner: dial.Checker{},
			Answer: "204",
			Answering: func() machines.CheckTarget {
				return machines.CheckTarget{Dial: at, URL: "http://infra:" + strconv.Itoa(l.Port()) + "/readyz", Name: "infra"}
			},
			Refusing: func() machines.CheckTarget {
				_ = l.Close()
				return machines.CheckTarget{Dial: at}
			},
		}
	})
}
