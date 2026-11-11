package skills

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const command = "skills"

type Options struct {
	Run    local.Runner
	FS     local.FS
	Net    local.Net
	Home   string
	APIURL string
	Token  string
}

type Skills struct{ o Options }

func New(o Options) *Skills { return &Skills{o: o} }

func (s *Skills) Method() catalog.Method { return catalog.MethodSkill }

func (s *Skills) Manager() contract.Manager {
	return contract.Manager{Name: command, App: command}
}

func (s *Skills) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true, UpdateAll: true}
}

func (s *Skills) Ready(context.Context) error {
	if _, ok := s.o.Run.LookPath(command); !ok {
		return contract.ErrMissing
	}
	return nil
}

func (s *Skills) Inventory(context.Context) (map[string]string, error) {
	// invariant: the lock is the list, read with or without the CLI: a skill it
	// records is installed, and it records no version.
	out := map[string]string{}
	for name := range s.lock() {
		out[name] = ""
	}
	return out, nil
}

// outside: mirrors ~/.agents/.skill-lock.json (schema version 3), the skills CLI's
// own record; only the fields the currency check reads are declared.
type lockEntry struct {
	Source     string `json:"source"`
	SourceType string `json:"sourceType"`
	SkillPath  string `json:"skillPath"`
	FolderHash string `json:"skillFolderHash"`
}

func (s *Skills) lock() map[string]lockEntry {
	var doc struct {
		Skills map[string]lockEntry `json:"skills"`
	}
	data, err := s.o.FS.ReadFile(path.Join(s.o.Home, ".agents", ".skill-lock.json"))
	if err != nil || json.Unmarshal(data, &doc) != nil {
		return map[string]lockEntry{}
	}
	return doc.Skills
}

func (s *Skills) Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error) {
	// invariant: the verdict compares git tree SHAs, the oracle the skills CLI updates
	// by; an inconclusive check (network, non-200, a folder gone upstream) leaves the
	// skill's currency unknown, never behind.
	lock := s.lock()
	var mu sync.Mutex
	folders := map[string]map[string]string{}
	listingOf := func(source, parent string) map[string]string {
		mu.Lock()
		defer mu.Unlock()
		key := source + "\x00" + parent
		if shas, ok := folders[key]; ok {
			return shas
		}
		shas := s.fetchFolder(ctx, source, parent)
		folders[key] = shas
		return shas
	}
	out := map[string]apps.Currency{}
	var wg sync.WaitGroup
	for _, key := range keys {
		entry, ok := lock[key]
		if !ok {
			continue
		}
		wg.Go(func() {
			verdict := apps.Unknown
			folder := path.Dir(entry.SkillPath)
			if entry.SourceType == "github" && entry.FolderHash != "" && folder != "." && catalog.CheckRepo(entry.Source) == nil && plain(folder) {
				parent := path.Dir(folder)
				if parent == "." {
					parent = ""
				}
				if upstream, ok := listingOf(entry.Source, parent)[path.Base(folder)]; ok {
					verdict = apps.Current
					if upstream != entry.FolderHash {
						verdict = apps.Behind
					}
				}
			}
			mu.Lock()
			out[key] = apps.Currency{Verdict: verdict}
			mu.Unlock()
		})
	}
	wg.Wait()
	return out, nil
}

// invariant: the lock is read from disk, so what it says is checked before
// it joins an API route: its source as any repo, its folder's segments as
// plain names of a path.
func plain(folder string) bool {
	for seg := range strings.SplitSeq(folder, "/") {
		if seg == "" || seg == "." || seg == ".." || strings.ContainsAny(seg, "?#% \\") {
			return false
		}
	}
	return true
}

func (s *Skills) fetchFolder(ctx context.Context, source, parent string) map[string]string {
	header := map[string]string{"Accept": "application/vnd.github+json"}
	// outside: unauthenticated api.github.com allows 60 requests an hour, a token
	// 5000; a rate-limited answer is a non-200, so the currency stays unknown.
	if s.o.Token != "" {
		header["Authorization"] = "Bearer " + s.o.Token
	}
	url := strings.TrimSuffix(s.o.APIURL, "/") + "/repos/" + source + "/contents/" + parent
	status, body, err := s.o.Net.Get(ctx, url, header)
	if err != nil || status != 200 {
		return nil
	}
	var entries []struct {
		Name string `json:"name"`
		SHA  string `json:"sha"`
	}
	if json.Unmarshal(body, &entries) != nil {
		return nil
	}
	shas := make(map[string]string, len(entries))
	for _, e := range entries {
		shas[e.Name] = e.SHA
	}
	return shas
}

func (s *Skills) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return s.each(ctx, installs, logs, add)
}

func (s *Skills) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return s.each(ctx, installs, logs, add)
}

func (s *Skills) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return s.each(ctx, installs, logs, func(v catalog.SkillInstall) local.Cmd {
		return local.Command(command, "remove", "-g", "-y", "--", v.Skill)
	})
}

func add(v catalog.SkillInstall) local.Cmd {
	return local.Command(command, slices.Concat([]string{"add", "-g", "-y", "--skill", v.Skill, "--agent"}, catalog.SkillAgents, []string{"--", v.Repo})...)
}

// invariant: the skills CLI takes an argument opening with `-` as an
// option, so a repo and a skill's name are checked by the catalog's rule
// before an argv carries them, and every argv ends its options with `--`
// before the positional one.
func checked(v catalog.SkillInstall) error {
	if err := catalog.CheckRepo(v.Repo); err != nil {
		return err
	}
	return catalog.CheckSkill(v.Skill)
}

func (s *Skills) each(ctx context.Context, installs []catalog.Install, logs contract.Logs, argv func(catalog.SkillInstall) local.Cmd) error {
	for _, i := range installs {
		v, ok := i.(catalog.SkillInstall)
		if !ok {
			return errors.New(i.Key() + " is not a skill")
		}
		if err := checked(v); err != nil {
			return err
		}
		c := argv(v)
		c.Logs = logs
		if err := local.Attach(ctx, s.o.Run, c); err != nil {
			return err
		}
	}
	return nil
}
