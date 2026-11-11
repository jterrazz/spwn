package machinescreate

import (
	"cmp"
	"context"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: the flags' sizes, parsed as the hypervisor takes them.
	Resources func(cpus int, memory, disk string) (machines.Resources, error)
	// value: every refusal a question cannot answer; unbuilt when the only
	// one is the image, not built yet, which the call asks to build first.
	Check func(ctx context.Context, r machines.CreateRequest) (unbuilt bool, err error)
	// value: the refusal of an image not built, off a terminal without --yes.
	NotBuilt func(image string) error
	// value: refuses this machine when the call's rule says no, before any
	// check of the request.
	Admit func() error
	// value: All machines, for the Mac question; read only.
	AllMachines func(ctx context.Context) (machines.AllMachines, error)
	ThisMachine page.Context
	Ask         kit.Asker
}

type Input struct {
	Image, Name, Memory, Disk, Network, Check string
	Persistent, Server                        bool
	// value: v3's --permanent, kept hidden for one release: --persistent.
	Permanent bool
	// value: v3's --task and --agent, gone with A2: a create that names
	// them is refused, pointed to station agents start.
	Task  string
	Agent bool
	CPUs  int
	// value: v2's --always-on, kept hidden for one release (SPEC §4.3).
	AlwaysOn bool
	// value: v2's --lifetime, kept hidden for one release (SPEC §4.3).
	Lifetime string
	request  machines.CreateRequest
	// value: another Mac chosen on a terminal, the create sent there.
	mac string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:   "create [<name>]",
		Short: "Make a workspace on this Mac, ephemeral or --persistent; or a server, always on, with --server",
		Example: "  station machines create\n" +
			"  station machines create api --persistent --memory 12G\n" +
			"  station machines create infra --server --check https://infra:6443/readyz --on mac-mini",
		Args: cobra.MaximumNArgs(1),
		Asks: true,
		Ask: func(in *Input) (bool, error) {
			return env.Prompt.Confirm(pagemachines.BuildConfirm(in.request.Image))
		},
		Refuse: func(in *Input) error { return d.NotBuilt(in.request.Image) },
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.Permanent, "permanent", false, "--persistent")
			f.BoolVar(&in.Persistent, "persistent", false, "kept until you delete it; without it, deleted when it powers off from inside")
			f.BoolVar(&in.Server, "server", false, "a server: always on, set up by your own tooling, never by station (default 4 CPU, 8G, 50G)")
			f.StringVar(&in.Check, "check", "", "a server's health check: a URL (https://infra:6443/readyz) or host:port")
			f.BoolVar(&in.AlwaysOn, "always-on", false, "a server (--server)")
			f.StringVar(&in.Image, "image", "linux", "the image the workspace is made from")
			f.IntVar(&in.CPUs, "cpus", 0, "virtual CPUs (default 4)")
			f.StringVar(&in.Memory, "memory", "", "memory with its unit, e.g. 8G or 512M (default 4G)")
			f.StringVar(&in.Disk, "disk", "", "disk size with its unit, e.g. 50G (default 20G)")
			f.StringVar(&in.Network, "network", machines.NetworkNAT, "nat: behind its Mac, no address on your LAN; bridged: its own address on your LAN (macOS images only)")
			f.StringVar(&in.Task, "task", "", "gone: station agents start \"<task>\" --new")
			f.BoolVar(&in.Agent, "agent", false, "gone: station agents start \"<task>\" --new")
			f.StringVar(&in.Name, "name", "", "the workspace's name")
			f.StringVar(&in.Lifetime, "lifetime", "", "ephemeral or persistent (--persistent)")
			_ = f.MarkHidden("permanent")
			_ = f.MarkHidden("name")
			_ = f.MarkHidden("lifetime")
			_ = f.MarkHidden("always-on")
			_ = f.MarkHidden("task")
			_ = f.MarkHidden("agent")
		},
		Read: func(*Input) bool { return false },
		Bind: func(in *Input, args []string) error {
			if len(args) == 1 {
				in.Name = args[0]
			}
			if in.Task != "" || in.Agent {
				return &machines.KindRuleError{Why: "station machines create makes the VM only: an agent is started on it", Fix: taskWords(in)}
			}
			if in.Permanent {
				in.Persistent = true
				env.Out.Now(nowWords(in))
			}
			if in.AlwaysOn {
				in.Server = true
				env.Out.Now(serverWords(in))
			}
			if in.Server && (in.Persistent || in.Lifetime != "") {
				flag := "persistent"
				if !in.Persistent {
					flag = "lifetime"
				}
				return &machines.ServerLifetimeError{Name: cmp.Or(in.Name, "<name>"), Flag: flag}
			}
			r, err := d.Resources(in.CPUs, in.Memory, in.Disk)
			if err != nil {
				return err
			}
			lifetime := machines.Ephemeral
			if in.Persistent {
				lifetime = machines.Persistent
			}
			if in.Lifetime != "" {
				var old machines.Lifetime
				_ = old.UnmarshalText([]byte(in.Lifetime))
				lifetime = old
				if old == machines.Persistent {
					env.Out.Now(nowWords(in))
				}
			}
			in.request = machines.CreateRequest{Image: in.Image, Name: in.Name, Lifetime: lifetime, Network: in.Network, Size: r, Check: in.Check, Server: in.Server}
			return nil
		},
		Validate: func(ctx context.Context, in *Input) error {
			if !env.Far() {
				if err := d.Admit(); err != nil {
					return err
				}
			}
			if unbuilt, err := d.Check(ctx, in.request); !unbuilt {
				return err
			}
			return nil
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if d.Ask != nil {
				if err := ask(ctx, d, in); err != nil {
					return err
				}
			}
			target := env.Target()
			if in.mac != "" {
				target = machine.Ref{Name: in.mac}
			}
			steps := kit.Steps(io)
			var created machines.Created
			schema := call.Schema(machinescalls.CreateWorkspace.Name)
			if in.Server {
				schema = call.Schema(machinescalls.CreateServer.Name)
			}
			err := io.Confirmed(func(answer string) (err error) {
				r := in.request
				if r.Lifetime == machines.Ephemeral {
					// invariant: the default lifetime travels as none, which
					// the create of every station reads as its own default,
					// an older one's included (ADR-032).
					r.Lifetime = ""
				}
				if in.Server {
					created, err = call.Invoke(ctx, env.Calls, target, machinescalls.CreateServer, machinescalls.CreateServerIn{Name: r.Name, Image: r.Image, Network: r.Network, Size: r.Size, Check: r.Check, Confirm: answer}, steps)
				} else {
					created, err = call.Invoke(ctx, env.Calls, target, machinescalls.CreateWorkspace, machinescalls.CreateIn{Request: r, Confirm: answer}, steps)
				}
				return err
			})
			steps.Stop()
			if err != nil {
				return err
			}
			return io.Page(pagemachines.Created(created, d.ThisMachine), pagemachines.CreatedContract(schema, created))
		},
	}, env)
}

