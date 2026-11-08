package pages

import (
	"slices"
	"time"

	"github.com/jterrazz/studio-station/src/ports"
)

func portsWorlds() []World {
	pro, mini := "machine."+proName, "machine."+miniMac
	return []World{
		{Name: "machines.ports", Edit: macs().then(portsAsHeard), Pages: []string{"ports", pro, mini, "machine.linux-1", "machine.review-bot"}},
		{Name: "machines.ports.https", Edit: macs().then(portsAsHeard, httpsLinksOn(miniMac)), Pages: []string{mini}},
		{Name: "machines.ports.exposed", Edit: macs().then(portsAsHeard, airPlayOnEvery), Pages: []string{"ports"}},
		{Name: "machines.ports.down", Edit: macs().then(portsAsHeard, infraDown), Pages: []string{"ports"}},
		{Name: "machines.ports.shared", Edit: macs().then(portsAsHeard, astroJustShared), Pages: []string{"machine.linux-1"}},
	}
}

// value: the fleet's ports as its Macs hear them in STORIES.md's shared
// fixture: HTTPS links off on every Mac, so each web share is an http://
// link; what listens is linux-1's Astro and Node inspector and linux-3's
// Postgres, review-bot listening on nothing.
func portsAsHeard(f *Scene) {
	quiet := map[string]bool{"agent-7f:6006": true, "review-bot:8000": true, miniMac + ":5900": true}
	for i := range f.Ports.Macs {
		h := &f.Ports.Macs[i]
		if h.Name == airName {
			continue
		}
		h.Ports = slices.DeleteFunc(h.Ports, func(p ports.Port) bool { return quiet[p.Key()] })
		serve(h, ports.HTTPSOff)
	}
}

func serve(h *ports.MacPorts, mode string) {
	h.HTTPS = mode
	for j, s := range h.Shared {
		web := s.HTTP
		r := ports.Reservation{VM: s.VM, GuestPort: s.GuestPort, MacPort: s.Reserved, HTTP: &web}
		h.Shared[j].URLs = ports.URLs(r, h.PublicName, mode, h.Serve == ports.ServeOn)
	}
}

func httpsLinksOn(mac string) func(*Scene) {
	return func(f *Scene) {
		for i := range f.Ports.Macs {
			if f.Ports.Macs[i].Name == mac {
				serve(&f.Ports.Macs[i], ports.HTTPSPort)
			}
		}
	}
}

// value: ports.shared-new: linux-1's Astro shared two seconds ago, its
// link the next port of This Mac's range.
func astroJustShared(f *Scene) {
	h := &f.Ports.Macs[0]
	h.Shared = append(h.Shared, ports.SharedPort{VM: "linux-1", GuestPort: 4321, Reserved: 40003, State: ports.PhaseShared, Process: "astro", HTTP: true,
		ReservedAt: storyNow.Add(-2 * time.Second)})
	serve(h, h.HTTPS)
}
