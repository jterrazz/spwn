package wire

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/keychain"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	devicescalls "github.com/jterrazz/studio-station/apps/engine/devices/calls"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	devicescmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/devices"
	devicesadd "github.com/jterrazz/studio-station/apps/engine/surface/cli/devices/add"
	devicesforget "github.com/jterrazz/studio-station/apps/engine/surface/cli/devices/forget"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func (m machine) devicesDeps() devicescalls.Deps {
	return devicescalls.Deps{
		Store: m.stores.devices, Sealer: devicescalls.Sealer{Key: m.deviceKey(), Mac: m.profile.Ref.Name},
		Presence: m.devicePresence, Now: m.sys.Clock.Now, Changed: m.bookChanged,
	}
}

func (m machine) bookChanged() {
	if m.far.engine != nil {
		m.far.engine.Rehello()
	}
}

// invariant: a VM of this Mac trusts the book's entries that may open
// pipe and whose seal holds against this Mac's device key; an empty book
// never asks the device key, and a book or a key that cannot be read
// trusts none.
func (m machine) pipeKeys() func() []string {
	if m.profile.Kind != coremachine.Computer {
		return nil
	}
	return func() []string {
		ctx := context.Background()
		book, err := m.stores.devices.Read(ctx)
		if err != nil || len(book.Devices) == 0 {
			return nil
		}
		own, err := m.deviceKey().Public(ctx)
		if err != nil {
			return nil
		}
		return devices.Check(book, m.profile.Ref.Name, own).Opening(devices.ChannelPipe)
	}
}

// invariant: adding a device asks the same sheet a grant asks, within the
// same budget; a world answers what its Mac's presence says.
func (m machine) devicePresence(ctx context.Context, reason string) devices.Answer {
	if m.world != nil {
		return m.world.presence()
	}
	switch askedPresence(ctx, reason) {
	case darwin.PresenceGiven:
		return devices.Given
	case darwin.PresenceRefused:
		return devices.Refused
	case darwin.PresenceAbsent:
	}
	return devices.Absent
}

func askedPresence(_ context.Context, reason string) darwin.Presence {
	return darwin.AskPresence(reason, keychain.PresenceBudget)
}

func devicesCommand(env kit.Env, m machine) *cobra.Command {
	return devicescmd.Command(devicescmd.Deps{ThisMachine: m.where}, env,
		devicesadd.Command(devicesadd.Deps{ThisMachine: m.where}, env),
		devicesforget.Command(devicesforget.Deps{ThisMachine: m.where}, env),
	)
}
