package catalog

import "strings"

const (
	RouteCloudflareTunnel = "cloudflare-tunnel"
	RouteTailscaleFunnel  = "tailscale-funnel"
	RouteLAN              = "lan"
)

type Route struct {
	// value: the word --via names it by, the owner's words for it, and for
	// what shows it is there.
	Name, Label, Seen string
}

var Routes = []Route{
	{Name: RouteCloudflareTunnel, Label: "Cloudflare Tunnel", Seen: "tunnel seen"},
	{Name: RouteTailscaleFunnel, Label: TailnetProduct + " Funnel", Seen: "Funnel lists it"},
	{Name: RouteLAN, Label: "your LAN", Seen: "listens on the LAN"},
}

func RouteNamed(name string) (Route, bool) {
	for _, r := range Routes {
		if r.Name == name {
			return r, true
		}
	}
	return Route{Name: name, Label: name}, false
}

func RouteNames() string {
	names := make([]string, len(Routes))
	for i, r := range Routes {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}

func (c Catalog) DeclaredBy(setting string) string {
	if it, ok := c.Setting(setting); ok && it.Declares != nil {
		return it.Declares.Label
	}
	return setting
}
