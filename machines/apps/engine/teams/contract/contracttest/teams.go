package contracttest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/teams"
	"github.com/jterrazz/studio-station/apps/engine/teams/contract"
)

type Declared struct {
	Slug, Name, Description, Color string
}

type Opener func(t *testing.T, declared []Declared, memberOf map[string]string) (contract.Teams, string)

func Teams(t *testing.T, open Opener) {
	// invariant: open gives Teams over a project of its own and that
	// project's root, the teams declared, and the agents made in the
	// project, each named with the team its agent.yaml gives it.
	ctx := context.Background()
	web := Declared{Slug: "web", Name: "Web Team", Description: "Ships the site", Color: "purple"}
	core := Declared{Slug: "core", Name: "Core"}
	memberOf := map[string]string{"neo": "web", "trinity": "web", "smith": "core", "oracle": "gone", "solo": ""}

	t.Run("no team declared lists none", func(t *testing.T) {
		m, project := open(t, nil, memberOf)
		got, err := m.List(ctx, project)
		if err != nil || len(got) != 0 {
			t.Fatalf("List = %+v, %v; want none", got, err)
		}
	})

	t.Run("the teams are listed by slug with the agents of the project that name them", func(t *testing.T) {
		m, project := open(t, []Declared{web, core}, memberOf)
		got, err := m.List(ctx, project)
		want := []teams.Team{
			{Slug: "core", Name: "Core", Members: []string{"smith"}},
			{Slug: "web", Name: "Web Team", Description: "Ships the site", Color: "purple", Members: []string{"neo", "trinity"}},
		}
		if err != nil || len(got) != len(want) {
			t.Fatalf("List = %+v, %v; want %+v", got, err, want)
		}
		for i := range want {
			if !same(got[i], want[i]) {
				t.Errorf("List[%d] = %+v; want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("a team no agent names has no member", func(t *testing.T) {
		m, project := open(t, []Declared{web}, nil)
		got, err := m.Team(ctx, project, "web")
		if err != nil || len(got.Members) != 0 || got.Name != "Web Team" {
			t.Fatalf("Team = %+v, %v; want web alone", got, err)
		}
	})

	t.Run("one team is read by its slug", func(t *testing.T) {
		m, project := open(t, []Declared{web, core}, memberOf)
		got, err := m.Team(ctx, project, "web")
		want := teams.Team{Slug: "web", Name: "Web Team", Description: "Ships the site", Color: "purple", Members: []string{"neo", "trinity"}}
		if err != nil || !same(got, want) {
			t.Fatalf("Team = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("a slug no team has, or no slug at all, is not found", func(t *testing.T) {
		m, project := open(t, []Declared{web}, memberOf)
		for _, slug := range []string{"gone", "", "../web", "web/x"} {
			if _, err := m.Team(ctx, project, slug); !errors.Is(err, teams.ErrNotFound) {
				t.Errorf("Team(%q) = %v; want ErrNotFound", slug, err)
			}
		}
	})
}

func same(a, b teams.Team) bool {
	return a.Slug == b.Slug && a.Name == b.Name && a.Description == b.Description && a.Color == b.Color && slices.Equal(a.Members, b.Members)
}
