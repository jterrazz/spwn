package local

import (
	"errors"
	"os"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestASwapLandsWhileTheFileHoldsWhatWasSeen(t *testing.T) {
	_, at := oldFile(t)
	if err := (disk{}).Swap(at, Seen{Data: []byte("old\n"), There: true}, []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(at)
	if data, _ := os.ReadFile(at); err != nil || string(data) != "new\n" || info.Mode().Perm() != 0o600 {
		t.Errorf("the file holds %q with mode %v, %v; want new, its own 0600", data, info.Mode().Perm(), err)
	}
}

func TestASwapOverAFileAnotherWriterChangedSinceItWasSeenLeavesHisBytes(t *testing.T) {
	for name, seen := range map[string]Seen{
		"other bytes":           {Data: []byte("older\n"), There: true},
		"a file seen not there": {},
	} {
		t.Run(name, func(t *testing.T) {
			dir, at := oldFile(t)
			err := (disk{}).Swap(at, seen, []byte("new\n"), 0o600)
			if !errors.Is(err, ErrChanged) {
				t.Errorf("Swap = %v, want ErrChanged", err)
			}
			if data, _ := os.ReadFile(at); string(data) != "old\n" {
				t.Errorf("the file holds %q, want the other writer's", data)
			}
			if left := names(t, dir); !slices.Equal(left, []string{".zshrc"}) {
				t.Errorf("left %v beside the file", left)
			}
		})
	}
}

func TestASwapReadsTheFileAgainJustBeforeItsRename(t *testing.T) {
	dir, at := oldFile(t)
	d := durable{sync: func(f *os.File) error {
		if f.Name() != dir {
			if err := os.WriteFile(at, []byte("theirs\n"), 0o600); err != nil {
				return err
			}
		}
		return f.Sync()
	}}
	if err := d.swap(at, Seen{Data: []byte("old\n"), There: true}, []byte("new\n"), 0o600); !errors.Is(err, ErrChanged) {
		t.Fatalf("swap = %v, want ErrChanged for a write landed once the new file was synced", err)
	}
	if data, _ := os.ReadFile(at); string(data) != "theirs\n" {
		t.Errorf("the file holds %q, want the other writer's", data)
	}
	if left := names(t, dir); !slices.Equal(left, []string{".zshrc"}) {
		t.Errorf("left %v beside the file", left)
	}
}

func TestASwapOfAFileSeenNotThereLandsOnlyWhereNoNameIsThere(t *testing.T) {
	dir, at := oldFile(t)
	if err := os.Remove(at); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("gone", at); err != nil {
		t.Fatal(err)
	}
	if err := (disk{}).Swap(at, Seen{}, []byte("new\n"), 0o600); !errors.Is(err, ErrChanged) {
		t.Errorf("Swap = %v, want ErrChanged for a name made there since, which a read finds nothing behind", err)
	}
	if to, err := os.Readlink(at); err != nil || to != "gone" {
		t.Errorf("the name there is now %q, %v; want the other writer's link", to, err)
	}
	if left := names(t, dir); !slices.Equal(left, []string{".zshrc"}) {
		t.Errorf("left %v beside the file", left)
	}
	if err := os.Remove(at); err != nil {
		t.Fatal(err)
	}
	if err := (disk{}).Swap(at, Seen{}, []byte("new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(at)
	if data, _ := os.ReadFile(at); err != nil || string(data) != "new\n" || info.Mode().Perm() != 0o600 || !info.Mode().IsRegular() {
		t.Errorf("made %q, %v, %v; want the new file, 0600", data, info, err)
	}
	if left := names(t, dir); !slices.Equal(left, []string{".zshrc"}) {
		t.Errorf("left %v beside the file", left)
	}
}

func TestASwapReadsTheFileAgainNoFurtherThanWhatWasSeen(t *testing.T) {
	_, at := oldFile(t)
	if err := os.Remove(at); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(at, 0o600); err != nil {
		t.Fatal(err)
	}
	endless := make(chan *os.File, 1)
	go func() {
		w, err := os.OpenFile(at, os.O_WRONLY, 0)
		endless <- w
		for i := 0; err == nil && i < 16<<10; i++ {
			_, err = w.WriteString(strings.Repeat("x", 4096))
		}
	}()
	done := make(chan error, 1)
	go func() { done <- unchanged(at, Seen{Data: []byte("old\n"), There: true}) }()
	closeWriter := func() {
		if w := <-endless; w != nil {
			_ = w.Close()
		}
	}
	var err error
	select {
	case err = <-done:
		closeWriter()
	case <-time.After(5 * time.Second):
		t.Error("the file was read on past what was seen")
		closeWriter()
		err = <-done
	}
	if !errors.Is(err, ErrChanged) {
		t.Errorf("unchanged = %v, want ErrChanged", err)
	}
}

func TestASwapInAFolderStationMayNotWriteInRefusesAndNeverOverwrites(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes in any folder")
	}
	dir, at := oldFile(t)
	if err := os.Chmod(dir, 0o500); err != nil { //nolint:gosec // value: a folder its owner may read and search, never write in.
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:gosec // value: the test's folder, emptied by its cleanup.
	if err := (disk{}).Swap(at, Seen{Data: []byte("old\n"), There: true}, []byte("new\n"), 0o600); !errors.Is(err, os.ErrPermission) {
		t.Errorf("Swap = %v, want the folder's refusal", err)
	}
	if data, _ := os.ReadFile(at); string(data) != "old\n" {
		t.Errorf("the file holds %q, want it untouched", data)
	}
}
