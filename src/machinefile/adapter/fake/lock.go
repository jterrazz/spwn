package fake

import (
	"context"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/world"
)

// value: the locks of a Mac of the world, kept as its station files
// `machinefiles/<name>.lock`, so a story seeds one with `world edit` and
// a lock is read and written in the bytes the real folder holds.

type Locks struct {
	w   *world.World
	mac string
}

var _ contract.Locks = Locks{}

func LocksOf(w *world.World, mac string) Locks { return Locks{w: w, mac: mac} }

func (l Locks) Where(name string) string {
	return "~/.config/station/machinefiles/" + name + catalog.LockSuffix
}

func (l Locks) Read(_ context.Context, name string) (catalog.LockFile, bool, error) {
	l.w.Lock()
	body, ok := l.w.Mac(l.mac).Files["machinefiles/"+name+catalog.LockSuffix]
	l.w.Unlock()
	if !ok {
		return catalog.LockFile{}, false, nil
	}
	lock, err := catalog.DecodeLock(l.Where(name), []byte(body))
	if err != nil {
		return catalog.LockFile{}, false, &machinefile.RefusedFileError{Format: "machinefile", Err: err}
	}
	return lock, true, nil
}

func (l Locks) Write(_ context.Context, was, now catalog.LockFile) error {
	l.w.Lock()
	defer l.w.Unlock()
	m := l.w.Mac(l.mac)
	if m.Files == nil {
		m.Files = map[string]string{}
	}
	at := "machinefiles/" + now.Machinefile + catalog.LockSuffix
	if body, ok := m.Files[at]; ok {
		current, err := catalog.DecodeLock(l.Where(now.Machinefile), []byte(body))
		if err != nil {
			return &machinefile.RefusedFileError{Format: "machinefile", Err: err}
		}
		now = machinefile.Merged(current, was, now)
	}
	body, err := catalog.EncodeLock(now)
	if err != nil {
		return err
	}
	m.Files[at] = string(body)
	return nil
}
