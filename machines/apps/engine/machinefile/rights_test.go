package machinefile_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func tree(origin machinefile.OriginKind, files map[string]string) machinefile.Tree {
	t := machinefile.Tree{Kind: origin, Root: string(origin)}
	for p, body := range files {
		t.Files = append(t.Files, machinefile.File{Path: p, Body: []byte(body)})
	}
	return t
}

const head = "machinefile: station.machinefile/v1\n"

// invariant: a VM's rights are its Machinefile's ports, else those of what
// it extends, the overlay's changing them key by key; no Machinefile reads
// none.
func TestRightsAreReadFromTheChainAndTheOverlayWins(t *testing.T) {
	user := tree(machinefile.User, map[string]string{
		"machinefiles/api.machine.yaml":    head + "name: api\nextends: [base]\nmachine: { kind: workspace }\n",
		"machinefiles/base.machine.yaml":   head + "name: base\nports: { open: 3000-3999, to: private }\n",
		"machinefiles/web.machine.yaml":    head + "name: web\nmachine: { kind: workspace }\nports: { open: 5173, to: local }\n",
		"machinefiles/web.local.yaml":      head + "overrides: web\nports: { open: [3000], to: public }\n",
		"machinefiles/broken.machine.yaml": head + "name: broken\nports: { open: 3000-3999, to: public }\n",
		"machinefiles/plain.machine.yaml":  head + "name: plain\nmachine: { kind: workspace }\n",
	})
	rights, by, err := machinefile.RightsOf([]machinefile.Tree{user}, "api", "")
	if err != nil || rights == nil || rights.To != catalog.AudiencePrivate || rights.Open[0] != (catalog.PortSpan{From: 3000, To: 3999}) || by != "base" {
		t.Fatalf("api's rights = %+v by %q, %v; want its base's 3000-3999 to private, by base", rights, by, err)
	}
	rights, by, err = machinefile.RightsOf([]machinefile.Tree{user}, "web", "")
	if err != nil || rights == nil || rights.To != catalog.AudiencePublic || len(rights.Open) != 1 || rights.Open[0].From != 3000 || by != "web.local" {
		t.Fatalf("web's rights = %+v by %q, %v; want its overlay's, 3000 to public, by web.local", rights, by, err)
	}
	if rights, by, err = machinefile.RightsOf([]machinefile.Tree{user}, "linux-1", ""); err != nil || rights != nil || by != "" {
		t.Fatalf("a VM with no Machinefile reads %+v by %q, %v; want none", rights, by, err)
	}
	if rights, by, err = machinefile.RightsOf([]machinefile.Tree{user}, "plain", ""); err != nil || rights != nil || by != "plain" {
		t.Fatalf("a Machinefile that says no ports reads %+v by %q, %v; want none, by plain", rights, by, err)
	}
	var refused *machinefile.RefusedFileError
	if _, _, err = machinefile.RightsOf([]machinefile.Tree{user}, "broken", ""); !errors.As(err, &refused) {
		t.Fatalf("a public range reads %v; want the refusal of its file", err)
	}
}

