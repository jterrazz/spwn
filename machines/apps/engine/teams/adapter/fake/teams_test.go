package fake_test

import (
	"encoding/json"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/teams/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/teams/contract"
	"github.com/jterrazz/studio-station/apps/engine/teams/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestTeams(t *testing.T) {
	contracttest.Teams(t, func(t *testing.T, declared []contracttest.Declared, memberOf map[string]string) (contract.Teams, string) {
		project := t.TempDir()
		teams := make([]map[string]string, 0, len(declared))
		minds := make([]map[string]string, 0, len(memberOf))
		for _, d := range declared {
			teams = append(teams, map[string]string{"Slug": d.Slug, "Name": d.Name, "Description": d.Description, "Color": d.Color})
		}
		for name, team := range memberOf {
			minds = append(minds, map[string]string{"Name": name, "Team": team})
		}
		stores := map[string]any{"teams": teams, "agents": map[string]any{project: minds}}
		raw, err := json.Marshal(map[string]any{"macs": map[string]any{"this-mac": map[string]any{"stores": stores}}})
		if err != nil {
			t.Fatal(err)
		}
		w := &world.World{}
		if err := json.Unmarshal(raw, w); err != nil {
			t.Fatal(err)
		}
		return fake.Of(w, "this-mac", unkept{}), project
	})
}

// value: the layout of a root the world keeps no project at, as the
// project family's fake reads it: act runs. An adapter's test imports no
// other adapter.
type unkept struct{}

func (unkept) Within(_ string, act func() error) error { return act() }
