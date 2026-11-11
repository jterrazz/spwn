package serve_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/remote/serve"
)

type headOut struct {
	Kind     machine.Kind
	Lifetime string
}

// value: what a Mac answers a caller that opens with hello, then asks for
// its head.
func answered(t *testing.T, hello string) string {
	t.Helper()
	registry, err := call.NewRegistry([]call.Handler{
		call.Bind(call.Call[struct{}, struct{}, headOut]{
			Name: "overview.read", Where: call.AtTarget, Applies: machine.Anywhere,
			Run: func(context.Context, struct{}, struct{}, call.Progress) (headOut, error) {
				return headOut{Kind: machine.Computer, Lifetime: "persistent"}, nil
			},
		}, struct{}{}),
	})
	if err != nil {
		t.Fatal(err)
	}
	mini := machine.Profile{Ref: machine.Ref{Name: "mac-mini"}, Kind: machine.Computer, System: machine.System{OS: machine.MacOS}}
	in := `{"v":2,"id":"1","hello":` + hello + "}\n" + `{"v":2,"id":"2","call":"overview.read"}` + "\n"
	var out bytes.Buffer
	if err := serve.Stream(t.Context(), strings.NewReader(in), &out, serve.Local(registry, mini, "v3.1.0"), nil); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestAnOlderCallerIsAnsweredInTheWordsItReads(t *testing.T) {
	for _, hello := range []string{`{}`, `{"version":"v3.0.0"}`} {
		got := answered(t, hello)
		if !strings.Contains(got, `"kind":"mac"`) || !strings.Contains(got, `"Kind":"mac"`) || !strings.Contains(got, `"Lifetime":"permanent"`) || strings.Contains(got, "computer") {
			t.Fatalf("a caller that said hello %s was answered\n%s", hello, got)
		}
	}
	got := answered(t, `{"version":"v3.1.0"}`)
	if !strings.Contains(got, `"kind":"computer"`) || !strings.Contains(got, `"Kind":"computer"`) || !strings.Contains(got, `"Lifetime":"persistent"`) {
		t.Fatalf("a caller of today's words was answered\n%s", got)
	}
}
