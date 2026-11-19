package process

import (
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type kept struct {
	got []engine.Record
	err error
}

func (k *kept) Write(r engine.Record) error {
	k.got = append(k.got, r)
	return k.err
}

func TestTheEngineKeepsTheBinaryItStartedFromAtEachStart(t *testing.T) {
	h := newHarness(t)
	k := &kept{}
	h.d.o.Records, h.d.o.Executable, h.d.o.Version = k, "/opt/bin/station", "v0.21.0"
	if err := h.d.Begin(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	want := engine.Record{V: 1, Executable: "/opt/bin/station", Version: "v0.21.0", At: h.clock.At.UTC().Truncate(time.Second)}
	if len(k.got) != 1 || k.got[0] != want {
		t.Fatalf("kept %+v, want one record %+v", k.got, want)
	}
}

func TestARecordThatCannotBeKeptDoesNotStopTheEngine(t *testing.T) {
	h := newHarness(t)
	h.d.o.Records = &kept{err: errors.New("disk full")}
	if err := h.d.Begin(t.Context(), nil); err != nil {
		t.Fatalf("Begin = %v, want the engine started", err)
	}
}
