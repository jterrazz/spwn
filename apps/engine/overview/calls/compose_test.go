package calls

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"
)

type downRouter struct{ gate <-chan struct{} }

func (r downRouter) Invoke(_ context.Context, _ machine.Ref, name string, _, _ any, _ call.Progress) error {
	if name != enginecalls.Read.Name && r.gate != nil {
		<-r.gate
	}
	return errors.New("down")
}

func (downRouter) Watch(context.Context, machine.Ref, string, any, call.Progress) (<-chan call.Update[any], error) {
	return nil, errors.New("down")
}

type noPrefs struct{}

func (noPrefs) Read(string, any) error  { return nil }
func (noPrefs) Write(string, any) error { return nil }

type noSettings struct{ gate <-chan struct{} }

func (s noSettings) List(context.Context) ([]struct{}, error) {
	if s.gate != nil {
		<-s.gate
	}
	return nil, errors.New("down")
}
func (noSettings) Placed(struct{}) (string, catalog.State) { return "", catalog.StateUnknown }

type noFacts struct{}

func (noFacts) Read(context.Context, catalog.Fact) (settings.Fact, error) {
	return settings.Fact{}, errors.New("down")
}

func readerOf(kind machine.Kind, gate <-chan struct{}) *Reader[struct{}, struct{}, struct{}] {
	return New(Deps[struct{}, struct{}, struct{}]{
		Calls: downRouter{gate: gate}, Kind: kind, ThisMac: ThisMac{Hostname: "macbook-pro"},
		Name: setupcontract.Name(noPrefs{}), Settings: noSettings{gate: gate}, Facts: noFacts{},
	})
}

func TestAComposeTheCallerLeavesStrandsNoGoroutine(t *testing.T) {
	before := runtime.NumGoroutine()
	gate := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	readerOf(machine.Computer, gate).Compose(ctx, true)
	cancel()
	close(gate)
	bound, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	for runtime.NumGoroutine() > before && bound.Err() == nil {
		runtime.Gosched()
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Fatalf("%d goroutines after a cancelled compose, %d before", after, before)
	}
}

func TestAMachineThatIsNotAComputerNeverReadsTheOtherMacs(t *testing.T) {
	var last overview.Summary
	done := make(chan struct{})
	updates := readerOf(machine.Server, nil).Compose(context.Background(), true)
	go func() {
		defer close(done)
		for u := range updates {
			last = u
		}
	}()
	bound, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	select {
	case <-done:
	case <-bound.Done():
		t.Fatal("a server asked for the other Macs never finished")
	}
	if last.Verdict.Sentence == "Reading macbook-pro." {
		t.Fatalf("a server's verdict still waits on the other Macs: %+v", last.Verdict)
	}
}

func TestARefusedFileOfTheOwnerIsANeedWithTheFileToFix(t *testing.T) {
	head := overview.Head{Name: "laptop", Kind: machine.Computer, ThisMac: true}
	refused := &Refusal{Text: "~/.config/station/catalog/apps/jq.yaml:3: unknown key", File: "~/.config/station/catalog/apps/jq.yaml"}

	got := summarize(head, nil, nil, nil, nil, &machines.RollUp{}, refused).Needs
	want := overview.Need{Machine: "laptop", Section: section.Overview, Broken: true, Severity: finding.Broken, Subject: overview.RefusedSubject,
		Text: refused.Text, Path: refused.File, Upkeep: overview.NeedsYou, Verb: overview.VerbFix, Mark: overview.Mark(overview.NeedsYou), Noun: "Overview"}
	if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
		t.Fatalf("needs %+v, want exactly %+v", got, want)
	}
	if none := summarize(head, nil, nil, nil, nil, &machines.RollUp{}, nil).Needs; len(none) != 0 {
		t.Errorf("no refused file, yet needs %+v", none)
	}
}

func TestALedgerAnOlderStationWroteIsANeedFixedByInit(t *testing.T) {
	laptop := machine.Ref{Name: "laptop"}
	older := &store.UnreadableError{Path: "~/.local/state/station/ledger.json", Cause: &store.OlderError{Version: 1, Current: 2}}
	found := overview.AppsRead(nil, laptop, nightsOf(machinefilecalls.Nightly{}, older, time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC))).Findings
	if len(found) != 1 || found[0].Fix.Command != "station init" || found[0].Severity != finding.Broken || !strings.Contains(found[0].Text, "written at version 1 by an older station") {
		t.Fatalf("findings %+v, want one broken line saying the ledger's version, fixed by station init", found)
	}
	if other := overview.AppsRead(nil, laptop, nightsOf(machinefilecalls.Nightly{}, errors.New("no such call"), time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC))).Findings; len(other) != 0 {
		t.Errorf("a night that cannot be asked for another reason, yet findings %+v", other)
	}
}
