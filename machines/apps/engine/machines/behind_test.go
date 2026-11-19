package machines

import "testing"

func TestAPreReleaseOrdersBetweenTheReleasesAroundIt(t *testing.T) {
	order := []string{"v3.1.0", "v4.0.0-alpha.1", "v4.0.0-alpha.2", "v4.0.0-alpha.10", "v4.0.0-beta", "v4.0.0-rc.2", "v4.0.0", "v4.0.1"}
	for i, lo := range order {
		for j, hi := range order {
			if got := Newer(hi, lo); got != (j > i) {
				t.Errorf("Newer(%q, %q) = %v; want %v", hi, lo, got, j > i)
			}
		}
	}
}

func TestAVersionThatIsNotAReleaseIsNeverNewer(t *testing.T) {
	for _, v := range []string{"dev", "", "4.0.0", "v4.0", "v4.0.0-"} {
		if Newer(v, "v3.0.0") || Newer("v3.0.0", v) {
			t.Errorf("newer on %q ordered it against v3.0.0", v)
		}
	}
}

func TestAMacOnAPreReleaseIsBehindARelease(t *testing.T) {
	far := MacView{StationVersion: "v4.0.0-alpha.1"}
	if fix, ok := Behind(far, "v4.0.0"); !ok || fix != "update station there (scripts/install.sh)" {
		t.Errorf("Behind = %q, %v; want the far Mac asked to update", fix, ok)
	}
	if fix, ok := Behind(MacView{StationVersion: "v4.0.0"}, "v4.0.0-alpha.1"); !ok || fix != "update station here (scripts/install.sh)" {
		t.Errorf("Behind = %q, %v; want this Mac asked to update", fix, ok)
	}
}

func TestAFleetOnOneAlphaDiffersNowhere(t *testing.T) {
	const alpha = "v0.1.0-alpha.1"
	if fix, ok := Behind(MacView{StationVersion: alpha}, alpha); ok {
		t.Errorf("Behind on two Macs at %s = %q, %v; want no difference", alpha, fix, ok)
	}
	if fix, ok := Behind(MacView{StationVersion: alpha}, "v0.1.0"); !ok || fix != "update station there (scripts/install.sh)" {
		t.Errorf("Behind = %q, %v; want the alpha asked to update", fix, ok)
	}
}
