package words

import (
	"cmp"
	"strings"

	"github.com/jterrazz/studio-station/src/accounts"
	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	appscalls "github.com/jterrazz/studio-station/src/apps/calls"
	awakecalls "github.com/jterrazz/studio-station/src/awake/calls"
	backgroundcalls "github.com/jterrazz/studio-station/src/background/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	diskcalls "github.com/jterrazz/studio-station/src/disk/calls"
	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
	imagescalls "github.com/jterrazz/studio-station/src/images/calls"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/ports"
	settingscalls "github.com/jterrazz/studio-station/src/settings/calls"
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
				return doing + " a " + e.Detail["kind"] + " needs its name", "--confirm=" + e.Subject
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
			return e.Subject + " is a " + e.Detail["kind"] + ": station does not manage its " + e.Detail["section"], "run station overview for what " + e.Subject + " has"
		},
		call.NeedMissing: func(e *fail.Error) (string, string) {
			return e.Subject + " is required by " + NeededBy(e.Detail["call"]), NeedFix(e.Subject, e.Detail["item"])
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
		accountscalls.ToServer: func(e *fail.Error) (string, string) {
			return e.Subject + " is a server: station shares no account with it", "give it its secrets with your own tooling"
		},
		machinescalls.NotFinished: func(e *fail.Error) (string, string) {
			return e.Subject + " stays: only a throwaway that is stopped, or whose agent finished, is deleted by a cleanup", "station machines cleanup"
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
				return "an agent started inside a workspace runs on it: --new makes a throwaway from a Mac", "station agents start \"<task>\""
			case "both":
				return "an agent starts on a workspace (--on) or on a new throwaway (--new), not both", "station agents start \"<task>\" --new"
			}
			return "an agent starts on a running workspace (--on <workspace>) or on a new throwaway (--new)", "station agents start \"<task>\" --on <workspace>"
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
			mac := cmp.Or(e.Detail["label"], e.Subject)
			return mac + " runs station " + e.Detail["version"] + ", which has no " + e.Detail["call"], "update station on " + mac + " (scripts/install.sh there)"
		},
	}
	for code, word := range usage() {
		all[code] = word
	}
	return all
}
