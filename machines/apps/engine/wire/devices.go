package wire

import (
	"context"
	"sync"

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
// pipe, when the book is the one this Mac anchored last and each seal holds
// against the device key this process holds; an empty book never asks the
// key, a book, an anchor or a key that cannot be read trusts none. The
// verdict is kept while the book's head, the anchored head and the key are
// the ones it was drawn from, so a walk re-checks no seal.
func (m machine) pipeKeys() func() []string {
	if m.profile.Kind != coremachine.Computer {
		return nil
	}
	var mu sync.Mutex
	var drawn struct {
		head, anchored devices.Head
		key            string
		keys           []string
	}
	return func() []string {
		ctx := context.Background()
		book, anchored, err := m.stores.devices.Read(ctx)
		if err != nil || len(book.Devices) == 0 {
			return nil
		}
		own, err := m.deviceKey().Kept(ctx)
		if err != nil {
			return nil
		}
		line, err := devices.LineOf(own)
		if err != nil {
			return nil
		}
		head := book.Head()
		mu.Lock()
		defer mu.Unlock()
		if drawn.key != line || drawn.head != head || drawn.anchored != anchored {
			drawn.head, drawn.anchored, drawn.key = head, anchored, line
			drawn.keys = devices.Check(book, anchored, m.profile.Ref.Name, own).Opening(devices.ChannelPipe)
		}
		return drawn.keys
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
