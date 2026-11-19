package section

import (
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type ID string

const (
	Overview   ID = "overview"
	Activity   ID = "activity"
	Apps       ID = "apps"
	Settings   ID = "settings"
	Disk       ID = "disk"
	Ports      ID = "ports"
	Accounts   ID = "accounts"
	Background ID = "background"
	Awake      ID = "awake"
	SSH        ID = "ssh"
	Images     ID = "images"
)

func (ID) Enum() []string {
	order := Order()
	out := make([]string, len(order))
	for i, id := range order {
		out[i] = string(id)
	}
	return out
}

func Order() []ID {
	// invariant: the one order of the sections, on every surface.
	return []ID{Overview, Activity, Apps, Settings, Disk, Ports, Accounts, Background, Awake, SSH, Images}
}

func For(k machine.Kind) []ID {
	// invariant: the table of SPEC §3.2. Images is an All machines section, and
	// a workspace's Background still needs its job manager (background's rule).
	switch k {
	case machine.Computer:
		return []ID{Overview, Activity, Apps, Settings, Disk, Ports, Accounts, Background, Awake}
	case machine.Workspace:
		return []ID{Overview, Activity, Apps, Settings, Disk, Ports, Accounts, Background}
	case machine.Server:
		return []ID{Overview, Activity, Ports, SSH}
	default:
		return nil
	}
}

func (id ID) Title() string {
	switch id {
	case SSH:
		return "SSH"
	default:
		return strings.ToUpper(string(id[:1])) + string(id[1:])
	}
}

func Shown(id ID) machine.Applies {
	// invariant: the one gate of a section (SPEC §3.2, §7.7): a kind the
	// table does not list it on is refused in words; a server is told to be
	// managed by its own tooling, through its ssh.
	return func(p machine.Profile) machine.Verdict {
		if slices.Contains(For(p.Kind), id) {
			return machine.Yes
		}
		if p.Kind == machine.Server {
			return machine.Verdict{Why: p.Ref.Name + " is a server: station does not manage its " + noun(id), Fix: "manage it with your own tooling: ssh " + p.Ref.Name}
		}
		return machine.No(p.Ref.Name + " is a " + p.Kind.Word(p.System.OS) + ": it has no " + noun(id) + " section")
	}
}

func noun(id ID) string {
	switch id {
	case Background:
		return "background tasks"
	case Awake:
		return "Awake"
	case SSH:
		return "SSH"
	default:
		return string(id)
	}
}
