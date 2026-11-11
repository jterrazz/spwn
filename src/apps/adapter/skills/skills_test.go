package skills_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path"
	"slices"
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
	case len(c.Args) > 5 && c.Args[0] == "add" && c.Args[len(c.Args)-2] == "--":
		lock[c.Args[4]] = entry{Source: c.Args[len(c.Args)-1], SourceType: "github"}
	case len(c.Args) == 5 && c.Args[0] == "remove" && c.Args[3] == "--":
		delete(lock, c.Args[4])
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

func TestAHostileNameNeverReachesTheSkillsCLI(t *testing.T) {
	// invariant: a repo or a skill that is no name by the catalog's rule is
	// refused before the CLI runs; every argv ends its options before the
	// repo, or the skill a removal names.
	for _, in := range []catalog.SkillInstall{
		{Repo: "--help", Skill: "x"}, {Repo: "-a/b", Skill: "x"}, {Repo: "https://evil.example/x.git", Skill: "x"}, {Repo: "../x", Skill: "x"},
		{Repo: "anthropics/skills", Skill: "-x"}, {Repo: "anthropics/skills", Skill: "--agent"}, {Repo: "anthropics/skills", Skill: "a b"}, {Repo: "anthropics/skills", Skill: ""},
	} {
		for verb, run := range map[string]func(*skills.Skills) error{
			"install": func(s *skills.Skills) error { return s.Install(context.Background(), []catalog.Install{in}, nil) },
			"update":  func(s *skills.Skills) error { return s.Update(context.Background(), []catalog.Install{in}, nil) },
			"remove":  func(s *skills.Skills) error { return s.Remove(context.Background(), []catalog.Install{in}, nil) },
		} {
			var calls []string
			a := newAgents(t)
			s := skills.New(skills.Options{Run: local.Played(func(c local.Cmd) (local.Output, error) {
				calls = append(calls, strings.Join(c.Args, " "))
				return local.Output{}, nil
			}, "skills"), FS: a.fs, Home: a.home})
			var name *catalog.NameError
			if err := run(s); !errors.As(err, &name) || len(calls) > 0 {
				t.Errorf("%s of %+v gave %v and ran %q; want it refused, nothing run", verb, in, err, calls)
			}
		}
	}
}

func TestTheCLIEndsItsOptionsBeforeTheRepoAndTheName(t *testing.T) {
	var calls []string
	a := newAgents(t)
	s := skills.New(skills.Options{Run: local.Played(func(c local.Cmd) (local.Output, error) {
		calls = append(calls, strings.Join(c.Args, " "))
		return local.Output{}, nil
	}, "skills"), FS: a.fs, Home: a.home})
	in := []catalog.Install{catalog.SkillInstall{Repo: "anthropics/skills", Skill: "frontend-design"}}

	if err := s.Install(context.Background(), in, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove(context.Background(), in, nil); err != nil {
		t.Fatal(err)
	}
	want := []string{"add -g -y --skill frontend-design --agent " + strings.Join(catalog.SkillAgents, " ") + " -- anthropics/skills", "remove -g -y -- frontend-design"}
	if !slices.Equal(calls, want) {
		t.Errorf("ran %q; want %q", calls, want)
	}
}

type spy struct {
	github
	urls []string
}

func (g *spy) Get(ctx context.Context, url string, h map[string]string) (status int, body []byte, err error) {
	g.urls = append(g.urls, url)
	return g.github.Get(ctx, url, h)
}

func TestALockSourceOrFolderThatIsNoNameNeverJoinsTheAPIRoute(t *testing.T) {
	// invariant: the lock file is read from disk: a source or a folder in it
	// that is no plain name is never put in a request.
	for name, e := range map[string]entry{
		"a source of another route": {Source: "../../../evil/repo", SourceType: "github", SkillPath: "skills/x/SKILL.md", FolderHash: "aaa"},
		"a source with a query":     {Source: "anthropics/skills?x=1", SourceType: "github", SkillPath: "skills/x/SKILL.md", FolderHash: "aaa"},
		"a folder going up":         {Source: "anthropics/skills", SourceType: "github", SkillPath: "../../x/SKILL.md", FolderHash: "aaa"},
	} {
		a := newAgents(t)
		a.write(map[string]entry{"x": e})
		net := &spy{github: github{"x": "aaa"}}

		got, err := a.skills(net).Outdated(context.Background(), []string{"x"})

		if err != nil || got["x"].Verdict != apps.Unknown || len(net.urls) > 0 {
			t.Errorf("%s: read %v (%v) and requested %q; want unknown, nothing requested", name, got, err, net.urls)
		}
	}
}
