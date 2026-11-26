package devices_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/devices"
)

func TestAFileTheEngineCannotReadStandsForWhatRefusedItWhateverTheHold(t *testing.T) {
	book := devices.Book{Generation: 1, Devices: []devices.Device{{Name: "laptop", Key: "ssh-ed25519 LAPTOP"}}}
	for _, hold := range []devices.Hold{devices.Keep(book.Head()), devices.Refuse(1), devices.Await(book, devices.Sum(nil))} {
		for found, want := range map[devices.Found]devices.Standing{devices.ANewer: devices.Newer, devices.Unread: devices.Unreadable} {
			if got := hold.Standing(devices.Book{}, found); got != want {
				t.Fatalf("a hold %q over a file found %q stands %q, want %q", hold.State, found, got, want)
			}
		}
	}
	if got := devices.Keep(book.Head()).Standing(devices.Book{}, devices.NotABook); got != devices.Changed {
		t.Fatalf("bytes that are no book stand %q, want changed outside Spwn", got)
	}
}

func TestABookAwaitingItsCheckIsANeedOnlyWhenItHoldsADevice(t *testing.T) {
	awaiting := devices.Await(devices.Book{Generation: 1}, devices.Sum(nil))
	if got := awaiting.Standing(devices.Book{}, devices.ABook); got != devices.Kept {
		t.Fatalf("an empty book awaiting its check stands %q, want no need", got)
	}
	laptop := devices.Book{Generation: 1, Devices: []devices.Device{{Name: "laptop", Key: "ssh-ed25519 LAPTOP"}}}
	if got := awaiting.Standing(laptop, devices.ABook); got != devices.Unchecked {
		t.Fatalf("a book of one device awaiting its check stands %q, want unchecked", got)
	}
	if got := awaiting.Standing(devices.Book{}, devices.NotABook); got != devices.Unchecked {
		t.Fatalf("an unreadable book awaiting its check stands %q, want unchecked", got)
	}
	if got := awaiting.Settle(`{"devices":[]}`, devices.Head{}); got.State != devices.Unheld {
		t.Fatalf("a book written while awaiting settles to %+v, want it held for nothing", got)
	}
	if got := awaiting.Settle(devices.Sum(nil), laptop.Head()); got.State != devices.Unheld {
		t.Fatalf("a head that is not the start book's settles to %+v, want it held for nothing", got)
	}
	if got := awaiting.Settle(devices.Sum(nil), devices.Book{Generation: 1}.Head()); got.State != devices.Holding {
		t.Fatalf("the start book's own head settles to %+v, want it held", got)
	}
}
