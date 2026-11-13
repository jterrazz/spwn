package words

import (
	"cmp"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	agentscalls "github.com/jterrazz/studio-station/apps/engine/agents/calls"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	awakecalls "github.com/jterrazz/studio-station/apps/engine/awake/calls"
	backgroundcalls "github.com/jterrazz/studio-station/apps/engine/background/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
)

type Wording func(e *fail.Error) (message, hint string)

// value: how the refusal of a gesture a typed name answers names the doing
// and the done, by its action.
func typedGesture(action string) (doing, done string, ok bool) {
	switch confirm.Action(action) {
	case confirm.Stop:
		return "stopping", "stopped", true
	case confirm.Delete:
		return "deleting", "deleted", true
	default:
		return "", "", false
	}
}

// value: SPEC A5: a VM's confirmation names the live agents that stop
// first; nothing for a VM that hosts none.
func agentsFirst(e *fail.Error) string {
	if names := e.Detail["agents"]; names != "" {
		return " (its agents stop first: " + names + ")"
	}
	return ""
}

func Codes() map[fail.Code]Wording {
	// invariant: the one table of every error code's words (rule
	// every-code-worded): a code raised anywhere is worded here once.
	all := map[fail.Code]Wording{
		confirm.Needed: func(e *fail.Error) (string, string) {
			if doing, _, ok := typedGesture(e.Detail["action"]); ok && e.Detail["by"] == "yes" {
				return doing + " a " + machine.Kind(e.Detail["kind"]).Word(machine.OS(e.Detail["os"])) + " needs its name", "--confirm=" + e.Subject
			}
			return "refusing without the name typed: " + e.Subject + agentsFirst(e), "in a script: --confirm=" + e.Subject
		},
		confirm.Mismatch: func(e *fail.Error) (string, string) {
			said := "you typed \"" + e.Detail["typed"] + "\", not \"" + e.Subject + "\""
			if _, done, ok := typedGesture(e.Detail["action"]); ok {
				said = "not " + done + ": " + said
			}
			return said, "in a script: --confirm=" + e.Subject
		},
		confirm.YesNeeded: func(e *fail.Error) (string, string) {
			return "refusing without --yes: " + e.Subject + agentsFirst(e), "re-run with --yes"
		},
		confirm.Declined: func(e *fail.Error) (string, string) {
			return e.Subject + " kept: you said no", "--yes skips the question"
		},
		call.NotForThisMachine: func(e *fail.Error) (string, string) {
			if why, fix := e.Detail["why"], e.Detail["fix"]; why != "" && fix != "" {
				return why, fix
			}
			if why := e.Detail["why"]; why != "" {
				return why, "run station overview for what " + e.Subject + " has"
			}
			return e.Subject + " is a " + machine.Kind(e.Detail["kind"]).Word(machine.OS(e.Detail["os"])) + ": station does not manage its " + e.Detail["section"], "run station overview for what " + e.Subject + " has"
		},
		call.NeedMissing: func(e *fail.Error) (string, string) {
			return e.Subject + " is required by " + NeededBy(e.Detail["call"]), NeedFix(e.Subject, e.Detail["package"])
		},
		call.BadInput: func(e *fail.Error) (string, string) {
			return e.Subject + " cannot read its input", "check the request against " + e.Detail["in"]
		},
		call.BadOutput: func(e *fail.Error) (string, string) {
			return e.Subject + " answered what it cannot write", "report it: station --version"
		},
		call.Unknown: func(e *fail.Error) (string, string) {
			return "no call is named " + e.Subject, "run station --help"
		},
		call.BadFrame: func(e *fail.Error) (string, string) {
			return "the request is not a " + e.Subject + " frame", "send one JSON object per line with v 2 and a call or a watch"
		},
		call.Unreachable: func(e *fail.Error) (string, string) {
			mac := cmp.Or(e.Detail["label"], e.Subject)
			return mac + " does not answer", strings.TrimSpace("station machines " + mac)
		},
		call.VMMacUnreachable: func(e *fail.Error) (string, string) {
			mac := cmp.Or(e.Detail["label"], e.Detail["mac"])
			return e.Subject + " is on " + mac + ", which does not answer", "station machines " + mac
		},
		enginecalls.StartFailed: func(e *fail.Error) (string, string) {
			return "the engine did not start: " + e.Detail["last"], "station engine"
		},
		machinefilecalls.NotFound: func(e *fail.Error) (string, string) {
			if by := e.Detail["by"]; by != "" {
				return by + " extends " + e.Subject + ", which no source holds", "station explain --machinefile " + by
			}
			if e.Detail["path"] != "" {
				return "no Machinefile at " + e.Subject, "name its file by a path that holds it"
			}
			return "no Machinefile named " + e.Subject, "give this machine one: station init --machinefile <name>"
		},
		machinefilecalls.NameTaken: func(e *fail.Error) (string, string) {
			return e.Detail["path"] + " already holds another " + e.Detail["name"] + ": station never replaces your Machinefile", "edit that file, or remove it to copy " + e.Subject + " instead"
		},
		machinefilecalls.Busy: func(*fail.Error) (string, string) {
			return "another station run holds this machine (an apply, an upgrade or the night's upgrade): one runs at a time", "run it again once that one ends"
		},
		machinefilecalls.NeedsTerminal: func(e *fail.Error) (string, string) {
			verb := cmp.Or(e.Detail["verb"], "apply")
			return e.Subject + " " + needsWords(e.Detail["needs"]) + ": " + verb + " asks it on a terminal, and this run has none", "run station " + verb + " on a terminal"
		},
		machinefilecalls.BadPhase: func(e *fail.Error) (string, string) {
			if e.Detail["remove"] != "" {
				return "--remove takes back what apply's phase lists no longer, and the " + e.Subject + " phase carries a part of it", "station apply --remove"
			}
			return "no phase " + e.Subject + ": a Machinefile's phases are " + e.Detail["phases"], "station apply --phase image"
		},
		machinefilecalls.Invalid: func(e *fail.Error) (string, string) {
			if strings.HasSuffix(e.Subject, catalog.LockSuffix) {
				return e.Detail["problem"], "remove it: the next station apply writes it anew from what is there"
			}
			if e.Detail["rename"] != "" {
				return e.Detail["problem"], "rename it " + e.Detail["rename"] + ": station reads a Machinefile by that name alone"
			}
			return e.Detail["problem"], "fix that line: station never rewrites a Machinefile"
		},
		machinefilecalls.NewerSchema: func(e *fail.Error) (string, string) {
			return e.Subject + " is " + e.Detail["schema"] + ", newer than this station reads (" + e.Detail["reads"] + ")", "update station"
		},
		machinefilecalls.CatalogInvalid: func(e *fail.Error) (string, string) {
			return e.Detail["problem"], "fix that line: station never rewrites a package file"
		},
		machinefilecalls.CatalogNewer: func(e *fail.Error) (string, string) {
			return e.Subject + " is " + e.Detail["schema"] + ", newer than this station reads (" + e.Detail["reads"] + ")", "update station"
		},
		machinefilecalls.Mixin: func(e *fail.Error) (string, string) {
			return e.Subject + " is a mixin (it says no machine): it is extended, never applied", "extend it from a Machinefile that says its machine"
		},
		machinefilecalls.VMOnComputer: func(e *fail.Error) (string, string) {
			return e.Subject + " is a computer, and " + e.Detail["file"] + " says vm: vm is a workspace's or a server's", "remove vm, or say kind: workspace or server"
		},
		machinefilecalls.Cycle: func(e *fail.Error) (string, string) {
			return "a cycle in " + e.Detail["of"] + ": " + e.Subject, "break it: one of them must not need the other"
		},
		machinefilecalls.NoMethod: func(e *fail.Error) (string, string) {
			by := ""
			if m := e.Detail["manager"]; m != "" {
				by = " by " + m
			}
			why := e.Detail["why"]
			if at := e.Detail["at"]; at != "" {
				why += ", " + at
			}
			return e.Subject + " has no method" + by + " for " + e.Detail["system"] + " (" + why + ")", "list it as { name: " + e.Subject + ", optional: true } to skip it there"
		},
		machinefilecalls.Unpinnable: func(e *fail.Error) (string, string) {
			said := e.Subject + " " + e.Detail["version"] + ": " + e.Detail["method"] + " holds " + cmp.Or(e.Detail["holds"], "only its newest") + ", never " + e.Detail["version"] + " (" + e.Detail["at"] + ")"
			if alt := e.Detail["alternative"]; alt != "" {
				return said, alt + " can: { name: " + e.Subject + ", version: \"" + e.Detail["version"] + "\", manager: " + alt + " }"
			}
			return said, "no method of " + e.Subject + " for this system holds it: drop its version"
		},
		machinefilecalls.UnknownPackage: func(e *fail.Error) (string, string) {
			where := cmp.Or(e.Detail["at"], e.Detail["by"])
			if is := e.Detail["is"]; is != "" {
				return e.Subject + " is " + is + " (" + where + ")", "station explain"
			}
			return "the catalog has no package " + e.Subject + " (" + where + ")", "station apps"
		},
		machinefilecalls.BadValue: func(e *fail.Error) (string, string) {
			at := " (" + e.Detail["at"] + ")"
			if k := e.Detail["key"]; k != "" {
				return e.Subject + "'s " + k + ": " + e.Detail["problem"] + at, "station explain"
			}
			return e.Subject + ": " + e.Detail["problem"] + at, "station explain"
		},
		machinefilecalls.UnknownKey: func(e *fail.Error) (string, string) {
			return e.Detail["name"] + " says nothing of " + e.Subject, "station explain --machinefile " + e.Detail["name"]
		},
		machinefilecalls.Unavailable: func(e *fail.Error) (string, string) {
			return "the source " + e.Subject + " cannot be read: " + e.Detail["cause"], "check its url in ~/.config/station/preferences.json"
		},
		imagescalls.Unknown: func(e *fail.Error) (string, string) {
			return "no image \"" + e.Subject + "\"", "station images"
		},
		appscalls.Unknown: func(e *fail.Error) (string, string) {
			return "no app named \"" + e.Subject + "\"", "station apps for every app here"
		},
		appscalls.Elsewhere: func(e *fail.Error) (string, string) {
			hint := "station apps install " + e.Subject + " --on <mac>"
			switch {
			case e.Detail["vm"] != "":
				hint = "station apps install " + e.Subject + " --on " + e.Detail["vm"]
			case e.Detail["lives"] != "Macs only":
				hint = "station machines create"
			}
			return e.Subject + " lives on " + e.Detail["lives"], hint
		},
		appscalls.NotInstalled: func(e *fail.Error) (string, string) {
			return e.Subject + " is not installed", "station apps install " + e.Subject
		},
		appscalls.NoMethod: func(e *fail.Error) (string, string) {
			if e.Detail["has"] == "" {
				return e.Subject + " has no method " + e.Detail["method"] + " here", "station apps " + e.Subject
			}
			said := e.Subject + " has no method " + e.Detail["method"] + " here, only " + e.Detail["has"]
			for _, other := range strings.Split(e.Detail["has"], ", ") {
				if other != e.Detail["held"] {
					return said, "station apps move " + e.Subject + " --to " + other
				}
			}
			return said, "station apps " + e.Subject
		},
		appscalls.Unpinnable: func(e *fail.Error) (string, string) {
			return e.Subject + " " + e.Detail["version"] + ": " + e.Detail["method"] + " holds " + cmp.Or(e.Detail["holds"], "only its newest") + ", never " + e.Detail["version"], "ask " + e.Subject + " with no version, or by a method that holds it"
		},
		accountscalls.Unknown: func(e *fail.Error) (string, string) {
			return "no account named \"" + e.Subject + "\"", "station accounts"
		},
		accountscalls.NotShareable: func(e *fail.Error) (string, string) {
			return e.Subject + " is not on the share list", "the share list is share_list in ~/.config/station/preferences.json"
		},
		accountscalls.NotSignedIn: func(e *fail.Error) (string, string) {
			hint := "station accounts"
			if e.Subject == catalog.ProviderClaude {
				hint = "station settings apply claude-token"
			}
			return accounts.Label(e.Subject) + " is not signed in on " + e.Detail["mac"], hint
		},
		accountscalls.NotShared: func(e *fail.Error) (string, string) {
			return accounts.Label(e.Subject) + " is not shared with " + e.Detail["vm"], "station accounts"
		},
		accountscalls.Unreadable: func(e *fail.Error) (string, string) {
			if e.Detail["log"] == "" {
				return accounts.Label(e.Subject) + " could not be read on " + e.Detail["mac"], "station accounts"
			}
			return accounts.Label(e.Subject) + " could not be read on " + e.Detail["mac"], "why is kept in " + e.Detail["log"]
		},
		accountscalls.VMStopped: func(e *fail.Error) (string, string) {
			return e.Subject + " is stopped", "station machines start " + e.Subject
		},
		accountscalls.UnknownVM: func(e *fail.Error) (string, string) {
			return "no machine named \"" + e.Subject + "\"", "station machines"
		},
		awakecalls.OutOfBounds: func(e *fail.Error) (string, string) {
			return "\"" + e.Subject + "\" is not between 1 min and 24 h", "station awake 4h"
		},
		awakecalls.NoLid: func(e *fail.Error) (string, string) {
			mac := cmp.Or(e.Detail["label"], e.Subject)
			return mac + " has no lid: --lid is for laptops", "station awake " + e.Detail["for"] + " --on " + mac
		},
		awakecalls.NoBattery: func(e *fail.Error) (string, string) {
			mac := cmp.Or(e.Detail["label"], e.Subject)
			return mac + " has no battery: --battery is for laptops", "station awake " + e.Detail["for"] + " --on " + mac
		},
		accountscalls.OtherMac: func(e *fail.Error) (string, string) {
			mac := cmp.Or(e.Detail["label"], e.Detail["mac"])
			return e.Subject + " is on " + mac + ": share from " + mac + "'s accounts", "station accounts share " + e.Detail["account"] + " --with " + e.Subject + " --on " + mac
		},
		accountscalls.TermOutOfBounds: func(e *fail.Error) (string, string) {
			return "a share lasts 1 min to 7 d, not " + e.Subject, "station accounts share " + e.Detail["account"] + " --with " + e.Detail["vm"] + " --for 12h"
		},
		accountscalls.ToServer: func(e *fail.Error) (string, string) {
			return e.Subject + " is a server: station shares no account with it", "give it its secrets with your own tooling"
		},
		machinescalls.NotFinished: func(e *fail.Error) (string, string) {
			return e.Subject + " stays: only an ephemeral workspace that is stopped, or whose agent finished, is deleted by a cleanup", "station machines cleanup"
		},
		agentscalls.UnknownAgent: func(e *fail.Error) (string, string) {
			return "no agent named \"" + e.Subject + "\"", "station agents"
		},
		agentscalls.Ended: func(e *fail.Error) (string, string) {
			said := e.Subject + " has finished"
			if r := e.Detail["reason"]; r != "" {
				said += ": " + r
			}
			return said, "station agents forget " + e.Subject
		},
		agentscalls.NoPR: func(e *fail.Error) (string, string) {
			return e.Subject + " has opened no PR", "station agents"
		},
		agentscalls.NoText: func(e *fail.Error) (string, string) {
			return "nothing to send to " + e.Subject, "station agents reply " + e.Subject + " <text>"
		},
		agentscalls.NoTerminal: func(e *fail.Error) (string, string) {
			return e.Subject + " runs in no terminal station opened", "station agents"
		},
		agentscalls.UnknownApp: func(e *fail.Error) (string, string) {
			return "no agent app \"" + e.Subject + "\": claudecode or codex", "station agents start --app claudecode"
		},
		agentscalls.NameTaken: func(e *fail.Error) (string, string) {
			if ws := e.Detail["workspace"]; ws != "" {
				return e.Subject + " is " + ws + "'s: a workspace names its own agents " + ws + " and " + e.Detail["nth"], "name it another way: --name <name>"
			}
			return "an agent is already named " + e.Subject, "name it another way: --name <name>"
		},
		agentscalls.BadName: func(e *fail.Error) (string, string) {
			return "\"" + e.Subject + "\" is no agent name: lowercase letters, digits and -", "--name review-bot"
		},
		agentscalls.StillLive: func(e *fail.Error) (string, string) {
			return e.Subject + " is " + e.Detail["state"] + ": only a finished agent is forgotten", "station agents stop " + e.Subject
		},
		agentscalls.NoWhere: func(e *fail.Error) (string, string) {
			switch e.Detail["both"] {
			case "inside":
				return "an agent started inside a workspace runs on it: --new makes an ephemeral workspace from a Mac", "station agents start \"<task>\""
			case "both":
				return "an agent starts on a workspace (--on) or on a new ephemeral workspace (--new), not both", "station agents start \"<task>\" --new"
			}
			return "an agent starts on a running workspace (--on <workspace>) or on a new ephemeral workspace (--new)", "station agents start \"<task>\" --on <workspace>"
		},
		agentscalls.Ambiguous: func(e *fail.Error) (string, string) {
			on := strings.Split(e.Detail["on"], ", ")
			named := make([]string, len(on))
			for i, vm := range on {
				named[i] = e.Subject + " on " + vm
			}
			return "more than one agent is named " + e.Subject + ": " + strings.Join(named, " and "), "name its workspace: --on " + strings.Join(on, " or --on ")
		},
		agentscalls.NotStopped: func(e *fail.Error) (string, string) {
			return e.Subject + " did not stop: " + e.Detail["cause"], "station agents stop " + e.Subject
		},
		backgroundcalls.Unknown: func(e *fail.Error) (string, string) {
			return "no background task \"" + e.Subject + "\"", "station background"
		},
		backgroundcalls.RootScope: func(e *fail.Error) (string, string) {
			return e.Subject + " runs as root: station never runs sudo", e.Detail["command"]
		},
		backgroundcalls.IsTheEngine: func(*fail.Error) (string, string) {
			return "that is station's engine", "station engine stop"
		},
		settingscalls.Unknown: func(e *fail.Error) (string, string) {
			return "no setting named \"" + e.Subject + "\"", "station settings"
		},
		settingscalls.IsChecked: func(e *fail.Error) (string, string) {
			return e.Subject + " is checked, not set: station never runs sudo", e.Detail["fix"]
		},
		settingscalls.NotForThisMachine: func(e *fail.Error) (string, string) {
			if e.Detail["why"] == "Mac only" {
				return e.Subject + " is for Macs only", "station settings apply " + e.Subject + " --on this-mac"
			}
			return e.Subject + " is for " + strings.TrimSuffix(e.Detail["why"], " only") + " only", "station settings"
		},
		settingscalls.NeedsAdmin: func(e *fail.Error) (string, string) {
			line := "sudo station settings " + cmp.Or(e.Detail["verb"], "apply") + " " + e.Subject
			if mac := e.Detail["label"]; mac != "" {
				line = "ssh -t " + mac + " " + line
			}
			return e.Subject + " needs admin rights", line
		},
		settingscalls.NotApplied: func(e *fail.Error) (string, string) {
			return e.Subject + " is not applied here", "station settings apply " + e.Subject
		},
		settingscalls.NeedsTerminal: func(e *fail.Error) (string, string) {
			return e.Subject + " asks a value: run it on a terminal", "station settings apply " + e.Subject
		},
		diskcalls.Unknown: func(e *fail.Error) (string, string) {
			return "no disk row for \"" + e.Subject + "\"", "station disk"
		},
		diskcalls.Elsewhere: func(e *fail.Error) (string, string) {
			return e.Detail["why"], "station disk"
		},
		diskcalls.NothingToFree: func(e *fail.Error) (string, string) {
			return e.Subject + " keeps nothing station can rebuild", "station disk"
		},
		diskcalls.NeverTouched: func(e *fail.Error) (string, string) {
			return e.Detail["path"] + " is never touched: station erases no key, token or setting", "station disk"
		},
		diskcalls.NothingToErase: func(e *fail.Error) (string, string) {
			return e.Subject + " has no data station can erase", cmp.Or(e.Detail["hint"], "station disk")
		},
		diskcalls.AppRunning: func(e *fail.Error) (string, string) {
			process, subject := e.Detail["process"], e.Detail["subject"]
			if e.Detail["verb"] == "move" {
				return process + " is running: " + subject + " cannot move now", "quit " + process + ", then station disk move " + subject
			}
			return process + " is running: its data cannot be erased now", "quit " + process + ", then station disk erase " + subject
		},
		diskcalls.NotMisplaced: func(e *fail.Error) (string, string) {
			return e.Subject + " is not misplaced", "station disk"
		},
		ports.CodeRangeFull: func(e *fail.Error) (string, string) {
			return e.Subject + " has " + e.Detail["max"] + " shared ports, the most a VM may", "station ports stop-sharing " + e.Subject + " <port>"
		},
		ports.CodeNotFound: func(e *fail.Error) (string, string) {
			return "nothing listens on " + e.Subject + " and nothing is reserved", "station ports --on " + e.Detail["vm"]
		},
		ports.CodeSignedOut: func(*fail.Error) (string, string) {
			return catalog.SignedOutLocalOnly, "open " + catalog.TailnetProduct + " and sign in"
		},
		ports.CodeNeedsVM: func(e *fail.Error) (string, string) {
			return "say which VM's port " + e.Subject + " it is", "station ports share <vm> " + e.Subject
		},
		ports.CodeNeedsRoute: func(e *fail.Error) (string, string) {
			return "say how " + e.Subject + " is reached from elsewhere", "add --via, one of " + e.Detail["routes"]
		},
		ports.CodeUnknownRoute: func(e *fail.Error) (string, string) {
			return `"` + e.Subject + `" is not a route station knows`, "--via is one of " + e.Detail["routes"]
		},
		ports.CodeNotDeclared: func(e *fail.Error) (string, string) {
			return e.Subject + " is not declared public", "station ports --on " + e.Detail["machine"]
		},
		ports.CodeElsewhere: func(e *fail.Error) (string, string) {
			return e.Subject + " runs on " + e.Detail["mac"], "add --on " + e.Detail["mac"]
		},
		call.NoStation: func(e *fail.Error) (string, string) {
			return "station is not installed on " + cmp.Or(e.Detail["label"], e.Subject), "install it there (scripts/install.sh), then add it again"
		},
		call.FarTooOld: func(e *fail.Error) (string, string) {
			far := cmp.Or(e.Detail["label"], e.Subject)
			repair := "update station on " + far + " (scripts/install.sh there)"
			if image := e.Detail["image"]; image != "" {
				repair = "station images build " + image + ", then make " + far + " again from it: station machines delete " + far + " --confirm=" + far + ", then station machines create " + far + " --image " + image
			}
			if floor := e.Detail["floor"]; floor != "" {
				return far + " runs station " + e.Detail["version"] + ": this station speaks to " + floor + " and later", repair
			}
			return far + " runs station " + e.Detail["version"] + ", which has no " + e.Detail["call"], repair
		},
		call.CallerTooOld: func(e *fail.Error) (string, string) {
			return "station " + e.Detail["version"] + " is older than " + e.Detail["floor"] + ", the first release the far station speaks to", "update station here (scripts/install.sh)"
		},
	}
	for code, word := range usage() {
		all[code] = word
	}
	return all
}

// value: what a plan's needs ask, as the refusal off a terminal says them.
func needsWords(needs string) string {
	var said []string
	for _, n := range strings.Split(needs, ", ") {
		switch n {
		case "sudo":
			said = append(said, "asks your password")
		case "terminal":
			said = append(said, "asks you questions")
		case "license":
			said = append(said, "asks you to accept a license")
		case "app-store-login":
			said = append(said, "needs the App Store signed in")
		}
	}
	return strings.Join(said, " and ")
}
