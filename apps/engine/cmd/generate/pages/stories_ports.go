package pages

import (
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

func portsWorlds() []World {
	pro, mini := "machine."+proName, "machine."+miniMac
	return []World{
		{Name: "machines.ports", Edit: macs().then(portsAsHeard), Pages: []string{"ports", pro, mini, "machine.linux-1", "machine.review-bot"}},
		{Name: "machines.ports.https", Edit: macs().then(portsAsHeard, httpsLinksOn(miniMac)), Pages: []string{mini}},
		{Name: "machines.ports.exposed", Edit: macs().then(portsAsHeard, airPlayOnEvery), Pages: []string{"ports"}},
		{Name: "machines.ports.down", Edit: macs().then(portsAsHeard, infraDown), Pages: []string{"ports"}},
		{Name: "machines.ports.quiet", Edit: macs().then(portsAsHeard, quietOnLinux1), Pages: []string{"ports", "machine.linux-1"}},
		{Name: "machines.ports.audiences", Edit: macs().then(portsAsHeard, sharedByAudience), Pages: []string{"ports", "machine.linux-1"}},
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

// value: ports.quiet: linux-1 listens on 3 ports of its system's temporary
// range, told as one line.
func quietOnLinux1(f *Scene) {
	f.Ports.Macs[0].Quiet = []ports.Quiet{{VM: "linux-1", Count: 3}}
}

// value: ports.shared-new: linux-1's Astro shared two seconds ago, its
// link the next port of This Mac's range.
func astroJustShared(f *Scene) {
	h := &f.Ports.Macs[0]
	h.Shared = append(h.Shared, ports.SharedPort{VM: "linux-1", GuestPort: 4321, Reserved: 40003, State: ports.PhaseShared, Process: "astro", HTTP: true,
		ReservedAt: storyNow.Add(-2 * time.Second)})
	serve(h, h.HTTPS)
}

// value: ports.audiences: linux-1 shares one port with each audience and
// flags one outside its rights: 3001 public on a Funnel port, 4000 local to
// its Mac, 8443 listening, not allowed.
func sharedByAudience(f *Scene) {
	h := &f.Ports.Macs[0]
	h.Shared = append(h.Shared,
		ports.SharedPort{VM: "linux-1", GuestPort: 3001, Reserved: 40003, State: ports.PhaseShared, Process: "node", HTTP: true, To: "public",
			URLs: ports.URLSet{Public: "https://" + h.PublicName + ":8443"}},
		ports.SharedPort{VM: "linux-1", GuestPort: 4000, Reserved: 40004, State: ports.PhaseShared, Process: "node", HTTP: true, To: "local",
			URLs: ports.URLSet{Public: "http://127.0.0.1:40004"}},
		ports.SharedPort{VM: "linux-1", GuestPort: 8443, State: ports.PhaseNotAllowed, Process: "node", Detail: "listens, not allowed (open: 3000-4999)"})
}
