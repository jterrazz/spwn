package machinefile_test

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
)

func tree(layer machinefile.Layer, files map[string]string) machinefile.Tree {
	t := machinefile.Tree{Layer: layer, Root: string(layer)}
	for p, body := range files {
		t.Files = append(t.Files, machinefile.File{Path: p, Body: []byte(body)})
	}
	return t
}

const head = "machinefile: station.machinefile/v1\n"

// invariant: a VM's rights are its Machinefile's ports, else those of what
// it extends, the overlay's winning whole; no Machinefile reads none.
func TestRightsAreReadFromTheChainAndTheOverlayWins(t *testing.T) {
	user := tree(machinefile.User, map[string]string{
		"machinefiles/api.yaml":       head + "name: api\nextends: [base]\nmachine: { kind: workspace }\n",
		"machinefiles/base.yaml":      head + "name: base\nports: { open: 3000-3999, to: private }\n",
		"machinefiles/web.yaml":       head + "name: web\nmachine: { kind: workspace }\nports: { open: 5173, to: local }\n",
		"machinefiles/web.local.yaml": head + "overrides: web\nports: { open: [3000], to: public }\n",
		"machinefiles/broken.yaml":    head + "name: broken\nports: { open: 3000-3999, to: public }\n",
	})
	rights, err := machinefile.RightsOf([]machinefile.Tree{user}, "api")
	if err != nil || rights == nil || rights.To != catalog.AudiencePrivate || rights.Open[0] != (catalog.PortSpan{From: 3000, To: 3999}) {
		t.Fatalf("api's rights = %+v, %v; want its base's 3000-3999 to private", rights, err)
	}
	rights, err = machinefile.RightsOf([]machinefile.Tree{user}, "web")
	if err != nil || rights == nil || rights.To != catalog.AudiencePublic {
		t.Fatalf("web's rights = %+v, %v; want its overlay's, to public", rights, err)
	}
	if rights, err = machinefile.RightsOf([]machinefile.Tree{user}, "linux-1"); err != nil || rights != nil {
		t.Fatalf("a VM with no Machinefile reads %+v, %v; want none", rights, err)
	}
	var refused *machinefile.RefusedFileError
	if _, err = machinefile.RightsOf([]machinefile.Tree{user}, "broken"); !errors.As(err, &refused) {
		t.Fatalf("a public range reads %v; want the refusal of its file", err)
	}
}
