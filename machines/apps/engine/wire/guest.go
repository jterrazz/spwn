package wire

import (
	"context"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/docker"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/guest"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/lsof"
	macosports "github.com/jterrazz/studio-station/apps/engine/ports/adapter/macos"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/procnet"
	portscontract "github.com/jterrazz/studio-station/apps/engine/ports/contract"
	guestcmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/guest"
	guestserve "github.com/jterrazz/studio-station/apps/engine/surface/cli/guest/serve"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func (m machine) guest() *guest.Agent {
	root := m.systemRoot()
	if root == "" {
		root = "/"
	}
	return guest.New(m.sys, guest.Options{State: m.paths.State, PortsEnv: filepath.Join(m.paths.Prefs, "services.env"), Kind: m.kind, Ephemeral: m.ephemeral(root), Version: Version(), Listeners: m.listeners(root), Containers: m.containers()})
}

// value: the ports this VM's containers publish; none in a world.
func (m machine) containers() guest.Containers {
	if m.world != nil {
		return nil
	}
	return docker.New(m.sys.Run)
}

// value: station guest, refused on a Mac.
type guestServer struct {
	*guest.Agent
	kind catalog.MachineKind
}

func (g guestServer) Check() error {
	if g.kind == catalog.Mac {
		return &model.GuestOnMacError{}
	}
	return nil
}

func (g guestServer) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	if err := g.Check(); err != nil {
		return err
	}
	return g.Agent.Run(ctx, in, out)
}

// value: what listens here, read the way this system can: a Linux kernel's
// /proc/net tables under root, a Mac's lsof.
func (m machine) listeners(root string) portscontract.Listeners {
	if m.world != nil {
		return m.world.listeners()
	}
	if m.kind == catalog.Linux {
		return procnet.New(m.sys.FS, root)
	}
	return lsof.New(m.sys.Run)
}

// value: the range of ports this system gives outgoing connections: a Linux
// kernel's /proc/sys under root, a macOS's kernel through its tool; a world
// reads none.
func (m machine) ephemeral(root string) guest.Ephemeral {
	switch {
	case m.world != nil:
		return nil
	case m.kind == catalog.Linux:
		return procnet.New(m.sys.FS, root)
	}
	return macosports.New(m.sys.Run)
}

func guestCommand(env kit.Env, m machine) *cobra.Command {
	cmd := guestcmd.Command(guestcmd.Deps{Guest: guestServer{m.guest(), m.kind}}, env)
	cmd.AddCommand(guestserve.Command(guestserve.Deps{Serve: serveFrames(env, m.socketDoor())}, env))
	return cmd
}
