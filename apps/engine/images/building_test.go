package images

import "testing"

func TestBuildingAtCountsTheStepsFromOne(t *testing.T) {
	for step, want := range map[BuildStep]string{
		BuildBaseSystem: "Step 1 of 4",
		BuildPackages:   "Step 2 of 4",
		BuildSettings:   "Step 3 of 4",
		BuildSeal:       "Step 4 of 4",
	} {
		if got := BuildingAt(step).Sentence; got != want {
			t.Errorf("%s: %q, want %q", step, got, want)
		}
	}
}

func TestDeletableReadsTheOneRuleOfUnused(t *testing.T) {
	building := BuildingAt(BuildPackages)
	for name, r := range map[string]Report{
		"the newest":                {Newest: true},
		"older and used":            {VMs: []string{"api"}},
		"older and unused":          {},
		"being built, nothing yet":  {Building: building},
		"newest and used and built": {Newest: true, VMs: []string{"api"}},
	} {
		if got := r.Judged().Deletable; got != r.Unused() {
			t.Errorf("%s: Deletable %v, Unused %v", name, got, r.Unused())
		}
	}
	if !(Report{}).Judged().Deletable {
		t.Error("an older version no machine uses is deletable")
	}
	if (Report{Building: building}).Judged().Deletable {
		t.Error("a version being built is never deletable")
	}
}
