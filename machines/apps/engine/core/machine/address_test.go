package machine_test

import (
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestAnAddressIsABareNameOrOneVMOfOneComputer(t *testing.T) {
	for _, c := range []struct {
		in   string
		want machine.Ref
	}{
		{"development-linux", machine.Ref{Name: "development-linux"}},
		{"development-linux.mini", machine.Ref{Name: "development-linux", On: "mini"}},
		{"mini", machine.Ref{Name: "mini"}},
	} {
		got, err := machine.ParseAddress(c.in)
		if err != nil || got != c.want {
			t.Errorf("ParseAddress(%q) = %+v, %v; want %+v", c.in, got, err, c.want)
		}
		if got.Address() != c.in {
			t.Errorf("%+v.Address() = %q, want %q", got, got.Address(), c.in)
		}
	}
}

func TestAnAddressWithAnEmptyHalfOrASecondDotIsRefused(t *testing.T) {
	for _, in := range []string{"", ".", "development-linux.", ".mini", "a.b.c", "mini.tail1234.ts.net"} {
		_, err := machine.ParseAddress(in)
		var bad *machine.BadAddressError
		if !errors.As(err, &bad) || bad.Address != in {
			t.Errorf("ParseAddress(%q) = %v, want a BadAddressError", in, err)
		}
	}
}
