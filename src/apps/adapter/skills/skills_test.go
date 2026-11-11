package skills_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/adapter/skills"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

type entry struct {
	Source     string `json:"source"`
	SourceType string `json:"sourceType"`
	SkillPath  string `json:"skillPath"`
	FolderHash string `json:"skillFolderHash"`
}

// value: the skills CLI as these tests play it: it keeps its lock under the
// test's own HOME, as the real one keeps ~/.agents/.skill-lock.json.
type agents struct {
	t     *testing.T
	fs    local.FS
	home  string
	fails bool
}

func newAgents(t *testing.T) *agents {
	home := t.TempDir()
	return &agents{t: t, fs: local.New([]string{"HOME=" + home}, nil, io.Discard, io.Discard).FS, home: home}
}

func (a *agents) write(lock map[string]entry) {
	body, err := json.Marshal(map[string]any{"version": 3, "skills": lock})
	if err == nil {
		err = a.fs.MkdirAll(path.Join(a.home, ".agents"), 0o755)
	}
	if err == nil {
		err = a.fs.WriteFile(path.Join(a.home, ".agents", ".skill-lock.json"), body, 0o644)
	}
	if err != nil {
		a.t.Fatal(err)
	}
}

func (a *agents) read() map[string]entry {
	var doc struct {
		Skills map[string]entry `json:"skills"`
	}
	data, _ := a.fs.ReadFile(path.Join(a.home, ".agents", ".skill-lock.json"))
	_ = json.Unmarshal(data, &doc)
	if doc.Skills == nil {
		doc.Skills = map[string]entry{}
	}
	return doc.Skills
}

func (a *agents) play(c local.Cmd) (local.Output, error) {
	if a.fails {
		return local.Output{ExitCode: 1}, nil
	}
	lock := a.read()
	switch {
	case len(c.Args) > 5 && c.Args[0] == "add":
		lock[c.Args[5]] = entry{Source: c.Args[1], SourceType: "github"}
	case len(c.Args) == 4 && c.Args[0] == "remove":
		delete(lock, c.Args[3])
	default:
		return local.Output{ExitCode: 97}, nil
	}
	a.write(lock)
	return local.Output{}, nil
}

type github map[string]string

func (g github) Dial(context.Context, string, int, time.Duration) bool { return false }

func (g github) Broadcast(context.Context, int, []byte) error { return nil }

func (g github) HardwareAddress(string) string { return "" }

func (g github) Get(_ context.Context, url string, _ map[string]string) (status int, body []byte, err error) {
	if g == nil {
		return 0, nil, errors.New("connection refused")
	}
	listing := make([]map[string]string, 0, len(g))
	for name, sha := range g {
		listing = append(listing, map[string]string{"name": name, "sha": sha})
	}
	if !strings.HasSuffix(url, "/repos/anthropics/skills/contents/skills") {
		return 404, nil, nil
	}
	body, err = json.Marshal(listing)
	return 200, body, err
}

func (a *agents) skills(net local.Net, names ...string) *skills.Skills {
	return skills.New(skills.Options{Run: local.Played(a.play, names...), FS: a.fs, Net: net, Home: a.home, APIURL: "http://github.test"})
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, catalog.SkillInstall{Repo: "anthropics/skills", Skill: "frontend-design"}, func(t *testing.T, seed contracttest.Seed) contract.Installer {
		a := newAgents(t)
		a.fails = seed.Fails
		lock := map[string]entry{}
		for k := range seed.Installed {
			lock[k] = entry{Source: "anthropics/skills", SourceType: "github"}
		}
		a.write(lock)
		names := []string{"skills"}
		if seed.Absent {
			names = nil
		}
		return a.skills(nil, names...)
	})
}

func held(t *testing.T) *agents {
	a := newAgents(t)
	a.write(map[string]entry{"frontend-design": {Source: "anthropics/skills", SourceType: "github", SkillPath: "skills/frontend-design/SKILL.md", FolderHash: "aaa"}})
	return a
}

func currency(t *testing.T, net local.Net) apps.Verdict {
	got, err := held(t).skills(net).Outdated(context.Background(), []string{"frontend-design"})
	if err != nil {
		t.Fatal(err)
	}
	return got["frontend-design"].Verdict
}

func TestASkillWhoseUpstreamFolderMovedOnIsBehind(t *testing.T) {
	if got := currency(t, github{"frontend-design": "bbb"}); got != apps.Behind {
		t.Errorf("currency %s", got)
	}
}

func TestASkillMatchingItsUpstreamFolderIsCurrent(t *testing.T) {
	if got := currency(t, github{"frontend-design": "aaa"}); got != apps.Current {
		t.Errorf("currency %s", got)
	}
}

func TestAnUnreachableSourceLeavesTheCurrencyUnknownNeverBehind(t *testing.T) {
	if got := currency(t, github(nil)); got != apps.Unknown {
		t.Errorf("currency %s", got)
	}
}

func TestTheLockListsASkillEvenWithoutTheCLI(t *testing.T) {
	got, err := held(t).skills(nil).Inventory(context.Background())
	if _, in := got["frontend-design"]; err != nil || !in {
		t.Errorf("inventory %v %v", got, err)
	}
}
