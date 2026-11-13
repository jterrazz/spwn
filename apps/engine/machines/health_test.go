package machines

import (
	"net/netip"
	"testing"
	"time"
)

func TestACheckOnTheVMsOwnNameReachesItsAddressUnverified(t *testing.T) {
	addr := netip.MustParseAddr("192.168.64.5")
	for _, c := range []struct {
		check string
		want  CheckTarget
	}{
		{"https://infra:6443/readyz", CheckTarget{Dial: "192.168.64.5:6443", URL: "https://infra:6443/readyz", Name: "infra"}},
		{"https://INFRA/readyz", CheckTarget{Dial: "192.168.64.5:443", URL: "https://INFRA/readyz", Name: "INFRA"}},
		{"http://infra/health", CheckTarget{Dial: "192.168.64.5:80", URL: "http://infra/health", Name: "infra"}},
		{"infra:6443", CheckTarget{Dial: "192.168.64.5:6443", Name: "infra"}},
		{"https://api.example.com/health", CheckTarget{Dial: "api.example.com:443", URL: "https://api.example.com/health", Name: "api.example.com", Verify: true}},
		{"db.tail1234.ts.net:5432", CheckTarget{Dial: "db.tail1234.ts.net:5432", Name: "db.tail1234.ts.net", Verify: true}},
	} {
		check, ok := ParseCheck(c.check)
		if !ok {
			t.Fatalf("%q does not parse", c.check)
		}
		if got := check.Target("infra", addr); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.check, got, c.want)
		}
	}
	if check, _ := ParseCheck("infra:6443"); check.Target("infra", netip.Addr{}).Verify != true {
		t.Error("a VM with no address yet is reached by its name, verified")
	}
	for _, bad := range []string{"infra", "https://", "infra:0", "infra:70000", ":6443"} {
		if _, ok := ParseCheck(bad); ok {
			t.Errorf("%q parses", bad)
		}
	}
}

func TestACheckWaitsUntilItPassesThenFailsSinceItsFirstFailure(t *testing.T) {
	at := func(m int) time.Time { return time.Date(2026, 11, 7, 14, m, 0, 0, time.UTC) }
	s, change := Health(CheckStatus{}, false, at(0))
	if s.State != CheckWaiting || change != CheckSame {
		t.Fatalf("a check that never passed: %v %v, want waiting and no change", s.State, change)
	}
	s, _ = Health(s, true, at(1))
	s, change = Health(s, false, at(2))
	if s.State != CheckFailing || !s.Since.Equal(at(2)) || change != CheckFails {
		t.Fatalf("a check that passed then fails: %+v %v", s, change)
	}
	s, change = Health(s, false, at(3))
	if !s.Since.Equal(at(2)) || change != CheckSame {
		t.Errorf("a check still failing moved its since to %v (%v)", s.Since, change)
	}
	s, change = Health(s, true, at(4))
	if s.State != CheckPassing || change != CheckBack {
		t.Errorf("a failing check that passes: %v %v, want passing and back", s.State, change)
	}
	if !s.Due(at(5)) || s.Due(at(4).Add(59*time.Second)) {
		t.Error("a check is due each minute")
	}
}