func TestARefusedChainOpensNoPort(t *testing.T) {
	user := tree(machinefile.User, map[string]string{
		"machinefiles/a.machine.yaml":     head + "name: a\napps: [{ name: fd, manager: skill }]\n",
		"machinefiles/b.machine.yaml":     head + "name: b\napps: [fd]\n",
		"machinefiles/web.machine.yaml":   head + "name: web\nmachine: { kind: workspace, system: { os: ubuntu } }\nextends: [a, b]\nports: { open: 5173, to: local }\n",
		"machinefiles/mac.machine.yaml":   head + "name: mac\nmachine: { kind: workspace, system: { os: macos, version: '26' } }\nextends: [a, b]\nports: { open: 5173, to: local }\n",
		"machinefiles/blank.machine.yaml": head + "name: blank\nmachine: { kind: workspace }\nextends: [a, b]\nports: { open: 5173, to: local }\n",
		"machinefiles/c.machine.yaml":     head + "name: c\naccounts: { signins: [claude] }\n",
		"machinefiles/d.machine.yaml":     head + "name: d\naccounts: { grants: [claude] }\n",
		"machinefiles/ai.machine.yaml":    head + "name: ai\nmachine: { kind: workspace, system: { os: ubuntu } }\nextends: [c, d]\nports: { open: 5173, to: local }\n",
		"machinefiles/ai.local.yaml":      head + "overrides: ai\naccounts: { grants: [claude] }\n",
	})
	builtin := machinefile.Tree{Kind: machinefile.Builtin, Root: "built-in"}
	for p, body := range packages() {
		builtin.Files = append(builtin.Files, machinefile.File{Path: p, Body: []byte(body)})
	}
	trees := []machinefile.Tree{user, builtin}
	var conflict *machinefile.SiblingConflictError
	if rights, _, err := machinefile.RightsOf(trees, "web", ""); !errors.As(err, &conflict) || conflict.Unjudged {
		t.Errorf("web's rights, its siblings giving fd two ways on ubuntu: %+v, %v; want the siblings refused", rights, err)
	}
	if rights, _, err := machinefile.RightsOf(trees, "mac", ""); err != nil || rights == nil {
		t.Errorf("mac's rights, skill being fd's default on macOS 26: %+v, %v; want its ports", rights, err)
	}
	if _, _, err := machinefile.RightsOf(trees, "blank", ""); !errors.As(err, &conflict) || !conflict.Unjudged {
		t.Errorf("a chain that says no system: %v, want refused as unjudged", err)
	}
	if rights, _, err := machinefile.RightsOf(trees, "ai", ""); err != nil || rights == nil {
		t.Errorf("ai, whose overlay settles its siblings' account: %+v, %v; want its ports", rights, err)
	}
	if _, _, err := machinefile.RightsOf(trees, "ai", "linux-1"); err != nil {
		t.Errorf("ai followed by linux-1, the overlay still read: %v, want its ports", err)
	}
}

// invariant: an overlay's ports change only the keys it says, and a merge
// that cannot be read is the error.
func TestAnOverlayChangesTheRightsKeyByKey(t *testing.T) {
	user := tree(machinefile.User, map[string]string{
		"machinefiles/web.machine.yaml": head + "name: web\nmachine: { kind: workspace }\nports: { open: 3000-3999, to: private, audiences: { 5432: local } }\n",
		"machinefiles/web.local.yaml":   head + "overrides: web\nports: { except: [3306], audiences: { 8080: public } }\n",
		"machinefiles/api.machine.yaml": head + "name: api\nmachine: { kind: workspace }\nports: { open: 3000-3999, to: private }\n",
		"machinefiles/api.local.yaml":   head + "overrides: api\nports: { to: public }\n",
	})
	rights, by, err := machinefile.RightsOf([]machinefile.Tree{user}, "web", "")
	if err != nil || rights.To != catalog.AudiencePrivate || len(rights.Open) != 1 || len(rights.Except) != 1 || rights.Audiences[5432] != catalog.AudienceLocal || rights.Audiences[8080] != catalog.AudiencePublic || by != "web.local" {
		t.Fatalf("web's rights = %+v by %q, %v; want its own with the overlay's except and audiences", rights, by, err)
	}
	var refused *machinefile.RefusedFileError
	if _, _, err := machinefile.RightsOf([]machinefile.Tree{user}, "api", ""); !errors.As(err, &refused) || !strings.Contains(err.Error(), "public opens ports one by one") {
		t.Fatalf("a public range merged reads %v; want the overlay refused", err)
	}
}

func TestAMachinesOwnChangesAreItsAlone(t *testing.T) {
	user := tree(machinefile.User, map[string]string{
		"machinefiles/dev.machine.yaml":   head + "name: dev\nmachine: { kind: workspace }\nports: { open: 3000-3999, to: private }\n",
		"machinefiles/linux-1.local.yaml": head + "overrides: dev\nports: { audiences: { 5432: local } }\n",
		"machinefiles/other.local.yaml":   head + "overrides: web\nports: { to: public }\n",
	})
	rights, by, err := machinefile.RightsOf([]machinefile.Tree{user}, "dev", "linux-1")
	if err != nil || rights.To != catalog.AudiencePrivate || rights.Audiences[5432] != catalog.AudienceLocal || by != "linux-1.local" {
		t.Fatalf("linux-1's rights = %+v by %q, %v; want dev's with its own audience for 5432", rights, by, err)
	}
	for _, vm := range []string{"agent-7f", "other", ""} {
		if rights, by, err := machinefile.RightsOf([]machinefile.Tree{user}, "dev", vm); err != nil || len(rights.Audiences) != 0 || by != "dev" {
			t.Errorf("%q's rights = %+v by %q, %v; want dev's alone", vm, rights, by, err)
		}
	}
}
