package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	intmanifest "spwn.sh/packages/project/internal/manifest"
	"spwn.sh/packages/project/lockfile"
)

func writeLockfile(t *testing.T, root string, l *lockfile.Lockfile) {
	t.Helper()
	if err := lockfile.Save(root, l); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}
}

// TestRulePackagesExist_localSkillFileForm verifies the local
// file-form skill path: spwn/packs/<name>.md counts as a valid
// bare-name local pack.
func TestRulePackagesExist_localSkillFileForm(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "spwn", "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "spwn", "plugins", "focus.md"), []byte("# focus"), 0o644); err != nil {
		t.Fatal(err)
	}

	ref := scaffoldAgent(t, root, "neo", `name: neo
plugins:
  - focus
  - missing-package
`)

	in := Input{
		Root: root,
		Manifest: &intmanifest.Manifest{
			Version: intmanifest.CurrentVersion,
			Name:    "t",
			Worlds:  map[string]intmanifest.World{"d": {Agents: []string{"neo"}, Workspaces: []string{"."}}},
		},
		AgentRefs: []AgentRef{ref},
	}

	issues := rulePacksExist(in)
	var missingFound bool
	var presentFound bool
	for _, iss := range issues {
		if strings.Contains(iss.Message, `"focus"`) {
			presentFound = true
		}
		if strings.Contains(iss.Message, `"missing-package"`) {
			missingFound = true
		}
	}
	if presentFound {
		t.Error("focus is on disk, should not error")
	}
	if !missingFound {
		t.Error("missing-package should error")
	}
}

func TestRulePackagesExist_registryUnsupported(t *testing.T) {
	root := t.TempDir()

	ref := scaffoldAgent(t, root, "neo", `name: neo
plugins:
  - "@jterrazz/focus"
`)
	in := Input{
		Root:      root,
		Manifest:  &intmanifest.Manifest{Version: intmanifest.CurrentVersion, Name: "t", Worlds: map[string]intmanifest.World{"d": {Agents: []string{"neo"}, Workspaces: []string{"."}}}},
		AgentRefs: []AgentRef{ref},
	}

	issues := rulePacksExist(in)
	if len(issues) == 0 || !strings.Contains(issues[0].Message, "remote registries are not yet supported") {
		t.Errorf("want registry-unsupported, got %+v", issues)
	}
}

func TestRuleLockfileConsistent_missingLockfileIsSilent(t *testing.T) {
	root := t.TempDir()
	ref := scaffoldAgent(t, root, "neo", `name: neo
plugins:
  - "@spwn/unix"
`)
	in := Input{
		Root:      root,
		Manifest:  &intmanifest.Manifest{Version: intmanifest.CurrentVersion, Name: "t", Worlds: map[string]intmanifest.World{"d": {Agents: []string{"neo"}, Workspaces: []string{"."}}}},
		AgentRefs: []AgentRef{ref},
	}
	if got := ruleLockfileConsistent(in); len(got) != 0 {
		t.Errorf("no lockfile → silent, got %+v", got)
	}
}

func TestRuleLockfileConsistent_driftFlagged(t *testing.T) {
	root := t.TempDir()
	ref := scaffoldAgent(t, root, "neo", `name: neo
plugins:
  - "@spwn/unix"
  - "@spwn/git"
  - "@spwn/mempalace"
`)
	// Lockfile only has @spwn/unix — git and mempalace are drift.
	l := lockfile.Empty()
	l.Add("@spwn/unix", lockfile.Entry{Version: "1", Source: lockfile.SourceBuiltin})
	writeLockfile(t, root, l)

	in := Input{
		Root:      root,
		Manifest:  &intmanifest.Manifest{Version: intmanifest.CurrentVersion, Name: "t", Worlds: map[string]intmanifest.World{"d": {Agents: []string{"neo"}, Workspaces: []string{"."}}}},
		AgentRefs: []AgentRef{ref},
	}
	issues := ruleLockfileConsistent(in)
	var sawGit, sawMempalace, sawUnix bool
	for _, iss := range issues {
		if strings.Contains(iss.Message, "@spwn/git") {
			sawGit = true
		}
		if strings.Contains(iss.Message, "@spwn/mempalace") {
			sawMempalace = true
		}
		if strings.Contains(iss.Message, "@spwn/unix") {
			sawUnix = true
		}
	}
	if !sawGit {
		t.Error("drift on @spwn/git not flagged")
	}
	if !sawMempalace {
		t.Error("drift on @spwn/mempalace not flagged")
	}
	if sawUnix {
		t.Error("@spwn/unix is in the lockfile, should not be flagged")
	}
}

func TestRuleLockfileConsistent_ignoresLocalRefs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "spwn", "plugins", "my-tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	ref := scaffoldAgent(t, root, "neo", `name: neo
plugins:
  - my-tool
`)
	// Empty lockfile — bare names should not produce errors.
	writeLockfile(t, root, lockfile.Empty())

	in := Input{
		Root:      root,
		Manifest:  &intmanifest.Manifest{Version: intmanifest.CurrentVersion, Name: "t", Worlds: map[string]intmanifest.World{"d": {Agents: []string{"neo"}, Workspaces: []string{"."}}}},
		AgentRefs: []AgentRef{ref},
	}
	if got := ruleLockfileConsistent(in); len(got) != 0 {
		t.Errorf("bare ref should not be in lockfile, got %+v", got)
	}
}
