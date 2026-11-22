package calls

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	"github.com/jterrazz/studio-station/apps/engine/disk/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

func TestTheLastKnownSizesKeepOnlyTheRootsOfTheWalkTheUnansweredAtTheirLastSize(t *testing.T) {
	m := keptFiles{onMachine: onMachine{Usage: fake.NewUsage(nil, 1<<40, 1<<39)}, files: map[string][]byte{}}
	s := NewSurveyor(m, nil, SurveyorOptions{ThisMachine: catalog.Mac, Expand: disk.Expander{Home: "/h", Roots: setup.Roots{Cache: "/h/.cache"}}})
	s.saveKnown([]string{"/h/vms/accept-1", "/h/vms/api", "/h/logs"}, []int64{5, 7, 1})
	s.saveKnown([]string{"/h/vms/api", "/h/logs"}, []int64{9, unmeasured})
	var k knownFile
	if err := json.Unmarshal(m.files["/h/.cache/station/sizes.json"], &k); err != nil {
		t.Fatal(err)
	}
	if got := slices.Sorted(maps.Keys(k.Sizes)); !slices.Equal(got, []string{"/h/logs", "/h/vms/api"}) {
		t.Errorf("the sizes kept are of %v, want the walk's roots alone", got)
	}
	if k.Sizes["/h/vms/api"].Bytes != 9 || k.Sizes["/h/logs"].Bytes != 1 {
		t.Errorf("sizes = %+v, want api at its new size and logs at its last", k.Sizes)
	}
}
