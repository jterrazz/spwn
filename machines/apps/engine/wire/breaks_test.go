package wire

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

func TestABreakInAStreamProcessIsALineOfItsMachinesEngineLog(t *testing.T) {
	fsys := localtest.NewFS()
	clock := &localtest.Clock{At: time.Date(2026, 11, 27, 9, 30, 0, 0, time.UTC)}
	m := machine{
		sys:   localtest.Mac(localtest.NewRunner(t), fsys, localtest.Env{HomeDir: "/home/u", OS: "linux"}, localtest.NewNet(), clock),
		paths: setup.Paths{Prefs: "/home/u/.config/station", State: "/home/u/.local/state/station"},
	}
	breaks := m.streamBreaks()
	if breaks == nil {
		t.Fatal("a stream process outside a world has no sink for its breaks: a break there is written nowhere")
	}
	breaks("broke in serve.call on ports.list: goes.Script\ngoroutine 1 [running]:")
	breaks("broke in serve.watch on ports.list: goes.Script")
	got, err := fsys.ReadFile("/home/u/.local/state/station/logs/engine.log")
	want := "2026-11-27T09:30:00Z engine: broke in serve.call on ports.list: goes.Script\ngoroutine 1 [running]:\n" +
		"2026-11-27T09:30:00Z engine: broke in serve.watch on ports.list: goes.Script\n"
	if err != nil || string(got) != want {
		t.Fatalf("engine.log = %q, %v; want each break appended, stamped as the engine stamps its own:\n%q", got, err, want)
	}
}
