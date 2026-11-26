package wire

import (
	"context"
	"errors"

	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/devices"
)

var (
	errDoorNotHere = errors.New("this machine opens no door: only a Mac does")
	errBookNotHeld = errors.New("this process does not hold the device book: only the engine lets a device in")
	errNotInBook   = errors.New("not in the device book")
)

func bookTrusted(standing devices.Standing) error {
	switch standing {
	case devices.Kept:
		return nil
	case devices.Changed:
		return errors.New("the device book changed outside Spwn: nobody is let in until a device is added again")
	case devices.Unchecked:
		return errors.New("the device book is not checked yet: nobody is let in until the login keychain answers")
	case devices.Unreadable:
		return errors.New("the device book cannot be read: nobody is let in")
	case devices.Newer:
		return errors.New("the device book was written by a newer Spwn: nobody is let in")
	}
	return errors.New("the device book stands " + string(standing) + ": nobody is let in")
}

// invariant: the door lets a key in only from the book this Mac's engine
// holds: in a process that is not the engine, with a book changed, not
// checked yet, unreadable or a newer Spwn's, nobody is let in, the entry of
// that very key included; then only an entry of the anchored book whose
// seal holds against the device key the engine holds, and whose key is the
// key offered.
func (m machine) admits(ctx context.Context, key string) (string, error) {
	if m.profile.Kind != coremachine.Computer {
		return "", errDoorNotHere
	}
	_, standing, ok := m.stores.devices.Held(ctx)
	if !ok {
		return "", errBookNotHeld
	}
	if err := bookTrusted(standing); err != nil {
		return "", err
	}
	book, anchored, err := m.stores.devices.Read(ctx)
	if err != nil {
		return "", err
	}
	own, err := m.deviceKey().Kept(ctx)
	if err != nil {
		return "", err
	}
	d, ok := devices.Check(book, anchored, m.profile.Ref.Name, own).Holder(key)
	if !ok {
		return "", errNotInBook
	}
	return d.Name, nil
}
