package calls

import (
	"errors"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func TestAnUpgradeIsJudgedAtTheNewestCommitItMayTake(t *testing.T) {
	// invariant: a plan off a terminal that asks something is refused
	// before the source moves; a clone whose commit does not resolve is
	// pulled first and judged again there; a newest commit that does not
	// resolve or is refused is put back; a source that cannot move leaves
	// the run where the clone stands.
	unresolved, refused := errors.New("does not resolve"), errors.New("asks a password")
	for _, c := range []struct {
		name                   string
		stood, newest          error
		moved                  bool
		refuses                []string
		wantPlan, wantErr      string
		pulled, back, unpulled bool
	}{
		{name: "refused where the clone stands", refuses: []string{"old"}, wantErr: "asks a password"},
		{name: "taken at the newest", moved: true, wantPlan: "new", pulled: true},
		{name: "a newest that does not resolve is put back", newest: unresolved, moved: true, wantPlan: "old", pulled: true, back: true, unpulled: true},
		{name: "a newest refused is put back", moved: true, refuses: []string{"new"}, wantErr: "asks a password", pulled: true, back: true},
		{name: "unresolved where it stands, fixed upstream", stood: unresolved, moved: true, wantPlan: "new", pulled: true},
		{name: "unresolved where it stands, refused at the newest", stood: unresolved, moved: true, refuses: []string{"new"}, wantErr: "asks a password", pulled: true, back: true},
		{name: "unresolved where it stands and at the newest", stood: unresolved, newest: unresolved, moved: true, wantErr: "does not resolve", pulled: true},
		{name: "unresolved where it stands, offline", stood: unresolved, wantErr: "does not resolve", pulled: true, unpulled: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			pulled, back := false, false
			judge := func() (machinefile.Plan, error) {
				if !pulled {
					return machinefile.Plan{Name: "old"}, c.stood
				}
				return machinefile.Plan{Name: "new"}, c.newest
			}
			pull := func() (func(), bool, string) {
				pulled = true
				said := ""
				if !c.moved {
					said = "its url does not answer"
				}
				return func() { back = true }, c.moved, said
			}
			refuse := func(p machinefile.Plan) error {
				for _, r := range c.refuses {
					if r == p.Name {
						return refused
					}
				}
				return nil
			}
			plan, unpulled, err := atNewest(judge, pull, refuse)
			switch {
			case err != nil && err.Error() != c.wantErr:
				t.Errorf("err %q; want %q", err, c.wantErr)
			case err == nil && c.wantErr != "":
				t.Errorf("no error; want %q", c.wantErr)
			}
			if err == nil && plan.Name != c.wantPlan {
				t.Errorf("plan at %q; want %q", plan.Name, c.wantPlan)
			}
			if pulled != c.pulled || back != c.back {
				t.Errorf("pulled %v, put back %v; want %v, %v", pulled, back, c.pulled, c.back)
			}
			if (unpulled != "") != c.unpulled || (strings.Contains(unpulled, "does not resolve") && c.stood != nil) {
				t.Errorf("unpulled %q; want said %v", unpulled, c.unpulled)
			}
		})
	}
}
