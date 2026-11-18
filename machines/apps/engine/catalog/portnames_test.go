package catalog_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// invariant: SPEC B5.17: a port is known by its software and wears its logo,
// the system's own by the name its process means, else by its number.
func TestAPortIsKnownByItsSoftware(t *testing.T) {
	cases := []struct {
		port       int
		process    string
		name, logo string
	}{
		{5173, "node", "Vite", "vite"},
		{4321, "node", "Astro", "astro"},
		{5432, "postgres", "Postgres", "postgres"},
		{443, "traefik", "Traefik", "traefik"},
		{6443, "k3s", "Kubernetes API", "kubernetes"},
		{9229, "node", "Node inspector", "node"},
		{6006, "node", "Storybook", "storybook"},
		{5000, "ControlCenter", "AirPlay Receiver", ""},
		{8080, "python", "Port 8080", ""},
	}
	for _, c := range cases {
		if s := catalog.PortNameOf(c.port, c.process); s.Name != c.name || s.Logo != c.logo {
			t.Errorf("%d %s: %q %q, want %q %q", c.port, c.process, s.Name, s.Logo, c.name, c.logo)
		}
	}
}

// invariant: one owner per port: a port is never named twice, every port
// a package's env is set for has its name, and a package names a port as the
// table does.
func TestEveryPortIsNamedOnceAndEveryPackagePortAgrees(t *testing.T) {
	seen := map[int]string{}
	for _, s := range catalog.PortNames {
		if _, known := catalog.KnownPortOf(s.Port, ""); known || seen[s.Port] != "" {
			t.Errorf("port %d is named twice", s.Port)
		}
		seen[s.Port] = s.Name
	}
	for _, p := range builtin(t).Ports() {
		if name, ok := seen[p.Port]; (len(p.Env) > 0 && !ok) || (ok && name != p.Label) {
			t.Errorf("%s: port %d carries the label %q, named %q", p.Package, p.Port, p.Label, name)
		}
	}
}

// invariant: an exposed listener names the app its process is.
func TestAnExposedListenerNamesItsApp(t *testing.T) {
	if got := catalog.AppOf("ControlCenter"); got != "AirPlay Receiver" {
		t.Errorf("ControlCenter is %q", got)
	}
	if got := catalog.AppOf("python"); got != "python" {
		t.Errorf("python is %q", got)
	}
}
