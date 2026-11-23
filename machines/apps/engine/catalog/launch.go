package catalog

import (
	"path"
	"strings"
)

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

var OwnLaunchPrefixes = []string{
	// value: the prefixes of every launchd label this product writes: the
	// bare engine's job and the menu bar's under StationLaunchPrefix,
	// and the jobs Spwn.app registers under its own bundle identifier
	// (ADR-073).
	StationLaunchPrefix, WindowBundle + ".",
}

func IsOwnLaunchLabel(label string) bool {
	for _, prefix := range OwnLaunchPrefixes {
		if strings.HasPrefix(label, prefix) {
			return true
		}
	}
	return false
}

const EngineLabel = StationLaunchPrefix + "engine"

func EnginePlist(home string) string {
	// value: the bare engine's launchd agent, on a Mac without Spwn.app: its
	// first start writes it and a stopped engine keeps it, so launchd's
	// Installed reads it to tell "stopped" from "not installed" (R9.D3).
	return agentPlist(home, EngineLabel)
}

const RetiredAgentLabel = StationLaunchPrefix + "agent" // value: the key agent's own job a Mac without Spwn.app carries from an earlier release, which the move into the bundle boots out with the bare engine's; the engine serves the key agent itself (ADR-076).

func RetiredAgentPlist(home string) string { return agentPlist(home, RetiredAgentLabel) }

const BundledEngineLabel = WindowBundle + ".engine" // value: the engine's one job inside Spwn.app (ADR-073, mode A); it never matches EngineLabel, so no Background Task Management record a bare job left is ever attached to the job the bundle registers.

var EngineArgs = []string{"engine", "run"} // value: the arguments the engine's job runs its binary with, in both modes.

const BundledEngine = "Spwn Engine" // value: the engine's file inside Spwn.app on a Mac with the window (ADR-073, mode A), the name Activity Monitor gives its processes; the bare binary of a Mac without the app keeps StationBinary.

const BundledEnginePath = "Contents/MacOS/" + BundledEngine // value: where Spwn.app holds the engine, relative to the bundle, as launchd's BundleProgram reads it.

const BundledMenuBarPath = "Contents/Library/LoginItems/Station.app" // value: where Spwn.app holds the menu bar, its login item, relative to the bundle.

const StandaloneMenuBarPath = "Applications/Station.app" // value: where an earlier release installed the menu bar on its own, relative to the home, which the move into Spwn.app deletes.

const JobStartFile = "jobs.json" // value: the file, in ~/.local/state/station, that a bundled job of the engine reads at its start (ADR-073, mode A).

func JobStartPath(home string) string {
	return path.Join(home, ".local", "state", StateFolder, JobStartFile)
}

func BundledPlist(label string) string {
	return "Contents/Library/LaunchAgents/" + label + ".plist"
}

const (
	WindowBundle  = "com.jterrazz.spwn"
	MenuBarBundle = "com.jterrazz.station.menubar"
)

const WindowName = "Spwn" // value: Spwn.app's product name, the name macOS's Local Network alert and row give the engine it carries (ADR-073, mode A).

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
