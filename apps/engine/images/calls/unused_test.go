package calls

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/images"
	imagesfake "github.com/jterrazz/studio-station/apps/engine/images/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

type noTemplates struct{}

func (noTemplates) Template(_ context.Context, name string) (images.Template, error) {
	return images.Template{}, &images.UnknownError{Image: name}
}
func (noTemplates) Names(context.Context) ([]string, error) { return nil, nil }

type madeFrom struct {
	contract.MachineStore
	records []machines.Record
}

func (m madeFrom) ListReadable(context.Context) ([]machines.Record, error) { return m.records, nil }

type unreadable struct{ *imagesfake.Marks }

func (unreadable) Marked(context.Context) ([]images.Making, error) {
	return nil, errors.New("the marks folder cannot be read")
}

func heldBuilder(t *testing.T, versions ...string) (*Builder, *imagesfake.Marks) {
	t.Helper()
	ctx := context.Background()
	held, marks := imagesfake.NewStore(), imagesfake.NewMarks()
	vms := madeFrom{records: []machines.Record{{Name: "api", Image: "base-ubuntu", Version: "2026.10.20"}}}
	for _, v := range versions {
		if err := held.Put(ctx, images.Record{Name: "base-ubuntu", Version: v}); err != nil {
			t.Fatal(err)
		}
	}
	s := &Builder{
		store: held, vms: vms, marks: marks, procs: running{10: true}, templates: noTemplates{},
		cat: catalog.Catalog{Images: []catalog.Image{{Name: "base-ubuntu"}}}, o: BuilderOptions{Mac: "macbook-pro"},
	}
	return s, marks
}

func TestDeleteRefusesEveryVersionTheRuleKeeps(t *testing.T) {
	s, marks := heldBuilder(t, "2026.11.01", "2026.10.20", "2026.10.01")
	if err := marks.Mark(context.Background(), images.Making{Image: "base-ubuntu", Version: "2026.11.04", Step: images.BuildPackages, Owner: &machines.Process{PID: 10}}); err != nil {
		t.Fatal(err)
	}
	for version, want := range map[string]*images.InUseError{
		"2026.11.04": {Image: "base-ubuntu", Version: "2026.11.04", Building: true},
		"2026.11.01": {Image: "base-ubuntu", Version: "2026.11.01", Newest: true},
		"2026.10.20": {Image: "base-ubuntu", Version: "2026.10.20", VMs: []string{"api"}},
		"2026.10.01": nil,
	} {
		err := s.CheckDelete(context.Background(), "base-ubuntu", version)
		var got *images.InUseError
		if want == nil {
			if err != nil {
				t.Errorf("%s: %v, want it deletable", version, err)
			}
			continue
		}
		if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %v, want %+v", version, err, want)
		}
	}
}

func TestTheInUseRefusalSaysWhy(t *testing.T) {
	for want, e := range map[string]images.InUseError{
		"image base-ubuntu 2026.11.04 is being built":                                    {Image: "base-ubuntu", Version: "2026.11.04", Building: true},
		"image base-ubuntu 2026.11.01 is the newest: the next machines are made from it": {Image: "base-ubuntu", Version: "2026.11.01", Newest: true},
		"image base-ubuntu 2026.10.20 has machines made from it: api, ci and linux-1":    {Image: "base-ubuntu", Version: "2026.10.20", VMs: []string{"api", "ci", "linux-1"}},
	} {
		if got := e.Error(); got != want {
			t.Errorf("%q, want %q", got, want)
		}
	}
}

func TestAVersionABuildHasRecordedReadsOnce(t *testing.T) {
	s, marks := heldBuilder(t, "2026.11.04", "2026.11.01")
	if err := marks.Mark(context.Background(), images.Making{Image: "base-ubuntu", Version: "2026.11.04", Step: images.BuildSeal, Owner: &machines.Process{PID: 10}}); err != nil {
		t.Fatal(err)
	}
	reports, err := s.ListOf(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	versions := make([]string, 0, len(reports))
	for _, r := range reports {
		versions = append(versions, r.Record.Version)
	}
	if !reflect.DeepEqual(versions, []string{"2026.11.04", "2026.11.01"}) || reports[0].Building != nil || !reports[0].Newest {
		t.Fatalf("listed %v, the first %+v; want each version once, the recorded one the newest", versions, reports[0])
	}
}

func TestMarksThatCannotBeReadAreNoBuild(t *testing.T) {
	s, marks := heldBuilder(t, "2026.11.01")
	s.marks = unreadable{marks}
	reports, err := s.ListOf(context.Background(), nil)
	if err != nil || len(reports) != 1 || reports[0].Building != nil {
		t.Fatalf("listed %+v, %v; want the version held and no build", reports, err)
	}
}
