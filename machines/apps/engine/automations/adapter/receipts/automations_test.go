package receipts_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

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
		trigger := "cron"
		if r.Scheduled.IsZero() {
			trigger = "fs"
		}
		err := writer.Write(automation.Receipt{World: "brain", Automation: r.Name, Trigger: trigger, Scheduled: r.Scheduled, Fired: r.At, Finished: r.At, OK: r.OK, Error: r.Error, Output: r.Output, Reason: "on-time"})
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

func TestTwoGroupsDeclaringOneNameAreBothListed(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	root := t.TempDir()
	body := "version: 1\nname: site\nworlds:\n  a:\n    agents: [neo]\n    automations:\n      same:\n        on:\n          cron: \"0 6 * * *\"\n        agent: neo\n        prompt: go\n  b:\n    agents: [neo]\n    automations:\n      same:\n        on:\n          cron: \"0 7 * * *\"\n        agent: neo\n        prompt: go\n"
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := receipts.Automations{}.List(t.Context(), root)
	if err != nil || len(got) != 2 || got[0].Trigger.Spec != "0 6 * * *" || got[1].Trigger.Spec != "0 7 * * *" {
		t.Fatalf("List = %+v, %v; want both, group a first", got, err)
	}
	first, err := receipts.Automations{}.Automation(t.Context(), root, "same")
	if err != nil || first.Trigger.Spec != "0 6 * * *" {
		t.Fatalf("Automation = %+v, %v; want the first group's", first, err)
	}
}
