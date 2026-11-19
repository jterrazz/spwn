package remote_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
)

func TestACreateAimedAtAnotherMacByItsShortNameIsThatMacsAndItsRefusalNamesIt(t *testing.T) {
	ran := 0
	minis, err := call.NewRegistry([]call.Handler{call.Bind(call.Call[struct{}, struct{}, struct{}]{
		Name: "machines.create-workspace", Where: call.AtTarget, Applies: machine.Anywhere,
		Run: func(context.Context, struct{}, struct{}, call.Progress) (struct{}, error) {
			ran++
			return struct{}{}, fail.New(fail.Refused, "machines.no-room", "linux-9", map[string]string{"mac": fullMini})
		},
	}, struct{}{})})
	if err != nil {
		t.Fatal(err)
	}
	s, macs, _ := laptopSocketOver(t, minis)
	frames := answered(t, s, `{"v":2,"id":"1","call":"machines.create-workspace","target":{"name":"mac-mini"},"in":{}}`)
	if len(frames) == 0 || frames[0].Error == nil || frames[0].Error.Detail["mac"] != fullMini || ran != 1 {
		t.Errorf("a create aimed at mac-mini: %+v, ran on the mini %d times; want the mini's refusal, naming %s", frames, ran, fullMini)
	}
	if !slices.Equal(macs.Dialed(), []string{fullMini}) {
		t.Errorf("a create aimed at mac-mini dialed %v; want %s", macs.Dialed(), fullMini)
	}
}

// value: the frame that answers one call line, within a bound.
func answered(t *testing.T, s serve.Server, line string) []api.Frame {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	t.Cleanup(func() { _ = inW.Close() })
	go s.Conn(t.Context(), struct {
		io.Reader
		io.Writer
		io.Closer
	}{inR, outW, closers{inR, outW}})
	go func() { _, _ = inW.Write([]byte(line + "\n")) }()
	got := make(chan []api.Frame, 1)
	go func() {
		lines := bufio.NewScanner(outR)
		for lines.Scan() {
			var f api.Frame
			if json.Unmarshal(lines.Bytes(), &f) == nil && f.Progress == nil {
				got <- []api.Frame{f}
				return
			}
		}
		got <- nil
	}()
	bound, stop := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop()
	select {
	case frames := <-got:
		return frames
	case <-bound.Done():
		t.Fatalf("%s: never answered", line)
		return nil
	}
}
