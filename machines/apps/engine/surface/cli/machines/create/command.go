package machinescreate

import (
	"cmp"
	"context"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
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
	// value: the name of the Machinefile --from names: a name as it is, a
	// file of this Mac copied among the owner's Machinefiles first.
	Machinefile func(ctx context.Context, from string) (string, error)
	// value: All machines, for the Mac question; read only.
	AllMachines func(ctx context.Context) (machines.AllMachines, error)
	// value: the session this command runs for, by its name
	// (STATION_SESSION); empty for the owner.
	Session     string
	ThisMachine page.Context
	Ask         kit.Asker
}

type Input struct {
	Image, Name, Memory, Disk, Network, Check string
	// value: what the VM is for, free text.
	Purpose            string
	Persistent, Server bool
	CPUs               int
	// value: the Machinefile the workspace is made from.
	From    string
	flags   *pflag.FlagSet
	request machines.CreateRequest
	// value: another Mac chosen on a terminal, the create sent there.
	mac string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:   "create [<name>]",
		Short: "Make a workspace on this Mac, ephemeral or --persistent; or a server, always on, with --server",
		Example: "  station machines create\n" +
			"  station machines create api --persistent --memory 12G\n" +
			"  station machines create web --from studio-linux\n" +
			"  station machines create infra --server --check https://infra:6443/readyz --on mac-mini",
		Args: cobra.MaximumNArgs(1),
		Asks: true,
		Ask: func(in *Input) (bool, error) {
			return env.Prompt.Confirm(pagemachines.BuildConfirm(in.request.Image))
		},
		Refuse: func(in *Input) error { return d.NotBuilt(in.request.Image) },
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.Persistent, "persistent", false, "kept until you delete it; without it, deleted when it powers off from inside")
			f.BoolVar(&in.Server, "server", false, "a server: always on, set up by your own tooling, never by Machines (default 4 CPU, 8G, 50G)")
			f.StringVar(&in.Check, "check", "", "a server's health check: a URL (https://infra:6443/readyz) or host:port")
			in.flags = f
			f.StringVar(&in.Image, "image", "base-ubuntu", "the image the workspace is made from")
			f.StringVar(&in.From, "from", "", "the Machinefile the workspace is made from: the image built from it, and how its vm says it runs")
			f.IntVar(&in.CPUs, "cpus", 0, "virtual CPUs (default 4)")
			f.StringVar(&in.Memory, "memory", "", "memory with its unit, e.g. 8G or 512M (default 8G)")
			f.StringVar(&in.Disk, "disk", "", "disk size with its unit, e.g. 50G (default: the image's, set by vm.resources in its Machinefile: 20G on base-ubuntu and base-debian, 80G on base-macos)")
			f.StringVar(&in.Purpose, "for", "", "what it is for, in your words, kept with the VM")
			f.StringVar(&in.Network, "network", machines.NetworkNAT, "nat: behind its Mac, no address on your LAN; bridged: its own address on your LAN (macOS images only)")
			f.StringVar(&in.Name, "name", "", "the workspace's name")
			_ = f.MarkHidden("name")
		},
		Read: func(*Input) bool { return false },
		Bind: func(in *Input, args []string) error {
			if len(args) == 1 {
				in.Name = args[0]
			}
			if in.Server && in.Persistent {
				return &machines.ServerLifetimeError{Name: cmp.Or(in.Name, "<name>"), Flag: "persistent"}
			}
			r, err := d.Resources(in.CPUs, in.Memory, in.Disk)
			if err != nil {
				return err
			}
			lifetime := machines.Ephemeral
			if in.Persistent {
				lifetime = machines.Persistent
			}
			in.request = machines.CreateRequest{Image: in.Image, Name: in.Name, Lifetime: lifetime, Network: catalog.Network(in.Network), Size: r, Check: in.Check, Server: in.Server, Purpose: in.Purpose, Creator: d.Session}
			return from(in)
		},
		Validate: func(ctx context.Context, in *Input) error {
			if !env.Far() {
				if err := d.Admit(); err != nil {
					return err
				}
				if in.From != "" {
					name, err := d.Machinefile(ctx, in.From)
					if err != nil {
						return err
					}
					in.request.Image, in.request.From = name, name
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
			schema := machinescalls.CreateWorkspace.Schema()
			if in.Server {
				schema = machinescalls.CreateServer.Schema()
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
					created, err = call.Invoke(ctx, env.Calls, target, machinescalls.CreateServer, machinescalls.CreateServerIn{Name: r.Name, Image: r.Image, Network: r.Network, Size: r.Size, Check: r.Check, Purpose: r.Purpose, Creator: r.Creator, Confirm: answer}, steps)
				} else {
					created, err = call.Invoke(ctx, env.Calls, target, machinescalls.CreateWorkspace, machinescalls.CreateIn{Request: r, Confirm: answer}, steps)
				}
				return err
			})
			steps.Stop()
			if err != nil {
				return err
			}
			return io.Page(pagemachines.Created(created, d.ThisMachine, func(g machines.FirstStartGrant) string { return kit.Refusal(g.Code, g.Subject) }), pagemachines.CreatedContract(schema, created))
		},
	}, env)
}

// invariant: a create --from makes the image named after its Machinefile,
// so it names no other; the network its flag does not say is the
// Machinefile's, and a server is set up by its own tooling, never from a
// Machinefile.
func from(in *Input) error {
	if in.From == "" {
		return nil
	}
	switch {
	case in.Server:
		return &machines.ServerTemplateError{Name: in.From}
	case in.flags != nil && in.flags.Changed("image"):
		return &machines.KindRuleError{Why: "--from names the image: the one built from its Machinefile", Fix: "station machines create --from " + in.From}
	}
	in.request.Image, in.request.From = in.From, in.From
	if in.flags != nil && !in.flags.Changed("network") {
		in.request.Network = ""
	}
	return nil
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
