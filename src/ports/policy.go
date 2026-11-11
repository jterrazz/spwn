package ports

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
)

const (
	HTTPSPort = "port"
	HTTPSPath = "path"
	// value: HTTPS links turned off on this Mac: every share is carried raw,
	// an HTTP server's link http://.
	HTTPSOff = "off"
)

func HTTPSLinks(mode string) bool { return mode != HTTPSOff }

type Policy struct {
	Low, High, MaxPerVM, MinPort int
	Ignored                      []int
	HTTPS                        string
	// value: the ports the owner stopped sharing, as <vm>:<port>: a listen
	// there is not shared again until the owner shares it.
	NotShared []string
}

func PolicyOf(s *PortsGroup) Policy {
	d := catalog.PortPolicy
	p := Policy{Low: d.RangeLow, High: d.RangeHigh, MaxPerVM: d.MaxPerVM, MinPort: d.MinPort, Ignored: d.Ignored, HTTPS: d.HTTPS}
	if s == nil {
		return p
	}
	if lo, hi, ok := strings.Cut(s.Range, "-"); ok {
		l, errL := strconv.Atoi(strings.TrimSpace(lo))
		h, errH := strconv.Atoi(strings.TrimSpace(hi))
		if errL == nil && errH == nil && l > 0 && h >= l && h < 65536 {
			p.Low, p.High = l, h
		}
	}
	if s.MaxPerVM > 0 {
		p.MaxPerVM = s.MaxPerVM
	}
	if s.MinPort > 0 {
		p.MinPort = s.MinPort
	}
	if s.IgnoredPorts != nil {
		p.Ignored = s.IgnoredPorts
	}
	if s.HTTPS == HTTPSPath || s.HTTPS == HTTPSOff {
		p.HTTPS = s.HTTPS
	}
	p.NotShared = s.NotShared
	return p
}

func (p Policy) Ignores(port int) bool { return port < p.MinPort || slices.Contains(p.Ignored, port) }

func (p Policy) InRange(port int) bool { return port >= p.Low && port <= p.High }

func (p Policy) Stopped(vm string, port int) bool {
	return slices.Contains(p.NotShared, vm+":"+strconv.Itoa(port))
}

func ProfileOf(profiles []catalog.Profile, port int) (catalog.Profile, bool) {
	for _, pr := range profiles {
		if pr.Port == port {
			return pr, true
		}
	}
	return catalog.Profile{}, false
}
