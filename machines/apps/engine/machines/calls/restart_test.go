package calls_test

import (
	"context"
	"errors"
	"testing"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	machines "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

type noTailnet struct{}

func (noTailnet) Peers(context.Context) (map[string]model.Node, error) {
	return nil, nil
}

func service(t *testing.T, answers func(context.Context, string) bool) (*machines.Machinist, *stillMac) {
	t.Helper()
	clock := &stillMac{at: time.Unix(0, 0)}
	h := clock
	paths := setup.Paths{Prefs: "/config", State: "/state"}
	prefs := memoryPrefs{}
	p := machines.Prefs{Name: setupcontract.Name(prefs), Macs: machinescontract.Macs(prefs), VMUser: machinescontract.VMUser(prefs)}
	if err := p.Name.Set("mac"); err != nil {
		t.Fatal(err)
	}
	if err := p.Macs.Set(map[string]model.MacRecord{"mac": {}, "x": {User: "u"}}); err != nil {
		t.Fatal(err)
	}
	return machines.New(h, nil, p, machinesfake.NewStore(), noTailnet{}, nil, nil, nil, machines.Options{ThisMachine: catalog.Mac, KnownHosts: paths.State + "/known_hosts", Logs: paths.State + "/logs", Unlocker: restarted{}, Answers: answers}), clock
}

type restarted struct{}

func (restarted) Unlock(context.Context, string, string) error { return nil }
func (restarted) PlannedRestart(context.Context, string) error { return nil }

// value: the Mac's station answering each poll in turn, the last answer kept.
type answering struct{ polls []bool }

func (a *answering) answers(context.Context, string) bool {
	up := a.polls[0]
	if len(a.polls) > 1 {
		a.polls = a.polls[1:]
	}
	return up
}

func restarting(t *testing.T, polls ...bool) (*machines.Machinist, *stillMac) {
	t.Helper()
	return service(t, (&answering{polls: polls}).answers)
}

func TestRestartWaitsForTheMacToGoAwayAndComeBack(t *testing.T) {
	svc, _ := restarting(t, true, true, false, false, true)
	var seen []string

	report, err := svc.Restart(context.Background(), "x", func(e model.RestartEvent) { seen = append(seen, e.Phase) })

	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if report.Waited != 6*time.Second {
		t.Fatalf("waited %v, want 6s (two polls still up, one down, one down, then up)", report.Waited)
	}
	if want := []string{"restarting", "down", "back"}; len(seen) != 3 || seen[1] != want[1] || seen[2] != want[2] {
		t.Fatalf("progress = %v, want %v", seen, want)
	}
}

func TestRestartDoesNotTakeAMacThatNeverWentDownForRestarted(t *testing.T) {
	svc, clock := restarting(t, true)

	_, err := svc.Restart(context.Background(), "x", nil)

	var timeout *model.RestartTimeoutError
	if !errors.As(err, &timeout) || timeout.Phase != "stop" {
		t.Fatalf("Restart error = %v, want a timeout in the stop phase", err)
	}
	if spent := clock.at.Sub(time.Unix(0, 0)); spent != time.Minute {
		t.Fatalf("spent %v before giving up, want the 60s stop budget", spent)
	}
}

func TestRestartGivesUpOnAMacThatNeverComesBack(t *testing.T) {
	svc, clock := restarting(t, false)

	_, err := svc.Restart(context.Background(), "x", nil)

	var timeout *model.RestartTimeoutError
	if !errors.As(err, &timeout) || timeout.Phase != "return" {
		t.Fatalf("Restart error = %v, want a timeout in the return phase", err)
	}
	if spent := clock.at.Sub(time.Unix(0, 0)); spent != 3*time.Minute {
		t.Fatalf("spent %v, want the 180s return budget", spent)
	}
}
