package macos

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/src/platform/local"
)

type answers map[string]string

func (a answers) Output(_ context.Context, c local.Cmd) (local.Output, error) {
	text, ok := a[c.Args[1]]
	if !ok {
		return local.Output{ExitCode: 1}, nil
	}
	return local.Output{Stdout: text}, nil
}

func TestTheRangeIsBothEndsTheKernelAnswers(t *testing.T) {
	from, to, ok := New(answers{"net.inet.ip.portrange.first": "49152\n", "net.inet.ip.portrange.last": "65535\n"}).Range(context.Background())
	if !ok || from != 49152 || to != 65535 {
		t.Fatalf("range %d-%d %v", from, to, ok)
	}
	if _, _, ok := New(answers{"net.inet.ip.portrange.first": "49152\n"}).Range(context.Background()); ok {
		t.Fatal("one end alone read as a range")
	}
}
