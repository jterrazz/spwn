package wire

import (
	"context"
	"errors"
	"sync"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/keychain"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	devicescalls "github.com/jterrazz/studio-station/apps/engine/devices/calls"
	devicescontract "github.com/jterrazz/studio-station/apps/engine/devices/contract"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/remote"
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

// value: the calls that write the device book, which only this Mac's
// engine runs, whichever surface or frame asks them.
func devicesWrites() []string { return []string{devicescalls.Add.Name, devicescalls.Forget.Name} }

// invariant: the engine is the device book's only writer: a write asked
// in any other process of this Mac rides engine.sock, and with the engine
// not running it is refused as the engine's own calls are, never run here.
func (m machine) bookWriter() call.Router { return bookWriter{remote.Endpoint(m.engineAPI().c)} }

type bookWriter struct{ engine call.Router }

func (b bookWriter) Invoke(ctx context.Context, target coremachine.Ref, name string, in, out any, p call.Progress) error {
	return downAs(b.engine.Invoke(ctx, target, name, in, out, p), &enginecalls.DownError{})
}

func (b bookWriter) Watch(ctx context.Context, target coremachine.Ref, name string, in any, p call.Progress) (<-chan call.Update[any], error) {
	updates, err := b.engine.Watch(ctx, target, name, in, p)
	return updates, downAs(err, &enginecalls.DownError{})
}

// invariant: a process of this Mac that is not its engine judges the book
// by the head the running engine holds, never by the keychain, which any
// process running `security` can move; with no engine running, by the
// keychain, as the engine does at its start.
type enginePeer struct{ api engineAPI }

func (p enginePeer) Held(ctx context.Context) (devices.Head, bool, error) {
	h, err := p.api.Hello(ctx)
	var down *enginecalls.DownError
	switch {
	case errors.As(err, &down):
		return devices.Head{}, false, nil
	case err != nil:
		return devices.Head{}, false, &devicescontract.AnchorUnreadableError{Cause: err.Error()}
	case h.Book == "":
		return devices.Head{}, false, nil
	case h.Book == (devices.Head{}).String():
		return devices.Head{}, true, nil
	}
	held, err := devices.ParseHead(h.Book)
	if err != nil {
		return devices.Head{}, false, &devicescontract.AnchorUnreadableError{Cause: err.Error()}
	}
	return held, true, nil
}

// value: what the engine's hello says of the book: the head it holds and
// whether the book on disk is another.
func (m machine) heldBook() (string, bool) {
	head, changed, ok := m.stores.devices.Held(context.Background())
	if !ok {
		return "", false
	}
	return head.String(), changed
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
