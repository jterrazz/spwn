package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/teams/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/teams/contract"
	"github.com/jterrazz/studio-station/apps/engine/teams/contract/contracttest"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func open(inProject bool) contracttest.Opener {
	return func(t *testing.T, declared []contracttest.Declared, memberOf map[string]string) (contract.Teams, string) {
		home := t.TempDir()
		t.Setenv("SPWN_HOME", home)
		project, agents := "", filepath.Join(home, "agents")
		if inProject {
			project = t.TempDir()
			agents = filepath.Join(project, "spwn", "agents")
		}
		for _, d := range declared {
			body := "name: " + d.Name + "\n"
			if d.Description != "" {
				body += "description: " + d.Description + "\n"
			}
			write(t, filepath.Join(home, "teams", d.Slug+".yaml"), body)
		}
		for name, team := range memberOf {
			body := "description: an agent\n"
			if team != "" {
				body += "team: " + team + "\n"
			}
			write(t, filepath.Join(agents, name, "agent.yaml"), body)
		}
		return files.Teams{}, project
	}
}

func TestTeams(t *testing.T) { contracttest.Teams(t, open(true)) }

func TestTheUsersOwnAgentsAreThoseOfNoProject(t *testing.T) { contracttest.Teams(t, open(false)) }
