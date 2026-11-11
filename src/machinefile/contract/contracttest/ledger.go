package contracttest

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

func installed(item, method string) machinefile.Installed {
	return machinefile.Installed{Item: item, Method: method, Machinefile: "laptop", At: time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)}
}

func Ledger(t *testing.T, open func(t *testing.T) contract.Ledger) {
	ctx := context.Background()
	t.Run("an install recorded is listed by item, and recorded again replaces it", func(t *testing.T) {
		l := open(t)
		for _, in := range []machinefile.Installed{installed("jq", "first"), installed("fd", "first"), installed("jq", "second")} {
			if err := l.Put(ctx, in); err != nil {
				t.Fatalf("Put: %v", err)
			}
		}
		got, err := l.List(ctx)
		want := []machinefile.Installed{installed("fd", "first"), installed("jq", "second")}
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("List = %+v, %v; want %+v", got, err, want)
		}
	})
	t.Run("a deleted install is gone, and deleting one never recorded changes nothing", func(t *testing.T) {
		l := open(t)
		if err := l.Put(ctx, installed("jq", "first")); err != nil {
			t.Fatal(err)
		}
		for _, item := range []string{"jq", "jq", "fd"} {
			if err := l.Delete(ctx, item); err != nil {
				t.Errorf("Delete(%s) = %v", item, err)
			}
		}
		if got, err := l.List(ctx); err != nil || len(got) != 0 {
			t.Errorf("List = %+v, %v; want none", got, err)
		}
	})
}

// value: an owner's folder whose path holds body (none when path is
// empty), and the owner's source over that folder.

type OwnedSubject func(t *testing.T, path, body string) (contract.Owned, contract.Source)

func Owned(t *testing.T, subject OwnedSubject) {
	ctx := context.Background()
	t.Run("a file read at its path is its bytes; a path with no file is not found", func(t *testing.T) {
		o, _ := subject(t, "laptop.yaml", "name: laptop\n")
		if got, err := o.Read(ctx, "laptop.yaml"); err != nil || string(got) != "name: laptop\n" {
			t.Errorf("Read = %q, %v", got, err)
		}
		if _, err := o.Read(ctx, "nothing.yaml"); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Read of no file = %v; want not found", err)
		}
	})
	t.Run("a Machinefile kept is read back by the owner's source", func(t *testing.T) {
		o, src := subject(t, "", "")
		for _, body := range []string{"old\n", "new\n"} {
			if err := o.Keep(ctx, "laptop", []byte(body)); err != nil {
				t.Fatal(err)
			}
		}
		tree, err := src.Read(ctx)
		if err != nil || len(tree.Files) != 1 || tree.Files[0].Path != "machinefiles/laptop.yaml" || string(tree.Files[0].Body) != "new\n" {
			t.Errorf("read back %+v, %v", tree.Files, err)
		}
	})
}
