package mcp

import (
	"fmt"
	"sort"
	"strings"
)

// Provider describes one hosted MCP server we know how to log in to.
// The URL is the streamable-HTTP endpoint mcp2cli connects to; OAuth
// metadata (authorize/token URLs, dynamic client registration) is
// discovered from the server itself, so we don't pin client_id /
// client_secret here.
type Provider struct {
	Name string
	URL  string
}

// Registry is the set of MCP providers spwn ships login support for.
// Keep entries minimal — anything that requires a custom OAuth
// scope or DCR client name belongs as fields on Provider, not as
// a special case at the call site.
var Registry = map[string]Provider{
	"notion": {Name: "notion", URL: "https://mcp.notion.com/mcp"},
}

// Lookup returns the Provider for name (case-insensitive) and a
// boolean ok flag.
func Lookup(name string) (Provider, bool) {
	p, ok := Registry[strings.ToLower(strings.TrimSpace(name))]
	return p, ok
}

// Names returns the sorted list of known provider names. Used for
// help text + error messages.
func Names() []string {
	out := make([]string, 0, len(Registry))
	for k := range Registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// UnknownProviderError returns a help-friendly error for an
// unrecognised name.
func UnknownProviderError(raw string) error {
	return fmt.Errorf("unknown MCP provider %q; try one of: %s", raw, strings.Join(Names(), ", "))
}
