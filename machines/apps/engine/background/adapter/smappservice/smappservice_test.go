package smappservice_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/background/adapter/smappservice"
	"github.com/jterrazz/studio-station/apps/engine/background/contract"
)

// value: what launchd and SMAppService were asked, in order, and which
// labels launchd holds.
type trail struct {
	calls   []string
	loaded  map[string]bool
	refuse  error
	removed map[string]bool
	// value: how many reads still see a removed job, as launchd does while
	// the job's process exits after its bootout.
	lingers int
}

type manager struct{ t *trail }

func (manager) Present() bool { return true }
func (manager) Names() contract.Names {
	return contract.Names{Manager: "jobd", Command: "jobctl"}
}
func (manager) List(context.Context) ([]background.Job, error) {
	return nil, nil
}
func (l manager) Read(_ context.Context, r contract.Ref) contract.Status {
	if l.t.removed[r.Label] {
		if l.t.lingers == 0 {
			l.t.loaded[r.Label] = false
		} else {
			l.t.lingers--
		}
	}
	return contract.Status{Loaded: l.t.loaded[r.Label]}
}
func (manager) Overrides(context.Context, contract.Domain) map[string]bool { return nil }
func (l manager) Install(context.Context, contract.JobSpec) (bool, error) {
	l.t.calls = append(l.t.calls, "manager install")
	return true, nil
}
func (l manager) Load(_ context.Context, r contract.Ref) error {
	l.t.calls = append(l.t.calls, "bootstrap "+r.Label)
	return nil
}
func (l manager) Start(_ context.Context, r contract.Ref) error {
	l.t.calls = append(l.t.calls, "kickstart "+r.Label)
	return nil
}
func (l manager) Restart(_ context.Context, r contract.Ref) error {
	l.t.calls = append(l.t.calls, "manager restart "+r.Label)
	return nil
}
func (l manager) Stop(_ context.Context, r contract.Ref) error {
	l.t.calls = append(l.t.calls, "bootout "+r.Label)
	l.t.loaded[r.Label] = false
	return nil
}
func (l manager) Remove(_ context.Context, r contract.Ref) error {
	l.t.calls = append(l.t.calls, "manager remove "+r.Label)
	l.t.removed[r.Label] = true
	return nil
}

type agents struct{ t *trail }

func (a agents) Register(plist string) error {
	a.t.calls = append(a.t.calls, "register "+plist)
	if a.t.refuse != nil {
		return a.t.refuse
	}
	a.t.loaded[strings.TrimSuffix(plist, ".plist")] = true
	return nil
}

func (a agents) Unregister(plist string) error {
	a.t.calls = append(a.t.calls, "unregister "+plist)
	a.t.loaded[strings.TrimSuffix(plist, ".plist")] = false
	return nil
}

type clock struct{}

func (clock) Sleep(context.Context, time.Duration) error { return nil }

const label = "com.jterrazz.station.engine"

var ref = contract.Ref{Domain: contract.Domain{UID: 501}, Label: label, File: "/home/o/Library/LaunchAgents/" + label + ".plist"}

func jobs(loaded bool) (*smappservice.Jobs, *trail) {
	t := &trail{loaded: map[string]bool{label: loaded}, removed: map[string]bool{}}
	return smappservice.New(manager{t}, agents{t}, clock{}), t
}

func TestABundledJobIsNeverWrittenItIsTheBundles(t *testing.T) {
	j, tr := jobs(false)
	changed, err := j.Install(t.Context(), contract.JobSpec{Label: label, File: ref.File})
	if err != nil || changed || len(tr.calls) != 0 {
		t.Errorf("install = %v, %v, calls %q", changed, err, tr.calls)
	}
}

func TestABundledJobIsLoadedBySMAppServiceThenStartedByItsJobManager(t *testing.T) {
	j, tr := jobs(false)
	if err := j.Load(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	if err := j.Start(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	if want := []string{"register " + label + ".plist", "kickstart " + label}; !slices.Equal(tr.calls, want) {
		t.Errorf("calls = %q, want %q", tr.calls, want)
	}
}

func TestARestartUnregistersThenRegistersSoTheBundlesBinaryRunsNow(t *testing.T) {
	j, tr := jobs(true)
	if err := j.Restart(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	if want := []string{"unregister " + label + ".plist", "register " + label + ".plist", "kickstart " + label}; !slices.Equal(tr.calls, want) {
		t.Errorf("calls = %q, want %q", tr.calls, want)
	}
}

func TestARefusedRegisterNamesTheCallAndTheReadThatJudgesTheJob(t *testing.T) {
	j, tr := jobs(false)
	tr.refuse = errors.New("SMAppServiceErrorDomain 125: Domain does not support specified action")
	err := j.Load(t.Context(), ref)
	var failed *contract.Failed
	if !errors.As(err, &failed) {
		t.Fatalf("err = %v", err)
	}
	if failed.Verb() != "SMAppService register" || !strings.Contains(failed.Stderr, "125") || failed.See != "jobctl print gui/$(id -u)/"+label {
		t.Errorf("failed = %+v", failed)
	}
}

func TestARemoveStopsAndUnregistersAndAJobAlreadyGoneIsNoFailure(t *testing.T) {
	j, tr := jobs(true)
	if err := j.Remove(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	if want := []string{"bootout " + label, "unregister " + label + ".plist"}; !slices.Equal(tr.calls, want) {
		t.Errorf("calls = %q, want %q", tr.calls, want)
	}
}

func TestARetiredBareJobIsWaitedOutOfItsJobManagerAndNeverUnregistered(t *testing.T) {
	j, tr := jobs(true)
	tr.lingers = 3
	if err := j.Retire(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	if j.Read(t.Context(), ref).Loaded {
		t.Error("the job manager still holds the retired job")
	}
	if want := []string{"manager remove " + label}; !slices.Equal(tr.calls, want) {
		t.Errorf("calls = %q, want %q", tr.calls, want)
	}
}

func TestABareJobItsJobManagerKeepsPastItsExitTimeoutFailsTheRetire(t *testing.T) {
	j, tr := jobs(true)
	tr.lingers = 1 << 20
	err := j.Retire(t.Context(), ref)
	var failed *contract.Failed
	if !errors.As(err, &failed) {
		t.Fatalf("err = %v", err)
	}
	if failed.Verb() != "jobctl bootout" || !strings.Contains(failed.Stderr, "20s") || failed.See != "jobctl print gui/$(id -u)/"+label {
		t.Errorf("failed = %+v", failed)
	}
	if want := []string{"manager remove " + label}; !slices.Equal(tr.calls, want) {
		t.Errorf("calls = %q, want %q", tr.calls, want)
	}
}
