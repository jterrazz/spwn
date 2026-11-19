package contracttest

import (
	"bufio"
	"bytes"
	"context"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const (
	Secret    = "s3cret-value"
	GuestPort = 8080
	Greeting  = "hello from 8080"
)

type Answer struct {
	Code           int
	Stdout, Stderr string
}

type GuestSubject struct {
	// value: the dialer, and the target of the one guest the subject runs,
	// its service on GuestPort greeting with Greeting.
	Dialer contract.GuestDialer
	Target func(t *testing.T) contract.GuestTarget
	// value: how the guest answers a command line, from now on.
	Program func(line string, a Answer)
	// value: the bytes a programmed command read on its stdin.
	Received func(line string) []byte
	// value: a file inside the guest, as Put left it.
	Read func(t *testing.T, path string) ([]byte, fs.FileMode)
	// value: a path inside the guest a Put may write.
	Path string
}

func (s GuestSubject) dial(t *testing.T) contract.Guest {
	t.Helper()
	g, err := s.Dialer.Dial(t.Context(), s.Target(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = g.Close() })
	return g
}

func Guest(t *testing.T, subject func(t *testing.T) GuestSubject) {
	t.Run("a command's exit code, output and errors come back apart", func(t *testing.T) {
		s := subject(t)
		s.Program("check", Answer{Code: 3, Stdout: "out\n", Stderr: "err\n"})
		got, err := s.dial(t).Exec(t.Context(), contract.GuestCommand{Line: "check"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Code != 3 || got.Stdout != "out\n" || got.Stderr != "err\n" {
			t.Errorf("the command came back %+v, want exit 3, out and err apart", got)
		}
	})
	t.Run("a command's output streams to the writers it is given", func(t *testing.T) {
		s := subject(t)
		s.Program("check", Answer{Stdout: "out\n", Stderr: "err\n"})
		var out, errs bytes.Buffer
		if _, err := s.dial(t).Exec(t.Context(), contract.GuestCommand{Line: "check", Stdout: &out, Stderr: &errs}); err != nil {
			t.Fatal(err)
		}
		if out.String() != "out\n" || errs.String() != "err\n" {
			t.Errorf("the writers got %q and %q, want out and err", out.String(), errs.String())
		}
	})
	t.Run("a secret reaches the command by its stdin, whole", func(t *testing.T) {
		s := subject(t)
		s.Program("import", Answer{})
		if _, err := s.dial(t).Exec(t.Context(), contract.GuestCommand{Line: "import", Stdin: strings.NewReader(Secret)}); err != nil {
			t.Fatal(err)
		}
		if got := string(s.Received("import")); got != Secret {
			t.Errorf("the command read %q on its stdin, want the secret", got)
		}
	})
	t.Run("a file put is in the guest whole, with its mode", func(t *testing.T) {
		s := subject(t)
		data := []byte("#!/bin/sh\necho station\n")
		if err := s.dial(t).Put(t.Context(), s.Path, data, 0o755); err != nil {
			t.Fatal(err)
		}
		got, mode := s.Read(t, s.Path)
		if !bytes.Equal(got, data) || mode.Perm() != 0o755 {
			t.Errorf("the guest holds %q with mode %v, want the bytes put with 0755", got, mode.Perm())
		}
	})
	t.Run("a forward reaches a port inside the guest", func(t *testing.T) {
		s := subject(t)
		conn, err := s.dial(t).Forward(t.Context(), "127.0.0.1", GuestPort)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil || strings.TrimSpace(line) != Greeting {
			t.Errorf("the forward read %q (%v), want the guest's greeting", line, err)
		}
	})
	t.Run("a closed connection says it is done", func(t *testing.T) {
		g := subject(t).dial(t)
		if err := g.Close(); err != nil {
			t.Fatal(err)
		}
		wait, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		select {
		case <-g.Done():
		case <-wait.Done():
			t.Error("Done never closed after Close")
		}
	})
}
