package receipts_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"spwn.sh/packages/automation"

	"github.com/jterrazz/studio-station/apps/engine/automations"
	"github.com/jterrazz/studio-station/apps/engine/automations/adapter/receipts"
	"github.com/jterrazz/studio-station/apps/engine/automations/contract"
	"github.com/jterrazz/studio-station/apps/engine/automations/contract/contracttest"
)

func manifest(declared []contracttest.Declared) string {
	var body strings.Builder
	body.WriteString("version: 1\nname: site\nworlds:\n  brain:\n    agents: [neo, trinity]\n    automations:\n")
	for _, d := range declared {
		body.WriteString("      " + d.Name + ":\n        on:\n")
		if d.Kind == automations.Cron {
			body.WriteString("          cron: " + strconv.Quote(d.Spec) + "\n")
		} else {
			body.WriteString("          fs:\n            path: " + strconv.Quote(d.Spec) + "\n")
		}
		body.WriteString("        agent: " + d.Agent + "\n        prompt: go\n")
	}
	return body.String()
}

func open(t *testing.T, declared []contracttest.Declared, ran []automations.Run) (opened contract.Automations, project string) {
	t.Setenv("SPWN_HOME", t.TempDir())
	project = t.TempDir()
	body := manifest(declared)
	if len(declared) == 0 {
		body = "version: 1\nname: site\nworlds:\n  brain:\n    agents: [neo]\n"
	}
	if err := os.WriteFile(filepath.Join(project, "spwn.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := automation.NewFileReceiptWriter(filepath.Join(project, ".spwn", "runs.jsonl"))
	for _, r := range ran {
		err := writer.Write(automation.Receipt{World: "brain", Automation: r.Name, Trigger: r.Trigger, Scheduled: r.Scheduled, Fired: r.At, Finished: r.At, DurationMS: r.DurationMS, OK: r.OK, Error: r.Error, Output: r.Output, Reason: r.Cause, Missed: r.Missed, EventPaths: make([]string, r.Paths)})
		if err != nil {
			t.Fatal(err)
		}
	}
	return receipts.Automations{}, project
}

func TestAutomations(t *testing.T) { contracttest.Automations(t, open) }

func TestAFolderNoProjectHoldsDeclaresNone(t *testing.T) {
	contracttest.NoProject(t, func(t *testing.T) (contract.Automations, string) {
		t.Setenv("SPWN_HOME", t.TempDir())
		return receipts.Automations{}, ""
	})
}

func TestARootWithoutAManifestDeclaresNone(t *testing.T) {
	contracttest.NoProject(t, func(t *testing.T) (contract.Automations, string) {
		t.Setenv("SPWN_HOME", t.TempDir())
		return receipts.Automations{}, t.TempDir()
	})
}

func twoGroups(t *testing.T) string {
	t.Helper()
	t.Setenv("SPWN_HOME", t.TempDir())
	root := t.TempDir()
	body := "version: 1\nname: site\nworlds:\n  a:\n    agents: [neo]\n    automations:\n      same:\n        on:\n          cron: \"0 6 * * *\"\n        agent: neo\n        prompt: go\n  b:\n    agents: [neo]\n    automations:\n      same:\n        on:\n          cron: \"0 7 * * *\"\n        agent: neo\n        prompt: go\n"
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	writer := automation.NewFileReceiptWriter(filepath.Join(root, ".spwn", "runs.jsonl"))
	at := time.Date(2026, time.May, 1, 6, 0, 0, 0, time.UTC)
	for i, group := range []string{"a", "b", "b"} {
		if err := writer.Write(automation.Receipt{World: group, Automation: "same", Trigger: "cron", Fired: at.Add(time.Duration(i) * time.Hour), OK: true, Reason: "on-time"}); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestTwoGroupsDeclaringOneNameAreBothListedWithTheirGroup(t *testing.T) {
	root := twoGroups(t)
	got, err := receipts.Automations{}.List(t.Context(), root)
	if err != nil || len(got) != 2 || got[0].Group != "a" || got[1].Group != "b" || !got[0].Shared || !got[1].Shared {
		t.Fatalf("List = %+v, %v; want both, shared, group a first", got, err)
	}
	if got[0].Fires != 1 || got[1].Fires != 2 {
		t.Fatalf("List = %+v; want each group's own runs counted", got)
	}
}

func TestANameTwoGroupsDeclareIsNotUniqueUntilQualified(t *testing.T) {
	root := twoGroups(t)
	var many *automations.NotUniqueError
	if _, err := (receipts.Automations{}).Automation(t.Context(), root, "same"); !errors.As(err, &many) || !slices.Equal(many.Groups, []string{"a", "b"}) {
		t.Fatalf("Automation(same) = %v; want not unique in a, b", err)
	}
	if _, err := (receipts.Automations{}).Log(t.Context(), root, "same", 0); !errors.As(err, &many) {
		t.Fatalf("Log(same) = %v; want not unique", err)
	}
	b, err := receipts.Automations{}.Automation(t.Context(), root, "b/same")
	if err != nil || b.Group != "b" || b.Trigger.Spec != "0 7 * * *" {
		t.Fatalf("Automation(b/same) = %+v, %v; want group b", b, err)
	}
	runs, err := receipts.Automations{}.Log(t.Context(), root, "b/same", 0)
	if err != nil || len(runs) != 2 || runs[0].Group != "b" || runs[1].Group != "b" {
		t.Fatalf("Log(b/same) = %+v, %v; want the two runs of b", runs, err)
	}
	if _, err := (receipts.Automations{}).Automation(t.Context(), root, "c/same"); !errors.Is(err, automations.ErrNotFound) {
		t.Fatalf("Automation(c/same) = %v; want ErrNotFound", err)
	}
}

func TestAManifestThatDoesNotReadIsOneInvalidProjectError(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"), []byte("version: 1\nworlds: [not, a, map]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var bad *automations.InvalidProjectError
	if _, err := (receipts.Automations{}).List(t.Context(), root); !errors.As(err, &bad) || bad.Reason == "" || bad.Path != filepath.Join(root, "spwn.yaml") {
		t.Fatalf("List = %v; want an InvalidProjectError naming the manifest and its error", err)
	}
	if _, err := (receipts.Automations{}).Automation(t.Context(), root, "morning"); !errors.As(err, &bad) {
		t.Fatalf("Automation = %v; want an InvalidProjectError", err)
	}
	if _, err := (receipts.Automations{}).Log(t.Context(), root, "morning", 0); !errors.As(err, &bad) {
		t.Fatalf("Log(morning) = %v; want an InvalidProjectError", err)
	}
	runs, err := receipts.Automations{}.Log(t.Context(), root, "", 0)
	if err != nil || len(runs) != 0 {
		t.Fatalf("Log() = %+v, %v; want the runs of the file with no manifest read", runs, err)
	}
}
