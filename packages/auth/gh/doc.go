// Package gh implements `spwn auth login github` — host-side
// gh-cli auth import / device-flow login, with the resulting
// hosts.yml landing under ~/.spwn/credentials/gh so spwn can
// bind-mount it into every world. One auth, two consumers: `gh`
// cobra commands and the `gh-mcp` wrapper that drives the
// MCP-over-stdio GitHub server.
//
// The mode of acquiring credentials differs from packages/auth/mcp:
// MCP servers expose OAuth via the protocol itself, so the helper
// container drives PKCE end-to-end. GitHub's `gh` ships its own
// device-flow login that prints a code + URL — there's no callback
// to forward, so spwn either (a) imports an existing host login by
// extracting the token and writing a plaintext hosts.yml, or (b)
// runs `gh auth login --web` directly inline. Most users have
// already done `gh auth login` on their machine, so import is the
// happy path.
package gh
