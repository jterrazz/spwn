package machinefile_test

import (
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func TestAPackageOfTheImagePhaseIsBakedByEveryChildAndNeverAppliedInARunningMachine(t *testing.T) {
	// invariant: a package whose phase is image (a kernel) is installed by
	// the image phase alone, whatever bake list a child says, and every
	// other apply leaves it, so an image that lacks it names it in its
	// drift instead.
	kernel := "package: station.package/v1\nkind: app\nname: kernel\ndescription: the kernel a VM boots\nphase: image\nmethods:\n    - { os: ubuntu, script: { run: 'true' } }\n"
	base := machine("base", "{ os: ubuntu, version: '24.04' }", "[kernel, fd]") + "vm: { image: { from: upstream, bake: [kernel, fd] } }\n"
	child := machine("child", "{ os: ubuntu, version: '24.04' }", "[]") + "extends: [base]\nvm: { image: { bake: [fd] } }\n"
	lib := library(t, map[string]string{"machinefiles/child.machine.yaml": child}, map[string]string{"machinefiles/base.machine.yaml": base, "catalog/system-utilities/kernel.yaml": kernel})
	r, err := machinefile.Resolve(lib, "child", machinefile.Machine{}, machinefile.Lock{})
	if err != nil {
		t.Fatal(err)
	}
	absent := map[string]machinefile.Seen{}
	for _, s := range r.Steps {
		absent[s.Package] = machinefile.Seen{Condition: machinefile.Absent}
		if s.Package == "kernel" && !s.Baked {
			t.Error("the child bakes no kernel: what the kernel requires would be left out of its image phase")
		}
	}
	actions := func(phase catalog.Phase) map[string]machinefile.Action {
		out := map[string]machinefile.Action{}
		for _, l := range machinefile.Judge(r, lib.Catalog, absent, nil, true, phase).Lines {
			out[l.Step.Package] = l.Action
		}
		return out
	}
	if got := actions(catalog.PhaseApply); got["kernel"] != machinefile.Leave || got["fd"] != machinefile.Install {
		t.Errorf("an apply would %v; want the kernel left and fd installed", got)
	}
	if got := actions(catalog.PhaseImage); got["kernel"] != machinefile.Install || got["fd"] != machinefile.Install {
		t.Errorf("the image phase would %v; want the kernel and fd installed", got)
	}
	tpl, err := machinefile.TemplateOf(lib, "child", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"fd", "kernel"}; !slices.Equal(tpl.Bake, want) {
		t.Errorf("the child bakes %v; want %v", tpl.Bake, want)
	}
	old := []byte("lock: station.lock/v1\nmachinefile: child\napps:\n  - { name: fd, method: shell }\n")
	if got, want := machinefile.ImageDrift(old, tpl), []string{"kernel is not baked into the image"}; !slices.Equal(got, want) {
		t.Errorf("an image built without the kernel drifts by %v; want %v", got, want)
	}
}