// value: the A2 words of a create written with v3's --task.
func taskWords(in *Input) string {
	task := in.Task
	if task == "" {
		task = "<task>"
	}
	words := "station agents start \"" + task + "\" --new"
	if in.Name != "" {
		words += " --name " + in.Name
	}
	return words
}

// value: the v3 words of a create written with v2's --always-on.
func serverWords(in *Input) string {
	words := "station machines create"
	if in.Name != "" {
		words += " " + in.Name
	}
	return words + " --server"
}

// value: the words of a create written with v2's --lifetime lasting or
// v3's --permanent.
func nowWords(in *Input) string {
	words := "station machines create"
	if in.Name != "" {
		words += " " + in.Name
	}
	return words + " --persistent"
}

func ask(ctx context.Context, d Deps, in *Input) error {
	var macs []pagemachines.MacChoice
	if allMachines, allMachinesErr := d.AllMachines(ctx); allMachinesErr == nil {
		for _, h := range allMachines.Macs {
			if h.Reachable {
				macs = append(macs, pagemachines.MacChoice{Name: h.Name, ThisMac: h.ThisMac, Detail: vmCount(len(h.VMs))})
			}
		}
	}
	answers := map[string]string{}
	if form := pagemachines.CreateForm(string(in.request.Lifetime), macs); len(form.Fields) > 0 {
		asked, err := d.Ask.Form(form)
		if err != nil {
			return err
		}
		answers = asked
	}
	for _, h := range macs {
		if h.Name == answers["mac"] && !h.ThisMac {
			in.mac = h.Name
			return nil
		}
	}
	return nil
}

func vmCount(n int) string {
	if n == 1 {
		return "1 VM"
	}
	return strconv.Itoa(n) + " VMs"
}
