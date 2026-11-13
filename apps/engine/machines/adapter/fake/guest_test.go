package fake

import (
	"io/fs"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
)

func TestTheFakeGuestKeepsTheContract(t *testing.T) {
	contracttest.Guest(t, func(*testing.T) contracttest.GuestSubject {
		guests := NewGuests()
		vm := guests.Add("dev")
		vm.Serve(contracttest.GuestPort, contracttest.Greeting)
		return contracttest.GuestSubject{
			Dialer:   guests,
			Target:   func(*testing.T) contract.GuestTarget { return contract.GuestTarget{VM: "dev"} },
			Program:  func(line string, a contracttest.Answer) { vm.Program(line, Answer(a)) },
			Received: vm.Received,
			Read: func(t *testing.T, path string) ([]byte, fs.FileMode) {
				f, ok := vm.File(path)
				if !ok {
					t.Fatalf("no file %s in the guest", path)
				}
				return f.Data, f.Mode
			},
			Path: "/tmp/station",
		}
	})
}
