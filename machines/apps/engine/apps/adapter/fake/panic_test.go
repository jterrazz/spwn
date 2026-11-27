package fake_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/apps/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/apps/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func panics(f func()) (broke bool) {
	defer func() { broke = recover() != nil }()
	f()
	return false
}

func answersWithin(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		f()
		close(done)
	}()
	bound, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case <-done:
	case <-bound.Done():
		t.Fatalf("%s after a world panic did not answer: the fake kept the world locked", what)
	}
}

func TestAnInstallOrABundleTheWorldScriptsToPanicLeavesTheWorldFree(t *testing.T) {
	install := catalog.ShellInstall{Line: "make tool"}
	calls := map[string]func(w *world.World) error{
		"Installer.Install": func(w *world.World) error {
			return fake.InstallerOf(w, "mbp", install.Method(), false).Install(t.Context(), []catalog.Install{install}, nil)
		},
		"Bundler.Bundle": func(w *world.World) error {
			return fake.BundlerOf(w, "mbp").Bundle(t.Context(), []string{"jq"}, false, nil)
		},
	}
	for call, run := range calls {
		t.Run(call, func(t *testing.T) {
			w := &world.World{ThisMac: "mbp"}
			w.Mac("mbp").Apps = map[catalog.Method]*world.Manager{install.Method(): {}}
			w.Script = []world.Fault{{Call: call, Times: 1, Panic: true}}
			if !panics(func() { _ = run(w) }) {
				t.Fatalf("%s did not panic: the test proves nothing", call)
			}
			answersWithin(t, "the next "+call, func() {
				if err := run(w); errors.Is(err, contract.ErrCannot) {
					t.Errorf("the next %s: %v", call, err)
				}
			})
		})
	}
}
