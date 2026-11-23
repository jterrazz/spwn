package contracttest

import (
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

type SSHConfigSubject struct {
	Blocks contract.SSHConfig
	// value: the names of the blocks station manages, sorted.
	Managed func(t *testing.T) []string
	// value: the owner writes a Host of their own, outside station's blocks.
	Foreign func(t *testing.T, name string)
}

func block(name string) contract.MachineBlock { return contract.MachineBlock{Name: name, User: "u"} }

func vm(name string) contract.MachineBlock {
	return contract.MachineBlock{Name: name, User: "vmu", Behind: Mac, ThisMac: true}
}

func SSHConfig(t *testing.T, subject func(t *testing.T) SSHConfigSubject) {
	t.Run("a block put is station's, a block removed no longer", func(t *testing.T) {
		s := subject(t)
		if err := s.Blocks.Put(t.Context(), []contract.MachineBlock{block(Mac), vm(VM)}); err != nil {
			t.Fatal(err)
		}
		if got := s.Managed(t); !slices.Equal(got, []string{VM, Mac}) {
			t.Fatalf("after the put station manages %v, want [%s %s]", got, VM, Mac)
		}
		if err := s.Blocks.Remove(t.Context(), VM); err != nil {
			t.Fatal(err)
		}
		if got := s.Managed(t); !slices.Equal(got, []string{Mac}) {
			t.Errorf("after the remove station manages %v, want [%s]", got, Mac)
		}
	})
	t.Run("a block put again stays one block", func(t *testing.T) {
		s := subject(t)
		for range 2 {
			if err := s.Blocks.Put(t.Context(), []contract.MachineBlock{block(Mac)}); err != nil {
				t.Fatal(err)
			}
		}
		if got := s.Managed(t); !slices.Equal(got, []string{Mac}) {
			t.Errorf("station manages %v, want one [%s]", got, Mac)
		}
	})
	t.Run("the whole set replaces the blocks: one not in it goes", func(t *testing.T) {
		s := subject(t)
		if err := s.Blocks.Put(t.Context(), []contract.MachineBlock{block(Mac), vm(VM), vm(Renamed)}); err != nil {
			t.Fatal(err)
		}
		if err := s.Blocks.Replace(t.Context(), []contract.MachineBlock{block(Mac), vm(VM)}); err != nil {
			t.Fatal(err)
		}
		if got := s.Managed(t); !slices.Equal(got, []string{VM, Mac}) {
			t.Errorf("after the replace station manages %v, want [%s %s]", got, VM, Mac)
		}
	})
	t.Run("a block the owner wrote is never shadowed, and the other blocks are still written", func(t *testing.T) {
		s := subject(t)
		s.Foreign(t, Mac)
		var refused *contract.SSHConfigError
		if err := s.Blocks.Put(t.Context(), []contract.MachineBlock{block(Mac), vm(VM)}); !errors.As(err, &refused) {
			t.Fatalf("shadowing the owner's block returned %v, want an SSHConfigError", err)
		}
		if got := s.Managed(t); !slices.Equal(got, []string{VM}) {
			t.Errorf("station manages %v, want only [%s]", got, VM)
		}
	})
	t.Run("a block whose name, user or Mac is no one word is never written, and the other blocks are", func(t *testing.T) {
		s := subject(t)
		blocks := []contract.MachineBlock{
			{Name: Mac, User: "u\n  ProxyCommand touch pwned"},
			{Name: Renamed, User: "u;touch pwned"},
			{Name: "a b", User: "u"},
			{Name: Renamed + "-far", User: "u", Behind: "m\n  ProxyCommand touch pwned"},
			vm(VM),
		}
		var refused *contract.SSHConfigError
		if err := s.Blocks.Put(t.Context(), blocks); !errors.As(err, &refused) {
			t.Fatalf("a block that is no ssh word returned %v, want an SSHConfigError", err)
		}
		if got := s.Managed(t); !slices.Equal(got, []string{VM}) {
			t.Errorf("station manages %v, want only [%s]", got, VM)
		}
	})
}
