package contracttest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const Ether = "a4:83:e7:1c:20:9f"

type WakerSubject struct {
	Waker contract.Waker
	// value: the hardware address each packet sent named, in order.
	Sent func() []string
}

func Waker(t *testing.T, subject func(t *testing.T) WakerSubject) {
	t.Run("a wake sends one packet naming the Mac's hardware address", func(t *testing.T) {
		s := subject(t)
		if err := s.Waker.Wake(context.Background(), Ether); err != nil {
			t.Fatalf("Wake: %v", err)
		}
		if got := s.Sent(); !slices.Equal(got, []string{Ether}) {
			t.Fatalf("sent %v, want one packet for %s", got, Ether)
		}
	})
	t.Run("an address spelled with dashes or capitals names the same Mac", func(t *testing.T) {
		s := subject(t)
		if err := s.Waker.Wake(context.Background(), "A4-83-E7-1C-20-9F"); err != nil {
			t.Fatalf("Wake: %v", err)
		}
		if got := s.Sent(); !slices.Equal(got, []string{Ether}) {
			t.Fatalf("sent %v, want one packet for %s", got, Ether)
		}
	})
	t.Run("an address that does not parse sends nothing", func(t *testing.T) {
		s := subject(t)
		for _, bad := range []string{"", "a4:83:e7", "zz:83:e7:1c:20:9f", "a4:83:e7:1c:20:9f:00"} {
			if err := s.Waker.Wake(context.Background(), bad); !errors.Is(err, contract.ErrBadHardwareAddress) {
				t.Errorf("Wake(%q) = %v, want ErrBadHardwareAddress", bad, err)
			}
		}
		if got := s.Sent(); len(got) != 0 {
			t.Fatalf("sent %v for addresses that do not parse", got)
		}
	})
}
