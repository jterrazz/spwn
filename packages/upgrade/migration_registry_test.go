package upgrade

import (
	"context"
	"testing"
)

func TestRegistryRegisterAndAll(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Migration{Number: 1, Description: "first", Apply: func(_ context.Context, _ string) error { return nil }})
	reg.Register(Migration{Number: 2, Description: "second", Apply: func(_ context.Context, _ string) error { return nil }})
	reg.Register(Migration{Number: 5, Description: "fifth", Apply: func(_ context.Context, _ string) error { return nil }})

	all := reg.All()
	if len(all) != 3 {
		t.Fatalf("len = %d, want 3", len(all))
	}
	if all[0].Number != 1 || all[1].Number != 2 || all[2].Number != 5 {
		t.Errorf("unexpected numbers: %d, %d, %d", all[0].Number, all[1].Number, all[2].Number)
	}
}

func TestRegistryPanicsOnOutOfOrder(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Migration{Number: 3, Description: "three", Apply: func(_ context.Context, _ string) error { return nil }})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for out-of-order registration")
		}
	}()

	reg.Register(Migration{Number: 2, Description: "two", Apply: func(_ context.Context, _ string) error { return nil }})
}

func TestRegistryPanicsOnDuplicate(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Migration{Number: 1, Description: "one", Apply: func(_ context.Context, _ string) error { return nil }})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for duplicate number")
		}
	}()

	reg.Register(Migration{Number: 1, Description: "one-again", Apply: func(_ context.Context, _ string) error { return nil }})
}

func TestRegistryAllReturnsCopy(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Migration{Number: 1, Description: "one", Apply: func(_ context.Context, _ string) error { return nil }})

	all := reg.All()
	all[0].Number = 999

	// Original should be unchanged
	original := reg.All()
	if original[0].Number != 1 {
		t.Errorf("All() did not return a copy; original was mutated to %d", original[0].Number)
	}
}
