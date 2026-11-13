package contract

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type PublicRoute interface {
	// invariant: the word --via names the route by.
	Name() string
	// invariant: seen only when the route's own evidence is there on the
	// machine the port is declared on; nothing seen is an answer, never an
	// error.
	Verify(ctx context.Context, d ports.Declaration, on ports.Machine) (ports.Verified, error)
}

type PublicRoutes []PublicRoute

func (rs PublicRoutes) Named(name string) (PublicRoute, bool) {
	i := slices.IndexFunc(rs, func(r PublicRoute) bool { return r.Name() == name })
	if i < 0 {
		return nil, false
	}
	return rs[i], true
}

func (rs PublicRoutes) Names() []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Name()
	}
	return out
}

func (rs PublicRoutes) Verify(ctx context.Context, ms []ports.Machine, ps []ports.Port) []ports.Port {
	// invariant: every declared port is told what its route shows; a route
	// station no longer knows, or one that cannot look, shows nothing.
	out := slices.Clone(ps)
	for i, p := range out {
		if p.Declared == nil {
			continue
		}
		v := ports.Verified{Missing: "station knows no route " + p.Declared.Via}
		j := slices.IndexFunc(ms, func(m ports.Machine) bool { return m.Ref == p.Machine })
		if r, ok := rs.Named(p.Declared.Via); ok && j >= 0 {
			if got, err := r.Verify(ctx, *p.Declared, ms[j]); err == nil {
				v = got
			} else {
				v.Missing = err.Error()
			}
		}
		out[i].Verified = &v
	}
	return out
}
