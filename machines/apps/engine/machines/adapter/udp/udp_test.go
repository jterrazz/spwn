package udp

import (
	"bytes"
	"context"
	"testing"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

func TestTheUDPWakerKeepsTheWakerContract(t *testing.T) {
	contracttest.Waker(t, func(*testing.T) contracttest.WakerSubject {
		n := localtest.NewNet()
		return contracttest.WakerSubject{Waker: New(n), Sent: func() []string {
			out := make([]string, 0, len(n.Broadcasts))
			for _, d := range n.Broadcasts {
				out = append(out, model.Spelled([6]byte(d.Payload[6:12])))
			}
			return out
		}}
	})
}

func TestTheMagicPacketIsSixFFThenTheAddressSixteenTimesToPort9(t *testing.T) {
	n := localtest.NewNet()
	if err := New(n).Wake(context.Background(), contracttest.Ether); err != nil {
		t.Fatal(err)
	}
	addr := []byte{0xa4, 0x83, 0xe7, 0x1c, 0x20, 0x9f}
	want := append(bytes.Repeat([]byte{0xFF}, 6), bytes.Repeat(addr, 16)...)
	if len(n.Broadcasts) != 1 || n.Broadcasts[0].Port != 9 || !bytes.Equal(n.Broadcasts[0].Payload, want) {
		t.Fatalf("broadcast %+v, want one 102-byte packet to port 9", n.Broadcasts)
	}
}
