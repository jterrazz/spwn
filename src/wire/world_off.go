//go:build !world

package wire

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	accountscontract "github.com/jterrazz/studio-station/src/accounts/contract"
	activitycontract "github.com/jterrazz/studio-station/src/activity/contract"
	"github.com/jterrazz/studio-station/src/agents"
	agentscontract "github.com/jterrazz/studio-station/src/agents/contract"
	appscontract "github.com/jterrazz/studio-station/src/apps/contract"
	awakecontract "github.com/jterrazz/studio-station/src/awake/contract"
	backgroundcontract "github.com/jterrazz/studio-station/src/background/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call/lanes"
	corecontract "github.com/jterrazz/studio-station/src/core/contract"
	diskcontract "github.com/jterrazz/studio-station/src/disk/contract"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/engine/process"
	imagescontract "github.com/jterrazz/studio-station/src/images/contract"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/ports"
	"github.com/jterrazz/studio-station/src/ports/adapter/funnel"
	portscontract "github.com/jterrazz/studio-station/src/ports/contract"
	remotecontract "github.com/jterrazz/studio-station/src/remote/contract"
	settingscontract "github.com/jterrazz/studio-station/src/settings/contract"
)

// invariant: the release build knows no world: openWorld answers none, so
// every seam below stays the real adapter's and no world command exists.
type (
	worldRun struct{}
	worldMac struct{}
)

func openWorld(map[string]string, bool) (*worldRun, error) { return nil, nil }

func (*worldRun) thisMac() *worldMac             { return nil }
func (*worldRun) farFrom(func(*worldMac) farMac) {}
func (*worldRun) command() *cobra.Command        { return nil }
func (*worldRun) close(context.Context) error    { return nil }

func (*worldMac) system(h local.System) local.System                    { return h }
func (*worldMac) handOver(context.Context) error                        { return nil }
func (*worldMac) watch([]string) func()                                 { return func() {} }
func (*worldMac) stores(*stores)                                        {}
func (*worldMac) self() corecontract.Self                               { return nil }
func (*worldMac) power() awakecontract.Power                            { return nil }
func (*worldMac) hypervisor(string) contract.Hypervisor                 { return nil }
func (*worldMac) guests() contract.GuestDialer                          { return nil }
func (*worldMac) imageSource(string) imagescontract.ImageSource         { return nil }
func (*worldMac) sshConfig() contract.SSHConfig                         { return nil }
func (*worldMac) unlocker() contract.Unlocker                           { return nil }
func (*worldMac) jobs() backgroundcontract.Jobs                         { return nil }
func (*worldMac) inspector() settingscontract.Inspector                 { return nil }
func (*worldMac) procedureAppliers() []settingscontract.Applier         { return nil }
func (*worldMac) meter() activitycontract.Meter                         { return nil }
func (*worldMac) disk() diskcontract.DiskUsage                          { return nil }
func (*worldMac) listeners() portscontract.Listeners                    { return nil }
func (*worldMac) journal() agentscontract.Journal                       { return nil }
func (*worldMac) hooks(agents.App) agentscontract.Hooks                 { return nil }
func (*worldMac) placement() (ports.VMLivesOn, bool)                    { return ports.VMLivesOn{}, false }
func (*worldMac) tailnet() portscontract.Tailnet                        { return nil }
func (*worldMac) funnel() funnel.Funnel                                 { return nil }
func (*worldMac) installer(catalog.Method, bool) appscontract.Installer { return nil }
func (*worldMac) bundler() appscontract.Bundler                         { return nil }
func (*worldMac) budgets() lanes.Clock                                  { return nil }
func (*worldMac) vault() accountscontract.Vault                         { return nil }
func (*worldMac) identities() accountscontract.Identities               { return nil }
func (*worldMac) secretSource() accountscontract.SecretSource           { return nil }
func (*worldMac) notifier() process.Notifier                            { return nil }
func (*worldMac) checks() contract.CheckRunner                          { return nil }
func (*worldMac) engineFrom(func() *process.Engine)                     {}
func (*worldMac) macs() remotecontract.EndpointDialer                   { return nil }
func (*worldMac) stopMarks() engine.StopMarks                           { return nil }
func (*worldMac) engineVersion() string                                 { return "" }
func (*worldMac) stationVersion() string                                { return "" }
func (*worldMac) engineStarted() time.Time                              { return time.Time{} }
