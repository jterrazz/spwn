package contracttest

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/automations"
	"github.com/jterrazz/studio-station/apps/engine/automations/contract"
)

type Declared struct {
	Name, Agent string
	automations.Trigger
}

type Opener func(t *testing.T, declared []Declared, ran []automations.Run) (contract.Automations, string)

type NoProjectOpener func(t *testing.T) (contract.Automations, string)

func at(day, hour int) time.Time { return time.Date(2026, time.May, day, hour, 0, 0, 0, time.UTC) }

func Automations(t *testing.T, open Opener) {
	// invariant: open gives Automations over a project of its own and that
	// project's root, the automations declared, and the runs written for it
	// in the order given.
	ctx := context.Background()
	morning := Declared{Name: "morning", Agent: "neo", Trigger: automations.Trigger{Kind: automations.Cron, Spec: "0 6 * * *"}}
	inbox := Declared{Name: "inbox", Agent: "trinity", Trigger: automations.Trigger{Kind: automations.FS, Spec: "./inbox"}}
	ran := []automations.Run{
		{Name: "morning", Scheduled: at(1, 6), At: at(1, 6), OK: true, Output: "brief written"},
		{Name: "inbox", At: at(1, 11), OK: true},
		{Name: "morning", Scheduled: at(2, 6), At: at(2, 6), Error: "no running world"},
		{Name: "morning", Scheduled: at(3, 6), At: at(3, 6), OK: true},
	}

	t.Run("no automation declared lists none", func(t *testing.T) {
		m, project := open(t, nil, nil)
		got, err := m.List(ctx, project)
		if err != nil || len(got) != 0 {
			t.Fatalf("List = %+v, %v; want none", got, err)
		}
	})

	t.Run("the automations are listed by name, never run", func(t *testing.T) {
		m, project := open(t, []Declared{morning, inbox}, nil)
		got, err := m.List(ctx, project)
		want := []automations.Automation{
			{Name: "inbox", Agent: "trinity", Trigger: inbox.Trigger},
			{Name: "morning", Agent: "neo", Trigger: morning.Trigger},
		}
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("List = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("the runs are counted for the automation that made them", func(t *testing.T) {
		m, project := open(t, []Declared{morning, inbox}, ran)
		got, err := m.Automation(ctx, project, "morning")
		want := automations.Automation{Name: "morning", Agent: "neo", Trigger: morning.Trigger, LastFired: at(3, 6), Fires: 3, Failed: 1}
		if err != nil || !same(got, want) {
			t.Fatalf("Automation = %+v, %v; want %+v", got, err, want)
		}
		all, err := m.List(ctx, project)
		if err != nil || len(all) != 2 || all[0].Name != "inbox" || all[0].Fires != 1 || !all[0].LastFired.Equal(at(1, 11)) {
			t.Fatalf("List = %+v, %v; want inbox ran once at 11:00", all, err)
		}
	})

	t.Run("a name no automation has, or no name at all, is not found", func(t *testing.T) {
		m, project := open(t, []Declared{morning}, ran)
		for _, name := range []string{"gone", "", "../morning"} {
			if _, err := m.Automation(ctx, project, name); !errors.Is(err, automations.ErrNotFound) {
				t.Errorf("Automation(%q) = %v; want ErrNotFound", name, err)
			}
		}
	})

	t.Run("the log is every run, oldest first", func(t *testing.T) {
		m, project := open(t, []Declared{morning, inbox}, ran)
		got, err := m.Log(ctx, project, "", 50)
		if err != nil || len(got) != len(ran) {
			t.Fatalf("Log = %+v, %v; want %d runs", got, err, len(ran))
		}
		for i := range ran {
			if !sameRun(got[i], ran[i]) {
				t.Errorf("Log[%d] = %+v; want %+v", i, got[i], ran[i])
			}
		}
	})

	t.Run("the log of one automation keeps its runs, and the newest last of them", func(t *testing.T) {
		m, project := open(t, []Declared{morning, inbox}, ran)
		got, err := m.Log(ctx, project, "morning", 2)
		if err != nil || len(got) != 2 || !sameRun(got[0], ran[2]) || !sameRun(got[1], ran[3]) {
			t.Fatalf("Log = %+v, %v; want the runs of the 2nd and 3rd", got, err)
		}
	})

	t.Run("an automation that never ran has an empty log", func(t *testing.T) {
		m, project := open(t, []Declared{morning, inbox}, ran[:1])
		got, err := m.Log(ctx, project, "inbox", 50)
		if err != nil || len(got) != 0 {
			t.Fatalf("Log = %+v, %v; want none", got, err)
		}
	})

	t.Run("the log of a name no automation has is not found", func(t *testing.T) {
		m, project := open(t, []Declared{morning}, ran)
		if _, err := m.Log(ctx, project, "gone", 50); !errors.Is(err, automations.ErrNotFound) {
			t.Fatalf("Log(gone) = %v; want ErrNotFound", err)
		}
	})
}

func NoProject(t *testing.T, open NoProjectOpener) {
	// invariant: open gives Automations and the project root of a folder no
	// project holds: it declares no automation and has no run.
	ctx := context.Background()
	m, project := open(t)
	if got, err := m.List(ctx, project); err != nil || len(got) != 0 {
		t.Fatalf("List = %+v, %v; want none", got, err)
	}
	if _, err := m.Automation(ctx, project, "morning"); !errors.Is(err, automations.ErrNotFound) {
		t.Fatalf("Automation = %v; want ErrNotFound", err)
	}
	if got, err := m.Log(ctx, project, "", 50); err != nil || len(got) != 0 {
		t.Fatalf("Log = %+v, %v; want none", got, err)
	}
	if _, err := m.Log(ctx, project, "morning", 50); !errors.Is(err, automations.ErrNotFound) {
		t.Fatalf("Log(morning) = %v; want ErrNotFound", err)
	}
}

func same(a, b automations.Automation) bool {
	return a.Name == b.Name && a.Agent == b.Agent && a.Trigger == b.Trigger && a.LastFired.Equal(b.LastFired) && a.Fires == b.Fires && a.Failed == b.Failed
}

func sameRun(a, b automations.Run) bool {
	return a.Name == b.Name && a.Scheduled.Equal(b.Scheduled) && a.At.Equal(b.At) && a.OK == b.OK && a.Error == b.Error && a.Output == b.Output
}
