package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/ports"
)

type SetHTTPSIn struct {
	On bool
}

type SetHTTPSOut struct {
	// value: the mode this Mac's shares are now served in: port, path or off.
	HTTPS string
}

var SetHTTPS = call.Call[Deps, SetHTTPSIn, SetHTTPSOut]{
	// invariant: SPEC B5.13: HTTPS links are a preference of the Mac that
	// shares the ports, kept in its ports group; on keeps a path mode the
	// owner chose, off serves every share raw with an http:// link. Its
	// engine serves the shares again at its next tick.
	Name: "ports.set-https", Section: section.Ports, Where: call.AtTarget, Applies: ports.LinksOnAMac,
	Run: func(_ context.Context, d Deps, in SetHTTPSIn, _ call.Progress) (SetHTTPSOut, error) {
		g, err := d.Prefs.Get()
		if err != nil {
			return SetHTTPSOut{}, err
		}
		if g == nil {
			g = &ports.PortsGroup{}
		}
		switch {
		case !in.On:
			g.HTTPS = ports.HTTPSOff
		case g.HTTPS != ports.HTTPSPath:
			g.HTTPS = ""
		}
		if err := d.Prefs.Set(g); err != nil {
			return SetHTTPSOut{}, err
		}
		return SetHTTPSOut{HTTPS: ports.PolicyOf(g).HTTPS}, nil
	},
}
