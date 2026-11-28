package catalog_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: an app whose Ubuntu method requires a session its macOS method
// does not, and the session, which lives on Linux alone.
func methodRequiring(t *testing.T) catalog.Catalog {
	t.Helper()
	c, err := catalogOf(map[string]string{
		"git/driver.yaml":  "package: station.package/v1\nkind: app\nname: driver\ndescription: d\nprovides: [driver]\nmethods:\n    - os: macos\n      brew: driver\n    - os: ubuntu\n      needs: [sudo]\n      requires: [session]\n      apt: {packages: [driver]}\n",
		"git/session.yaml": "package: station.package/v1\nkind: app\nname: session\ndescription: d\nmethods:\n    - family: debian\n      needs: [sudo]\n      apt: {packages: [session]}\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func requiredOf(links []catalog.Link) []string {
	var out []string
	for _, l := range links {
		if l.Kind == catalog.LinkRequires {
			out = append(out, l.Packages...)
		}
	}
	return out
}

func TestAMethodsRequiresHoldWhereThatMethodIsTheOneChosen(t *testing.T) {
	c := methodRequiring(t)
	p, _ := c.Package("driver")
	if got := requiredOf(c.Links(p.Package, "apt", catalog.System{OS: catalog.OSUbuntu, Version: "24.04"})); !slices.Equal(got, []string{"session"}) {
		t.Errorf("driver by apt on Ubuntu requires %v, want its method's session", got)
	}
	if got := requiredOf(c.Links(p.Package, "brew", catalog.System{OS: catalog.OSMacOS, Version: "26"})); got != nil {
		t.Errorf("driver by brew on macOS requires %v, want nothing: the session is its Ubuntu method's", got)
	}
	app, _ := c.App("driver")
	for kind, want := range map[catalog.MachineKind][]string{catalog.Mac: nil, catalog.MacOS: nil, catalog.Linux: {"session"}} {
		in, _ := app.InstallOn(kind)
		if got := requiredOf(c.InstallLinks(app, in, kind)); !slices.Equal(got, want) {
			t.Errorf("driver's install on %s requires %v, want %v", kind, got, want)
		}
	}
}
