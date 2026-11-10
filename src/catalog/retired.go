package catalog

const TailnetCLI = "tailscale"

// value: what the round-2 userspace tailnet daemon left behind, by name:
// setup's layout step retires exactly these.
const (
	RetiredNodeDir    = "tailscaled"
	RetiredLegacyDir  = "tailscale"
	RetiredNodeNote   = "the userspace tailscaled stops in the background"
	RetiredNodePID    = "tailscaled.pid"
	RetiredAwakePID   = "caffeinate.pid"
	RetiredNodeState  = "tailscaled.state"
	retiredNodeLog    = "tailscaled.log"
	retiredNodeSocket = "tailscaled.sock"
	retiredAwakeLog   = "caffeinate.log"
	nodeConsole       = "https://login.tailscale.com/admin/machines"
	OAuthConsole      = "https://login.tailscale.com/admin/settings/oauth"
)

func RetiredNodeFiles() []string {
	return []string{RetiredNodeState, retiredNodeLog, retiredNodeSocket, RetiredNodePID, RetiredAwakePID, retiredAwakeLog}
}

func RetiredNodeDetail(thisMac string) string {
	return "userspace tailscaled retired; its node \"" + thisMac + "\" stays at " + nodeConsole + " until you remove it"
}

func SSHCommand(user, mac string) string {
	if user != "" {
		return "ssh " + user + "@" + mac
	}
	return "ssh " + mac
}
