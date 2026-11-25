//go:build !world

package wire

import (
	"context"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"time"

	"github.com/spf13/cobra"

	accountscontract "github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	activitycontract "github.com/jterrazz/studio-station/apps/engine/activity/contract"
	appscontract "github.com/jterrazz/studio-station/apps/engine/apps/contract"
	awakecontract "github.com/jterrazz/studio-station/apps/engine/awake/contract"
	backgroundcontract "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	corecontract "github.com/jterrazz/studio-station/apps/engine/core/contract"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	devicescontract "github.com/jterrazz/studio-station/apps/engine/devices/contract"
	diskcontract "github.com/jterrazz/studio-station/apps/engine/disk/contract"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	enginecontract "github.com/jterrazz/studio-station/apps/engine/engine/contract"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	imagescontract "github.com/jterrazz/studio-station/apps/engine/images/contract"
	machinefilecontract "github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/funnel"
	portscontract "github.com/jterrazz/studio-station/apps/engine/ports/contract"
	remotecontract "github.com/jterrazz/studio-station/apps/engine/remote/contract"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
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

func (*worldMac) system(h local.System, _ catalog.Catalog) local.System  { return h }
func (*worldMac) handOver(context.Context) error                         { return nil }
func (*worldMac) runs() bool                                             { return false }
func (*worldMac) logAt(string)                                           {}
func (*worldMac) beside(f func())                                        { f() }
func (*worldMac) watch([]string) func()                                  { return func() {} }
func (*worldMac) stores(*stores)                                         {}
func (*worldMac) self() corecontract.Self                                { return nil }
func (*worldMac) power() awakecontract.Power                             { return nil }
func (*worldMac) hypervisor(string, catalog.Install) contract.Hypervisor { return nil }
func (*worldMac) guests() contract.GuestDialer                           { return nil }
func (*worldMac) deviceKey() contract.DeviceKey                          { return nil }

func specDeviceKey(map[string]string) contract.DeviceKey                            { return nil }
func (*worldMac) localNetwork() func(context.Context) model.LocalNetwork            { return nil }
func (*worldMac) agent() func(context.Context) (func(context.Context) error, error) { return nil }
func (*worldMac) imageSource(string) imagescontract.ImageSource                     { return nil }
func (*worldMac) sshConfig() contract.SSHConfig                                     { return nil }
func (*worldMac) unlocker() contract.Unlocker                                       { return nil }
func (*worldMac) waker() contract.Waker                                             { return nil }
func (*worldMac) processes(kernel contract.Processes) contract.Processes            { return kernel }
func (*worldMac) jobs() backgroundcontract.Jobs                                     { return nil }
func (*worldMac) releases() enginecontract.Releases                                 { return nil }
func (*worldMac) inspector() settingscontract.Inspector                             { return nil }
func (*worldMac) procedureAppliers() []settingscontract.Applier                     { return nil }
func (*worldMac) meter() activitycontract.Meter                                     { return nil }
func (*worldMac) disk() diskcontract.DiskUsage                                      { return nil }
func (*worldMac) listeners() portscontract.Listeners                                { return nil }
func (*worldMac) placement() (ports.VMLivesOn, bool)                                { return ports.VMLivesOn{}, false }
func (*worldMac) tailnet() sharingNode                                              { return nil }
func (*worldMac) funnel() funnel.Funnel                                             { return nil }
func (*worldMac) installer(catalog.Method, bool) appscontract.Installer             { return nil }
func (*worldMac) bundler() appscontract.Bundler                                     { return nil }
func (*worldMac) managers(map[catalog.Method]appscontract.Installer)                {}
func (*worldMac) budgets() lanes.Clock                                              { return nil }
func (*worldMac) poolContext(ctx context.Context) context.Context                   { return ctx }
func (*worldMac) watchContext(ctx context.Context) context.Context                  { return ctx }
func (*worldMac) vault() accountscontract.Vault                                     { return nil }
func (*worldMac) signIns(func() time.Time, func(context.Context, time.Duration) error) accountscontract.SignIns {
	return nil
}
func (*worldMac) passwords(func() time.Time, func(context.Context, time.Duration) error) accountscontract.Passwords {
	return nil
}
func (*worldMac) grants(func() time.Time, func(context.Context, time.Duration) error) accountscontract.Grants {
	return nil
}
func (*worldMac) forgetGrants(context.Context, string) error { return nil }
func (*worldMac) userSource() machinefilecontract.Source     { return nil }
func (*worldMac) owned() machinefilecontract.Owned           { return nil }
func (*worldMac) locks() machinefilecontract.Locks           { return nil }
func (*worldMac) password() machinefilecontract.Password     { return nil }
func (*worldMac) turn() machinefilecontract.Turn             { return nil }
func (*worldMac) phaseShell() machinefilecontract.Shell      { return nil }
func (*worldMac) sharedSource(machinefilecontract.SharedSource) machinefilecontract.Source {
	return nil
}
func (*worldMac) notifier() process.Notifier          { return nil }
func (*worldMac) checks() contract.CheckRunner        { return nil }
func (*worldMac) engineFrom(func() *process.Engine)   {}
func (*worldMac) macs() remotecontract.EndpointDialer { return nil }
func (*worldMac) stopMarks() engine.StopMarks         { return nil }
func (*worldMac) records() engine.Records             { return nil }
func (*worldMac) engineVersion() string               { return "" }
func (*worldMac) engineLacks(string) bool             { return false }
func (*worldMac) holdsFar() bool                      { return true }
func (*worldMac) absent(string) bool                  { return false }
func (*worldMac) stationVersion() string              { return "" }
func (*worldMac) engineStarted() time.Time            { return time.Time{} }

func (*worldMac) presence() devices.Answer { return devices.Absent }

func specDeviceAnchor(map[string]string) devicescontract.Anchor { return nil }
