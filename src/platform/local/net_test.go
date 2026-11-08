package local

import (
	"net"
	"testing"
)

func TestTheHardwareAddressIsTheOneOfTheInterfaceThatHoldsTheAddress(t *testing.T) {
	ip, want := "", ""
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range ifaces {
		if len(i.HardwareAddr) == 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil {
				ip, want = n.IP.String(), i.HardwareAddr.String()
			}
		}
	}
	if ip == "" {
		t.Skip("no interface here holds both a hardware and an IPv4 address")
	}
	if got := (network{}).HardwareAddress(ip); got != want {
		t.Fatalf("HardwareAddress(%s) = %q, want %q", ip, got, want)
	}
	if got := (network{}).HardwareAddress("192.0.2.77"); got != "" {
		t.Fatalf("an address no interface holds gave %q", got)
	}
}
