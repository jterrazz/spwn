//go:build !world

package wire

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

func TestTheEnginesStartSaysABreakOnItsSocketInItsLog(t *testing.T) {
	var mu sync.Mutex
	var logged []string
	log := func(line string) {
		mu.Lock()
		defer mu.Unlock()
		logged = append(logged, line)
	}
	broken := call.Bind(call.Call[struct{}, struct{}, struct{}]{
		Name: "machines.break", Where: call.InEngine, Applies: coremachine.Anywhere,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (struct{}, error) {
			panic(errors.New("index out of range [3] with length 0"))
		},
	}, struct{}{})
	s := serve.Server{
		ThisMachine: coremachine.Profile{Ref: coremachine.Ref{Name: "macbook-pro"}, Kind: coremachine.Computer},
		Log:         "~/.local/state/station/logs/engine.log",
		Calls:       func(name string) (call.Handler, bool) { return broken, name == "machines.break" },
	}
	var answered api.Frame
	engine := runner{func(ctx context.Context) {
		inR, inW := io.Pipe()
		outR, outW := io.Pipe()
		defer func() { _, _ = inW.Close(), outR.Close() }()
		go s.Conn(ctx, piped{Reader: inR, Writer: outW, closers: []io.Closer{inR, outW}})
		if _, err := inW.Write([]byte(`{"v":2,"id":"1","call":"machines.break"}` + "\n")); err != nil {
			t.Error(err)
			return
		}
		lines := bufio.NewScanner(outR)
		if !lines.Scan() || json.Unmarshal(lines.Bytes(), &answered) != nil {
			t.Errorf("the socket answered nothing: %v", lines.Err())
		}
	}}
	book := bookWithLaptop(t).store(nil)
	err := soleEngine{Engine: engine, book: book, log: log, clock: &localtest.Clock{}, exit: func(code int) { t.Errorf("the engine exited %d", code) }}.Run(t.Context(), func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if answered.Error == nil || answered.Error.Code != call.Broke {
		t.Fatalf("the call that panicked answered %+v, want it coded broke", answered)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(logged) != 1 || !strings.HasPrefix(logged[0], "broke in serve.call on machines.break: *errors.errorString: index out of range [3] with length 0\n") {
		t.Fatalf("the engine's log = %q; want the break's line then its stack: the engine's start sets core/goes", logged)
	}
}
