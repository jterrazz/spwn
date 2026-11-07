package catalog_test

import (
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
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

// invariant: one owner per port: a port is never named twice, and every
// profile's port has its name.
func TestEveryPortIsNamedOnceAndEveryProfileHasItsName(t *testing.T) {
	seen := map[int]bool{}
	for _, s := range catalog.PortNames {
		if _, known := catalog.KnownPortOf(s.Port, ""); known || seen[s.Port] {
			t.Errorf("port %d is named twice", s.Port)
		}
		seen[s.Port] = true
	}
	for _, p := range catalog.Profiles {
		if !seen[p.Port] {
			t.Errorf("profile %s: port %d has no name", p.Name, p.Port)
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
