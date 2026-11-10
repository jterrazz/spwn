package catalog

import "path"

type LaunchOwner struct {
	Prefix string
	App    string
	Bundle string
}

var LaunchOwners = []LaunchOwner{
	// value: the apps whose launchd jobs this Mac and the owner's others were seen
	// to carry (2026-11-03); a Bundle names the app whose absence makes its jobs
	// orphans, an empty one an owner installed without a bundle (brew, a pkg).
	{Prefix: "com.github.facebook.watchman", App: "watchman"},
	{Prefix: "com.canonical.multipass", App: "multipass"},
	{Prefix: "TC3Q7MAJXF.com.adguard.safari.", App: "AdGuard", Bundle: "/Applications/AdGuard for Safari.app"},
	{Prefix: "com.adguard.", App: "AdGuard", Bundle: "/Applications/AdGuard.app"},
	{Prefix: "notion.id.", App: "Notion", Bundle: "/Applications/Notion.app"},
	{Prefix: "dev.orbstack.", App: "OrbStack", Bundle: "/Applications/OrbStack.app"},
	{Prefix: "com.docker.", App: "Docker.app", Bundle: "/Applications/Docker.app"},
	{Prefix: "com.bjango.istatmenus", App: "iStat Menus", Bundle: "/Applications/iStat Menus.app"},
	{Prefix: "com.anthropic.claudefordesktop", App: "Claude", Bundle: "/Applications/Claude.app"},
	{Prefix: "com.discord.", App: "Discord", Bundle: "/Applications/Discord.app"},
	{Prefix: "io.tailscale.", App: "Tailscale", Bundle: "/Applications/Tailscale.app"},
	{Prefix: "com.google.keystone", App: "Google Updater"},
	{Prefix: "com.microsoft.", App: "Microsoft"},
}

var LaunchSystemPrefixes = []string{
	// value: macOS's own jobs, and the per-app instances launchd names itself.
	"com.apple.", "application.", "com.openssh.",
}

const StationLaunchPrefix = "com.jterrazz.station."

const EngineLabel = StationLaunchPrefix + "engine"

const (
	// value: the label station v2 gave the engine's job; setup moves it once.
	RetiredEngineLabel = "com.jterrazz.station.daemon"
)

func EnginePlist(home string) string {
	// value: the one engine's launchd agent: its first start writes it, a stopped engine
	// keeps it, so its presence tells "stopped" from "not installed" (R9.D3).
	return agentPlist(home, EngineLabel)
}

func RetiredEnginePlist(home string) string { return agentPlist(home, RetiredEngineLabel) }

func agentPlist(home, label string) string {
	return path.Join(home, "Library", "LaunchAgents", label+".plist")
}

func LaunchOwnerOf(label string) (LaunchOwner, bool) {
	for _, o := range LaunchOwners {
		if len(label) >= len(o.Prefix) && label[:len(o.Prefix)] == o.Prefix {
			return o, true
		}
	}
	return LaunchOwner{}, false
}
