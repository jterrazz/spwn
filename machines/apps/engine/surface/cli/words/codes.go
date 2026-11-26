package words

import (
	"cmp"
	"errors"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
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
	"github.com/jterrazz/studio-station/apps/engine/images"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"
	journalcalls "github.com/jterrazz/studio-station/apps/engine/journal/calls"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	projectcalls "github.com/jterrazz/studio-station/apps/engine/project/calls"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	skillscalls "github.com/jterrazz/studio-station/apps/engine/skills/calls"
)

type Wording func(e *fail.Error) (message, hint string)

const linuxBinaryHint = "download station-linux-arm64 from https://github.com/" + catalog.StationRepo + "/releases, then station images build base-ubuntu --binary <that file>"

// value: the way out of a file a newer station wrote: its schema, or a key
// this station does not know.
const updateStation = "update station"

func newerWrote(e *fail.Error) string {
	if e.Detail["unknown"] == "" {
		return ""
	}
	return "; if a newer station wrote " + e.Detail["unknown"] + ", " + updateStation
}

// value: how the refusal of a gesture a typed name answers names the doing
// and the done, by its action.
func typedGesture(action string) (doing, done string, ok bool) {
	switch confirm.Action(action) {
	case confirm.Stop:
		return "stopping", "stopped", true
	case confirm.Delete, confirm.DeleteAgent, confirm.DeleteSkill:
		return "deleting", "deleted", true
	case confirm.RestartServer:
		return "restarting", "restarted", true
	default:
		return "", "", false
	}
}

// value: a held VM's confirmation says who holds it, in the sentence the
// engine worded.
func heldBy(e *fail.Error) string {
	if said := e.Detail["holder"]; said != "" {
		return ". " + said
	}
	return ""
}

func carried(e *fail.Error, generic, advice string) (message, hint string) {
	// value: the engine words its own refusals where it raises them, naming
	// the VM, the port or the duration at hand; a caller prints those
	// words, and its own only where the refusal carries none.
	var said *fail.Said
	if errors.As(e.Cause, &said) && said.Message != "" {
		return said.Message, cmp.Or(said.Hint, advice)
	}
	return generic, advice
}

func programLabel(e *fail.Error) string { return accounts.Program(e.Subject).Label() }

