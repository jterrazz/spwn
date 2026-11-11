package contracttest

import (
	"context"
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

func lockOf(name string, apps ...catalog.LockedApp) catalog.LockFile {
	return catalog.LockFile{Lock: catalog.LockSchema, Machinefile: name, Apps: apps}
}

// value: the locks of a fresh machine, and a way to put bytes of the
// owner's at a Machinefile's lock.

type LocksSubject func(t *testing.T) (contract.Locks, func(name, body string))

func Locks(t *testing.T, subject LocksSubject) {
	ctx := context.Background()
	t.Run("a lock never written reads as none", func(t *testing.T) {
		l, _ := subject(t)
		if got, ok, err := l.Read(ctx, "laptop"); err != nil || ok || len(got.Apps) != 0 {
			t.Errorf("Read = %+v, %v, %v; want none", got, ok, err)
		}
	})
	t.Run("a lock written is read back, each Machinefile its own", func(t *testing.T) {
		l, _ := subject(t)
		laptop := lockOf("laptop", catalog.LockedApp{Name: "jq", Method: "pkg", Version: "1.7.1"})
		laptop.Source = &catalog.LockSource{Name: "studio", Commit: "0123456789abcdef0123456789abcdef01234567"}
		for _, w := range []catalog.LockFile{laptop, lockOf("server", catalog.LockedApp{Name: "node", Method: "rt", Version: "22.11.0"})} {
			if err := l.Write(ctx, w); err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
		got, ok, err := l.Read(ctx, "laptop")
		if err != nil || !ok || !reflect.DeepEqual(got, laptop) {
			t.Errorf("Read = %+v, %v, %v; want %+v", got, ok, err, laptop)
		}
	})
	t.Run("a lock edited into another shape is refused at its file", func(t *testing.T) {
		l, put := subject(t)
		put("laptop", "lock: station.lock/v1\nmachinefile: laptop\napps:\n    - { name: jq }\n")
		if _, _, err := l.Read(ctx, "laptop"); err == nil {
			t.Error("Read = nil; want the lock refused")
		}
	})
}
