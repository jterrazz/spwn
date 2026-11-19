package catalog_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const machinefileHead = "machinefile: station.machinefile/v1\n"

func decodeMachinefile(t *testing.T, file, body string) (catalog.Machinefile, error) {
	t.Helper()
	return catalog.DecodeMachinefile("~/.config/station/machinefiles/"+file, []byte(machinefileHead+body))
}

func machinefileRefused(t *testing.T, file, body, want string) {
	t.Helper()
	_, err := decodeMachinefile(t, file, body)
	var refused *catalog.FileError
	if !errors.As(err, &refused) || !strings.Contains(err.Error(), want) {
		t.Errorf("%s decoded with %v, want a refusal saying %q", body, err, want)
	}
}

// invariant: audiences gives a port its own audience, a port from 1 to
// 65535 and one of the three audiences, and its public ports count with
// those open gives the internet, three at most.
func TestAudiencesAreCheckedAndCountTheirPublicPorts(t *testing.T) {
	mf, err := decodeMachinefile(t, "web.machine.yaml", "name: web\nports: { open: 3000-3999, to: private, audiences: { 5432: local, 3001: public } }\n")
	if err != nil || mf.Ports.Audiences[5432] != catalog.AudienceLocal || mf.Ports.Audiences[3001] != catalog.AudiencePublic {
		t.Fatalf("decoded %+v, %v; want 5432 local and 3001 public", mf.Ports, err)
	}
	if mf, err = decodeMachinefile(t, "web.machine.yaml", "name: web\nports: { audiences: { 5432: local } }\n"); err != nil || len(mf.Ports.Open) != 0 {
		t.Errorf("audiences alone decoded %+v, %v; want a file that opens 5432 alone", mf.Ports, err)
	}
	machinefileRefused(t, "web.machine.yaml", "name: web\nports: { audiences: { 70000: local } }\n", "audiences 70000: a port from 1 to 65535")
	machinefileRefused(t, "web.machine.yaml", "name: web\nports: { audiences: { 5432: everyone } }\n", `audiences 5432: "everyone" is none of local, private, public`)
	machinefileRefused(t, "web.machine.yaml", "name: web\nports: { open: [3000, 3001, 3002], to: public, audiences: { 4000: public } }\n", "public opens 3 ports at most")
	machinefileRefused(t, "web.machine.yaml", "name: web\nports: { audiences: { 1: public, 2: public, 3: public, 4: public } }\n", "public opens 3 ports at most")
	if _, err := decodeMachinefile(t, "web.machine.yaml", "name: web\nports: { open: [3000, 3001, 3002], to: public, audiences: { 3002: local, 4000: public } }\n"); err != nil {
		t.Errorf("a public port given local leaves room for one of audiences: %v", err)
	}
}
