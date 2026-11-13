package architecture_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/output"
	"github.com/jterrazz/studio-station/apps/engine/page"
	remotefake "github.com/jterrazz/studio-station/apps/engine/remote/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	remoteserve "github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

const planted = "fake-vault-value-of-the-tripwire"

type trips struct {
	mu   sync.Mutex
	said []string
}

func (t *trips) trip(said string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.said = append(t.said, said)
}

func (t *trips) count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.said)
}

type conn struct {
	io.Reader
	io.Writer
	closers []io.Closer
}

func (c conn) Close() error {
	for _, x := range c.closers {
		_ = x.Close()
	}
	return nil
}

type leakOut struct{ Token string }

func leaking() call.Handler {
	return call.Bind(call.Call[struct{}, struct{}, leakOut]{
		Name: "machines.leak", Where: call.InEngine, Applies: machine.Anywhere,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (leakOut, error) {
			return leakOut{Token: planted}, nil
		},
	}, struct{}{})
}

func leakingServer() remoteserve.Server {
	leak := leaking()
	return remoteserve.Server{
		Calls:       func(name string) (call.Handler, bool) { return leak, name == "machines.leak" },
		ThisMachine: machine.Profile{Ref: machine.Ref{Name: "jterrazz-macbook-pro"}, Kind: machine.Computer},
	}
}

// invariant: rule tripwire-catches: once the world build arms the
// tripwire, a frame the engine serves, an answer from another Mac, a write
// of the CLI and a sealed page that carry a secret of the world each trip
// it, so every story over a world is a leak check.
func TestTripwireCatches(t *testing.T) {
	seen := &trips{}
	secret.Arm([]string{planted}, seen.trip)
	t.Cleanup(secret.Disarm)
	for _, c := range []struct {
		surface string
		answer  func(t *testing.T)
	}{
		{"a frame the engine serves", func(t *testing.T) {
			s := leakingServer()
			client := api.NewClient(func(ctx context.Context) (io.ReadWriteCloser, error) {
				inR, inW := io.Pipe()
				outR, outW := io.Pipe()
				go s.Conn(ctx, conn{Reader: inR, Writer: outW, closers: []io.Closer{inR, outW}})
				return conn{Reader: outR, Writer: inW, closers: []io.Closer{outR, inW}}, nil
			})
			var out leakOut
			_ = client.Call(t.Context(), "machines.leak", machine.Ref{Name: "jterrazz-macbook-pro"}, struct{}{}, &out, nil)
		}},
		{"an answer from another Mac", func(t *testing.T) {
			far, err := call.NewRegistry([]call.Handler{leaking()})
			if err != nil {
				t.Fatal(err)
			}
			mini := machine.Profile{Ref: machine.Ref{Name: "jterrazz-mac-mini"}, Kind: machine.Computer}
			d := remotefake.New()
			d.Serve(mini.Ref.Name, remotefake.Mac{Registry: far, Profile: mini})
			e, err := d.Dial(t.Context(), mini.Ref.Name)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = e.Invoke(t.Context(), machine.Ref{}, "machines.leak", json.RawMessage(`{}`), nil)
		}},
		{"a write of the CLI", func(*testing.T) {
			var stdout bytes.Buffer
			_ = output.NewWriter(output.Stream{W: &stdout}, output.Stream{W: io.Discard}, nil).Raw([]byte("token " + planted + "\n"))
		}},
		{"an error block of the CLI", func(*testing.T) {
			output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: io.Discard}, nil).Failure(errors.New("refused for " + planted))
		}},
		{"a station.error/v1 document", func(*testing.T) {
			output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: io.Discard}, nil).FailureJSON(errors.New("refused for " + planted))
		}},
		{"a note of the CLI", func(*testing.T) {
			output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: io.Discard}, nil).Note("token", planted)
		}},
		{"a finding of the CLI", func(*testing.T) {
			_ = output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: io.Discard}, nil).Finding(page.Attention, "token", planted)
		}},
		{"a page the CLI prints", func(*testing.T) {
			_ = output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: io.Discard}, nil).Page(page.Page{Command: "station " + planted}, nil, output.Want{Page: true})
		}},
		{"the alias line of the CLI", func(*testing.T) {
			output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: io.Discard}, nil).Now("station " + planted)
		}},
		{"a menu of the CLI", func(*testing.T) {
			output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: io.Discard}, nil).Menu(func(w io.Writer) { _, _ = io.WriteString(w, planted) })
		}},
		{"a wait of the CLI", func(*testing.T) {
			output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: io.Discard, IsTerminal: true}, nil).Wait("reading " + planted)()
		}},
		{"a step said ahead on the CLI", func(*testing.T) {
			output.NewWriter(output.Stream{W: io.Discard}, output.Stream{W: io.Discard, IsTerminal: true}, nil).Ahead("installing", planted)
		}},
		{"a live page of the CLI", func(*testing.T) {
			pages := make(chan page.Page, 1)
			pages <- page.Page{Command: "station " + planted}
			close(pages)
			_ = output.NewWriter(output.Stream{W: io.Discard, IsTerminal: true}, output.Stream{W: io.Discard}, nil).LivePage(pages)
		}},
		{"a sealed page", func(*testing.T) {
			page.Seal(page.Page{Command: "station " + planted})
		}},
	} {
		before := seen.count()
		c.answer(t)
		if seen.count() == before {
			t.Errorf("%s carried a secret of the world and the tripwire held still", c.surface)
		}
	}
	for _, said := range seen.said {
		if strings.Contains(said, planted) {
			t.Errorf("a trip said the secret: %q", said)
		}
	}
}