func Codes() map[fail.Code]Wording {
	// invariant: the one table of every error code's words (rule
	// every-code-worded): a code raised anywhere is worded here once.
	all := map[fail.Code]Wording{
		confirm.Needed: func(e *fail.Error) (string, string) {
			if doing, _, ok := typedGesture(e.Detail["action"]); ok && e.Detail["by"] == "yes" {
				return doing + " a " + machine.Kind(e.Detail["kind"]).Word(machine.OS(e.Detail["os"])) + " needs its name", "--confirm=" + e.Subject
			}
			return "refusing without the name typed: " + e.Subject + heldBy(e), "in a script: --confirm=" + e.Subject
		},
		confirm.Mismatch: func(e *fail.Error) (string, string) {
			said := "you typed \"" + e.Detail["typed"] + "\", not \"" + e.Subject + "\""
			if _, done, ok := typedGesture(e.Detail["action"]); ok {
				said = "not " + done + ": " + said
			}
			return said, "in a script: --confirm=" + e.Subject
		},
		confirm.YesNeeded: func(e *fail.Error) (string, string) {
			return "refusing without --yes: " + e.Subject, "re-run with --yes"
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
			return e.Subject + " is a " + machine.Kind(e.Detail["kind"]).Word(machine.OS(e.Detail["os"])) + ": Machines does not manage its " + e.Detail["section"], "run station overview for what " + e.Subject + " has"
		},
		call.NeedMissing: func(e *fail.Error) (string, string) {
			return e.Subject + " is required by " + NeededBy(e.Detail["call"]), NeedFix(e.Subject, e.Detail["package"])
		},
		call.BadInput: func(e *fail.Error) (string, string) {
			return e.Subject + " cannot read its input", "check the request against " + e.Detail["in"]
		},
		call.BadOutput: func(e *fail.Error) (string, string) {
			return e.Subject + " answered what it cannot write", "report it: Machines --version"
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
		enginecalls.ReleaseUnreachable: func(e *fail.Error) (string, string) {
			return "the latest release was not read: " + e.Detail["cause"], "station engine"
		},
		enginecalls.StateNoKey: func(*fail.Error) (string, string) {
			return "the state was asked with no key: name a key, <family>:* or local", "station engine"
		},
		enginecalls.StateUnknownKey: func(e *fail.Error) (string, string) {
			return e.Subject + " is no key of the engine's state: name a key, <family>:* or local", "station engine"
		},
		enginecalls.VMStopped: func(e *fail.Error) (string, string) {
			return carried(e, "the VM is stopped", "station machines")
		},
		enginecalls.UnknownVM: func(e *fail.Error) (string, string) {
			return carried(e, "the engine knows no VM of that name", "station machines")
		},
		enginecalls.UnknownOp: func(e *fail.Error) (string, string) {
			return carried(e, "the engine knows no such operation", "station machines")
		},
		enginecalls.UnknownPage: func(e *fail.Error) (string, string) {
			return carried(e, "the engine serves no such page", "station overview")
		},
		enginecalls.GuestAbsent: func(e *fail.Error) (string, string) {
			return carried(e, "the VM's guest is not connected", "station machines")
		},
		enginecalls.GuestTimeout: func(e *fail.Error) (string, string) {
			return carried(e, "the VM's guest did not answer in time", "station machines")
		},
		enginecalls.LocalNetwork: func(e *fail.Error) (string, string) {
			return carried(e, "the Mac has not approved Machines for Local Network", "station machines")
		},
		enginecalls.IgnoredPort: func(e *fail.Error) (string, string) {
			return carried(e, "Machines never shares port 22 nor a port below 1024", "station ports")
		},
		enginecalls.NotAllowed: func(e *fail.Error) (string, string) {
			return carried(e, "the port is outside the rights of its VM", "station ports")
		},
		enginecalls.RangeFull: func(e *fail.Error) (string, string) {
			return carried(e, "no port of the range is free", "station ports")
		},
		enginecalls.AwakeRange: func(e *fail.Error) (string, string) {
			return carried(e, "the awake duration is out of its range", "station awake 4h")
		},
		enginecalls.AwakeGuard: func(e *fail.Error) (string, string) {
			return carried(e, "a guard holds the awake session back", "station awake")
		},
		enginecalls.AwakeUnsupported: func(e *fail.Error) (string, string) {
			return carried(e, "this Mac cannot be kept awake", "station awake")
		},
		enginecalls.Internal: func(e *fail.Error) (string, string) {
			return carried(e, "the engine failed on its own", "station overview")
		},
		machinefilecalls.NotFound: func(e *fail.Error) (string, string) {
			if by := e.Detail["by"]; by != "" {
				return by + " extends " + e.Subject + ", which no source holds", "station machinefile explain --machinefile " + by
			}
			if e.Detail["path"] != "" {
				return "no Machinefile at " + e.Subject, "name its file by a path that holds it"
			}
			return "no Machinefile named " + e.Subject, "give this machine one: station init --machinefile <name>"
		},
		machinefilecalls.NameTaken: func(e *fail.Error) (string, string) {
			return e.Detail["path"] + " already holds another " + e.Detail["name"] + ": Machines never replaces your Machinefile", "edit that file, or remove it to copy " + e.Subject + " instead"
		},
		machinefilecalls.Busy: func(*fail.Error) (string, string) {
			return "another station run holds this machine (an apply, an upgrade or the night's upgrade): one runs at a time", "run it again once that one ends"
		},
		machinefilecalls.NeedsTerminal: func(e *fail.Error) (string, string) {
			verb := cmp.Or(e.Detail["verb"], "apply")
			return e.Subject + " " + needsWords(e.Detail["needs"]) + ": " + verb + " asks it on a terminal, and this run has none", "run station machinefile " + verb + " on a terminal"
		},
		machinefilecalls.BadPhase: func(e *fail.Error) (string, string) {
			if e.Detail["remove"] != "" {
				return "--remove takes back what apply's phase lists no longer, and the " + e.Subject + " phase carries a part of it", "station machinefile apply --remove"
			}
			return "no phase " + e.Subject + ": a Machinefile's phases are " + e.Detail["phases"], "station machinefile apply --phase image"
		},
		machinefilecalls.Invalid: func(e *fail.Error) (string, string) {
			if strings.HasSuffix(e.Subject, catalog.LockSuffix) {
				return e.Detail["problem"], "remove it: the next station machinefile apply writes it anew from what is there"
			}
			if e.Detail["rename"] != "" {
				return e.Detail["problem"], "rename it " + e.Detail["rename"] + ": Machines reads a Machinefile by that name alone"
			}
			return e.Detail["problem"], "fix that line: Machines never rewrites a Machinefile" + newerWrote(e)
		},
		machinefilecalls.NewerSchema: func(e *fail.Error) (string, string) {
			return e.Subject + " is " + e.Detail["schema"] + ", newer than this station reads (" + e.Detail["reads"] + ")", updateStation
		},
		machinefilecalls.CatalogInvalid: func(e *fail.Error) (string, string) {
			return e.Detail["problem"], "fix that line: Machines never rewrites a package file" + newerWrote(e)
		},
		machinefilecalls.CatalogNewer: func(e *fail.Error) (string, string) {
			return e.Subject + " is " + e.Detail["schema"] + ", newer than this station reads (" + e.Detail["reads"] + ")", updateStation
		},
		machinefilecalls.NotATemplate: func(e *fail.Error) (string, string) {
			return (&machinefile.NotTemplateError{Name: e.Subject}).Error(), "station machinefile explain --machinefile " + e.Subject
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
				return e.Subject + " is " + is + " (" + where + ")", "station machinefile explain"
			}
			return "the catalog has no package " + e.Subject + " (" + where + ")", "station apps"
		},
		machinefilecalls.BadValue: func(e *fail.Error) (string, string) {
			at := " (" + e.Detail["at"] + ")"
			if k := e.Detail["key"]; k != "" {
				return e.Subject + "'s " + k + ": " + e.Detail["problem"] + at, "station machinefile explain"
			}
			return e.Subject + ": " + e.Detail["problem"] + at, "station machinefile explain"
		},
		machinefilecalls.UnknownKey: func(e *fail.Error) (string, string) {
			return e.Detail["name"] + " says nothing of " + e.Subject, "station machinefile explain --machinefile " + e.Detail["name"]
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
		accounts.LoginUnknownProgram: func(e *fail.Error) (string, string) {
			return "no program named \"" + e.Subject + "\"", "Machines signs in gh, npm, claude and wrangler"
		},
		accounts.LoginNotInstalled: func(e *fail.Error) (string, string) {
			return programLabel(e) + " is not installed here", "install it first: station apps"
		},
		accounts.LoginAlreadySignedIn: func(e *fail.Error) (string, string) {
			return programLabel(e) + " is already signed in here", "sign out first to sign in again"
		},
		accounts.LoginMachineStopped: func(e *fail.Error) (string, string) {
			return programLabel(e) + " cannot sign in: its machine is stopped", "start the machine, then sign in"
		},
		accounts.LoginBusy: func(e *fail.Error) (string, string) {
			return programLabel(e) + " is already signing in", "finish that sign-in or wait for it to expire"
		},
		accounts.LoginSiteRefused: func(e *fail.Error) (string, string) {
			return programLabel(e) + " asked to open a site Machines does not know", "nothing was opened"
		},
		accounts.LoginExpired: func(e *fail.Error) (string, string) {
			return "the sign-in expired before " + programLabel(e) + " was approved", "start the sign-in again"
		},
		accounts.LoginFailed: func(e *fail.Error) (string, string) {
			return programLabel(e) + " could not sign in", "start the sign-in again"
		},
		accounts.LoginNotSignedIn: func(e *fail.Error) (string, string) {
			return programLabel(e) + " is not signed in here", "there is nothing to sign out of"
		},
		accounts.LoginGranted: func(e *fail.Error) (string, string) {
			return programLabel(e) + " is granted here, not signed in here", "remove the grant instead"
		},
		accounts.LoginNoExchange: func(e *fail.Error) (string, string) {
			return "no sign-in " + e.Subject + " is in progress", "start the sign-in again"
		},
		accounts.LoginOtherMac: func(e *fail.Error) (string, string) {
			return "this sign-in runs on " + e.Subject + ", which another Mac holds", "open it from " + e.Subject
		},
		accounts.LoginNotAsking: func(e *fail.Error) (string, string) {
			return "sign-in " + e.Subject + " waits for no code", "finish it in the browser"
		},
		accounts.LoginCodeRefused: func(*fail.Error) (string, string) {
			return "the code was refused", "copy it again from the page"
		},
		accounts.VaultLabelTaken: func(e *fail.Error) (string, string) {
			return "a password is already labeled \"" + e.Subject + "\"", "label it another way"
		},
		accounts.VaultEmptyLabel: func(*fail.Error) (string, string) {
			return "a password needs a label", "name it so you can tell it apart"
		},
		accounts.VaultEmptyValue: func(*fail.Error) (string, string) {
			return "a typed password cannot be empty", "type it, or add it from a sign-in"
		},
		accounts.VaultNotSignedIn: func(e *fail.Error) (string, string) {
			return programLabel(e) + " is not signed in on this Mac: there is nothing to add", "sign in first"
		},
		accounts.VaultCannotCapture: func(e *fail.Error) (string, string) {
			return "Machines cannot take " + programLabel(e) + "'s sign-in into the vault", "type its token instead"
		},
		accounts.VaultUnknown: func(e *fail.Error) (string, string) {
			return "no password " + e.Subject + " in this Mac's vault", "read the vault for what it keeps"
		},
		accounts.GrantUnknownEntry: func(e *fail.Error) (string, string) {
			return "no password " + e.Subject + " in this Mac's vault", "read the vault for what it keeps"
		},
		accounts.GrantUnknownMachine: func(e *fail.Error) (string, string) {
			return "no machine named \"" + e.Subject + "\"", "station machines"
		},
		accounts.GrantToServer: func(e *fail.Error) (string, string) {
			return e.Subject + " is a server: a grant never reaches it", "give it its secrets with your own tooling"
		},
		accounts.GrantOtherMac: func(e *fail.Error) (string, string) {
			return e.Subject + " is on " + e.Detail["mac"] + ": a grant is made from that Mac's vault", "do it from " + e.Detail["mac"]
		},
		accounts.GrantMachineStopped: func(e *fail.Error) (string, string) {
			return e.Subject + " is stopped: a grant has nowhere to land", "start it first"
		},
		accounts.GrantSignedInHere: func(e *fail.Error) (string, string) {
			return "that program is already signed in on " + e.Subject, "sign it out there first"
		},
		accounts.GrantPresenceRefused: func(*fail.Error) (string, string) {
			return "this Mac did not confirm you are there: nothing was granted", "grant again and confirm on this Mac"
		},
		accounts.GrantNoPresence: func(*fail.Error) (string, string) {
			return "this Mac cannot ask you to confirm: nothing was granted", "unlock this Mac's screen and grant again"
		},
		accounts.GrantCannotDeliver: func(e *fail.Error) (string, string) {
			return "the grant could not reach " + e.Subject + ": nothing was granted", "check that " + e.Subject + " answers, then grant again"
		},
		accounts.GrantNotGranted: func(e *fail.Error) (string, string) {
			return "that password is not granted to " + e.Subject, "read the vault for its grants"
		},
		accounts.GrantNoPassword: func(e *fail.Error) (string, string) {
			return "no password for " + e.Subject + " in this Mac's vault", "add one to the vault, then grant it"
		},
		accounts.GrantManyPasswords: func(e *fail.Error) (string, string) {
			return "more than one password for " + e.Subject + " in this Mac's vault: grant one from the window", "read the vault for what it keeps"
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
		machinescalls.NotEnoughMemory: func(e *fail.Error) (string, string) {
			said := "not enough memory on " + e.Subject + ": " + e.Detail["asked_gb"] + " GB asked, " + e.Detail["left_gb"] + " GB left for VMs"
			if left, _ := strconv.Atoi(e.Detail["left_gb"]); left < 1 {
				return said, "stop a VM first: station machines"
			}
			return said, "ask for less: --memory " + e.Detail["left_gb"] + "G"
		},
		machinescalls.NoImage: func(e *fail.Error) (string, string) {
			if e.Detail["fetched"] == "yes" {
				return "image " + e.Subject + " is not on " + e.Detail["mac"], "station images update " + e.Subject
			}
			return "image " + e.Subject + " is not built on " + e.Detail["mac"], "station images build " + e.Subject
		},
		machinescalls.EngineDown: func(e *fail.Error) (string, string) {
			if e.Detail["installed"] == "yes" {
				return "the engine is stopped on " + e.Subject, "station engine start"
			}
			return "the engine is not installed on " + e.Subject, "station engine start"
		},
		machinescalls.BadLifetime: func(e *fail.Error) (string, string) {
			return (&machines.BadLifetimeError{Value: e.Subject}).Error(), "station machines create --persistent"
		},
		machinescalls.ServerNeedsName: func(*fail.Error) (string, string) {
			return machinescalls.ServerNeedsNameRule.Why, machinescalls.ServerNeedsNameRule.Fix
		},
		machinescalls.CheckForServers: func(*fail.Error) (string, string) {
			return machinescalls.CheckForServersRule.Why, machinescalls.CheckForServersRule.Fix
		},
		machinescalls.ServerBridged: func(*fail.Error) (string, string) {
			return machinescalls.ServerBridgedRule.Why, machinescalls.ServerBridgedRule.Fix
		},
		machinescalls.BadCheck: func(e *fail.Error) (string, string) {
			return (&machines.BadCheckError{Value: e.Subject}).Error(), "--check https://<name>:6443/readyz, or --check <name>:6443"
		},
		machinescalls.BadCreator: func(e *fail.Error) (string, string) {
			return (&machines.BadCreatorError{Session: e.Subject}).Error(), "name the session in lowercase letters, digits and \"-\", 1 to 40 characters, never \"" + machines.ByOwner + "\""
		},
		machinescalls.UnknownImage: func(e *fail.Error) (string, string) {
			return (&machines.UnknownImageError{Image: e.Subject}).Error(), "station images for the images"
		},
		machinescalls.TemplateName: func(e *fail.Error) (string, string) {
			return (&machines.TemplateNameError{Name: e.Subject, Template: e.Detail["template"]}).Error(), "name it apart: station machines create <name> --from " + e.Detail["template"]
		},
		machinescalls.ServerTemplate: func(e *fail.Error) (string, string) {
			return (&machines.ServerTemplateError{Name: e.Subject}).Error(), "station machines create <name> --server"
		},
		machinescalls.BadNetwork: func(e *fail.Error) (string, string) {
			bridge := e.Detail["bridge"] == "yes"
			said := (&machines.BadNetworkError{Value: e.Subject, Bridge: bridge}).Error()
			if bridge {
				return said, "share its ports instead: station ports share"
			}
			return said, "station machines create --network nat"
		},
		machinescalls.NoHypervisor: func(e *fail.Error) (string, string) {
			return (&machines.NoEngineError{Hypervisor: e.Subject}).Error(), "station apps install " + e.Subject
		},
		imagescalls.NoLinuxBinary: func(*fail.Error) (string, string) {
			return (&images.NoLinuxBinaryError{}).Error(), linuxBinaryHint
		},
		imagescalls.BinaryNotFound: func(e *fail.Error) (string, string) {
			return (&images.BinaryNotFoundError{Path: e.Subject}).Error(), linuxBinaryHint
		},
		imagescalls.NoRecipe: func(e *fail.Error) (string, string) {
			return (&images.NoRecipeError{Image: e.Subject}).Error(), "station images update " + e.Subject
		},
		imagescalls.OldRecord: func(e *fail.Error) (string, string) {
			return (&images.OldRecordError{File: e.Detail["file"]}).Error(), "station images delete " + e.Subject + "@" + e.Detail["version"]
		},
		imagescalls.InUse: func(e *fail.Error) (string, string) {
			said := images.InUseError{Image: e.Subject, Version: e.Detail["version"], Building: e.Detail["building"] == "true", Newest: e.Detail["newest"] == "true"}
			if vms := e.Detail["vms"]; vms != "" {
				said.VMs = strings.Split(vms, ",")
			}
			return said.Error(), "station images lists the versions you can delete"
		},
		imagescalls.NoHypervisor: func(e *fail.Error) (string, string) {
			return (&images.NoEngineError{Hypervisor: e.Subject}).Error(), "station apps install " + e.Subject
		},
		machinescalls.TooLittle: func(*fail.Error) (string, string) {
			return (&machines.TooLittleMemoryError{}).Error(), "ask for " + strconv.Itoa(machines.MinMemoryBytes>>30) + " GB or more: --memory " + strconv.Itoa(machines.MinMemoryBytes>>30) + "G"
		},
		machinescalls.BadPurpose: func(e *fail.Error) (string, string) {
			if e.Detail["secret"] == "yes" {
				return (&machines.BadPurposeError{Secret: true}).Error(), "keep the secret in the vault and say what the machine is for"
			}
			return (&machines.BadPurposeError{Purpose: e.Subject}).Error(), "say what it is for in one shorter line"
		},
		machinescalls.BeingCreated: func(e *fail.Error) (string, string) {
			return (&machines.BeingCreatedError{Name: e.Subject, Mac: e.Detail["mac"]}).Error(), "wait for its create to end, or interrupt it where it runs"
		},
		machinescalls.BadName: func(e *fail.Error) (string, string) {
			return "\"" + e.Subject + "\" is not a valid name", "use lowercase letters, digits and \"-\", 1 to 40 characters, never \"-\" first"
		},
		machinescalls.NameTaken: func(e *fail.Error) (string, string) {
			name, on := "\""+e.Subject+"\"", e.Detail["on"]
			switch e.Detail["by"] {
			case "mac":
				return name + " already names a Mac", "pick another name"
			case "vm":
				return name + " is taken (on " + on + ")", "pick another name"
			}
			return "machine " + name + " already exists", "station machines forget " + e.Subject + " first, to change it"
		},
		machinescalls.NotFinished: func(e *fail.Error) (string, string) {
			return e.Subject + " stays: only a stopped ephemeral workspace is deleted by a cleanup", "station machines cleanup"
		},
		agentscalls.NotFound: func(e *fail.Error) (string, string) {
			return "no agent named \"" + e.Subject + "\"", "station agents"
		},
		agentscalls.Exists: func(e *fail.Error) (string, string) {
			return "an agent named \"" + e.Subject + "\" exists", "pick another name"
		},
		agentscalls.BadName: func(e *fail.Error) (string, string) {
			if e.Subject == "" {
				return "the new agent needs a name", "--as <name>"
			}
			return "\"" + e.Subject + "\" is not a name an agent takes", "lowercase letters, digits and dashes, a letter first, and no verb of agents"
		},
		agentscalls.UnknownHarness: func(e *fail.Error) (string, string) {
			return "\"" + e.Subject + "\" is not a harness", "--harness " + strings.Join(catalog.Harnesses(), " or ")
		},
		agentscalls.NothingAsked: func(e *fail.Error) (string, string) {
			return "nothing to change on " + e.Subject, "--team <team> or --harness <harness>"
		},
		agentscalls.NotShared: func(e *fail.Error) (string, string) {
			said := "the project at " + e.Detail["path"] + " is not shared with " + e.Subject
			if e.Detail["vm"] == "" {
				return said, "run it on " + e.Subject + ", from a project of its own"
			}
			return said, "add it to " + e.Subject + "'s Machinefile vm.mounts"
		},
		agentscalls.BadArchive: func(e *fail.Error) (string, string) {
			return e.Subject + " is not an agent's archive", "station agents export <agent> --to <folder> writes one"
		},
		skillscalls.NotFound: func(e *fail.Error) (string, string) {
			return "no skill named \"" + e.Subject + "\"", "station skills"
		},
		skillscalls.Exists: func(e *fail.Error) (string, string) {
			return "a skill named \"" + e.Subject + "\" exists", "pick another name"
		},
		skillscalls.BadName: func(e *fail.Error) (string, string) {
			return "\"" + e.Subject + "\" is not a name a skill takes", "lowercase letters, digits and dashes, a letter first, and no verb of skills"
		},
		projectcalls.NotFound: func(e *fail.Error) (string, string) {
			return "no project in " + e.Subject + " or above it: no spwn.yaml", "station project create <name>"
		},
		projectcalls.Exists: func(e *fail.Error) (string, string) {
			return e.Subject + " exists: a project is made as a new folder", "pick another name"
		},
		projectcalls.BadName: func(e *fail.Error) (string, string) {
			return "\"" + e.Subject + "\" is not a name a project takes", "lowercase letters, digits and dashes, a letter or a digit first"
		},
		projectcalls.UnknownHarness: func(e *fail.Error) (string, string) {
			return "no harness named \"" + e.Subject + "\"", "one of " + e.Detail["known"]
		},
		projectcalls.UnknownExample: func(e *fail.Error) (string, string) {
			return "no example named \"" + e.Subject + "\"", "one of " + e.Detail["known"]
		},
		projectcalls.ExampleDeclaresHarness: func(e *fail.Error) (string, string) {
			return "the example \"" + e.Subject + "\" declares its own harness", "drop --harness, or make a blank project with it"
		},
		projectcalls.Invalid: func(e *fail.Error) (string, string) {
			n := e.Detail["errors"]
			noun := " errors"
			if n == "1" {
				noun = " error"
			}
			return "the project at " + e.Subject + " has " + n + noun, "fix each one above, then station project plan"
		},
		projectcalls.LinkInTheWay: func(e *fail.Error) (string, string) {
			return e.Subject + " is a link: apply never follows one", "remove the link, then station project apply"
		}, backgroundcalls.Unknown: func(e *fail.Error) (string, string) {
			return "no background task \"" + e.Subject + "\"", "station background"
		},
		backgroundcalls.RootScope: func(e *fail.Error) (string, string) {
			return e.Subject + " runs as root: Machines never runs sudo", e.Detail["command"]
		},
		backgroundcalls.IsTheEngine: func(*fail.Error) (string, string) {
			return "that is Machines' engine", "station engine stop"
		},
		settingscalls.Unknown: func(e *fail.Error) (string, string) {
			return "no setting named \"" + e.Subject + "\"", "station settings"
		},
		settingscalls.IsChecked: func(e *fail.Error) (string, string) {
			return e.Subject + " is checked, not set: Machines never runs sudo", e.Detail["fix"]
		},
		settingscalls.NotForThisMachine: func(e *fail.Error) (string, string) {
			if e.Detail["why"] == "Mac only" {
				return e.Subject + " is for Macs only", "station settings apply " + e.Subject + " --on this-mac"
			}
			return e.Subject + " is for " + strings.TrimSuffix(e.Detail["why"], " only") + " only", "station settings"
		},
		settingscalls.NeedsAdmin: func(e *fail.Error) (string, string) {
			line := SettingsAsRoot(cmp.Or(e.Detail["verb"], "apply"), e.Subject)
			if mac := e.Detail["label"]; mac != "" {
				line = catalog.OnMac(mac, line)
			}
			return e.Subject + " needs admin rights", line
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
			return e.Subject + " keeps nothing Machines can rebuild", "station disk"
		},
		diskcalls.NeverTouched: func(e *fail.Error) (string, string) {
			return e.Detail["path"] + " is never touched: Machines erases no key, token or setting", "station disk"
		},
		diskcalls.NothingToErase: func(e *fail.Error) (string, string) {
			return e.Subject + " has no data Machines can erase", cmp.Or(e.Detail["hint"], "station disk")
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
		diskcalls.Linked: func(e *fail.Error) (string, string) {
			return e.Subject + " is a link to " + e.Detail["target"] + ", not moved", "station disk"
		},
		diskcalls.DestinationHeld: func(e *fail.Error) (string, string) {
			held := e.Detail["path"]
			return held + " already exists: " + e.Subject + " cannot move onto it", "move or remove " + held + ", then station disk move " + e.Subject
		},
		diskcalls.NotFreeable: func(e *fail.Error) (string, string) {
			return e.Subject + " is not freeable: in use, gone or not Machines', so nothing is freed", "station disk"
		},
		diskcalls.NamesNeeded: func(*fail.Error) (string, string) {
			return "nothing named: free-unused frees only what it names", "station disk"
		},
		journalcalls.UnknownMac: func(e *fail.Error) (string, string) {
			return "no Mac named \"" + e.Subject + "\"", "station machines"
		},
		journalcalls.UnknownMachine: func(e *fail.Error) (string, string) {
			return "no machine named \"" + e.Subject + "\"", "station machines"
		},
		journalcalls.BadDay: func(e *fail.Error) (string, string) {
			return "\"" + e.Subject + "\" is not a day: write it YYYY-MM-DD", "station journal"
		},
		journalcalls.Unreadable: func(e *fail.Error) (string, string) {
			return e.Subject + " cannot be read: what was done is not shown until it can", "repair " + e.Subject + " by hand, or remove it: rm " + e.Subject
		},
		diskcalls.RecordUnreadable: func(e *fail.Error) (string, string) {
			return e.Subject + " cannot be read: nothing is freed until it can", "repair " + e.Subject + " by hand, or remove it: rm " + e.Subject
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
			return `"` + e.Subject + `" is not a route Machines knows`, "--via is one of " + e.Detail["routes"]
		},
		ports.CodeUnknownAudience: func(e *fail.Error) (string, string) {
			return `"` + e.Subject + `" is not an audience`, "--to is one of " + e.Detail["audiences"]
		},
		ports.CodeAudienceInside: func(e *fail.Error) (string, string) {
			return "a VM's share takes the audience its rights give it", "on its Mac: station ports share " + e.Detail["vm"] + " " + e.Subject + " --to " + e.Detail["to"]
		},
		ports.CodeEngineTooOld: func(e *fail.Error) (string, string) {
			older := "an older station"
			if v := e.Detail["version"]; v != "" {
				older += " (" + v + ")"
			}
			return "the engine on " + e.Subject + " runs " + older + ", which cannot set a share's audience", "on " + e.Subject + ": station engine start"
		},
		ports.CodeAudienceNotKept: func(e *fail.Error) (string, string) {
			got := cmp.Or(e.Detail["got"], "its rights' audience")
			shared := "the engine on " + e.Detail["mac"] + " shared " + e.Subject + " with " + got + ", not " + e.Detail["asked"]
			if e.Detail["released"] == "" {
				if cause := e.Detail["cause"]; cause != "" {
					shared += " (" + cause + ")"
				}
				return shared + ", and it is still shared", "station ports stop-sharing " + e.Detail["vm"] + " " + e.Detail["port"]
			}
			return shared + ": it is shared no more, and stays closed until you share it again", "on " + e.Detail["mac"] + ": station engine start, then station ports share " + e.Detail["vm"] + " " + e.Detail["port"] + " --to " + e.Detail["asked"]
		},
		ports.CodeNotDeclared: func(e *fail.Error) (string, string) {
			return e.Subject + " is not declared public", "station ports --on " + e.Detail["machine"]
		},
		ports.CodeElsewhere: func(e *fail.Error) (string, string) {
			return e.Subject + " runs on " + e.Detail["mac"], "add --on " + e.Detail["mac"]
		},
		call.BadMacName: func(e *fail.Error) (string, string) {
			return e.Detail["why"], machines.MacRecordFix
		},
		call.NoStation: func(e *fail.Error) (string, string) {
			return "station is not installed on " + cmp.Or(e.Detail["label"], e.Subject), "install it there (" + catalog.InstallLine + "), then add it again"
		},
		call.FarTooOld: func(e *fail.Error) (string, string) {
			far := cmp.Or(e.Detail["label"], e.Subject)
			repair := "update station on " + far + " (run " + catalog.InstallLine + " there)"
			if image := e.Detail["image"]; image != "" {
				repair = "station images build " + image + ", then make " + far + " again from it: station machines delete " + far + " --confirm=" + far + ", then station machines create " + far + " --image " + image
			}
			if lacked := e.Detail["call"]; lacked != "" {
				return far + " runs station " + e.Detail["version"] + ", which has no " + lacked, repair
			}
			return far + " runs station " + e.Detail["version"] + ": this station speaks only to its own release", repair
		},
		call.CallerTooOld: func(e *fail.Error) (string, string) {
			repair := "update station here (" + catalog.InstallLine + ")"
			if far := cmp.Or(e.Detail["label"], e.Subject); far != "" {
				return far + " runs station " + e.Detail["version"] + ": this station speaks only to its own release", repair
			}
			return "station " + e.Detail["version"] + " is not the release the far station runs, the only one it speaks to", repair
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
