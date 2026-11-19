package ports

import "testing"

// invariant: a range of the Mac's ports never covers a Funnel port: a
// private share on one would be on the internet whenever it is funneled.
func TestARangeOverAFunnelPortIsReadAsTheDefault(t *testing.T) {
	for _, r := range []string{"400-500", "8000-9000", "9000-11000"} {
		if p := PolicyOf(&PortsGroup{Range: r}); p.Low != 40000 || p.High != 40999 {
			t.Errorf("range %s = %d-%d, want the default 40000-40999", r, p.Low, p.High)
		}
	}
	if p := PolicyOf(&PortsGroup{Range: "20000-20999"}); p.Low != 20000 || p.High != 20999 {
		t.Errorf("range 20000-20999 = %d-%d, kept as said", p.Low, p.High)
	}
}
