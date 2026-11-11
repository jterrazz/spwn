package githubrelease_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jterrazz/studio-station/src/apps/adapter/githubrelease"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
)

func bare(n *localtest.Net) {
	body := []byte("echo pwned\n")
	serve(n, served{tag: "v1", assets: map[string][]byte{"tool-linux-arm64": body}, digests: map[string]string{"tool-linux-arm64": "sha256:" + sum(body)}})
}

func TestAHostileNameNeverBuildsAPath(t *testing.T) {
	// invariant: a command or a repo that is no name by the catalog's rule
	// is refused before anything is fetched, written or removed.
	hostile := make([]catalog.GitHubReleaseInstall, 0, len(contracttest.HostileCommands())+len(contracttest.HostileRepos()))
	for _, c := range contracttest.HostileCommands() {
		hostile = append(hostile, catalog.GitHubReleaseInstall{Repo: repo, Asset: "tool-{os}-{arch}", Commands: []string{c}})
	}
	for _, r := range contracttest.HostileRepos() {
		hostile = append(hostile, catalog.GitHubReleaseInstall{Repo: r, Asset: "tool-{os}-{arch}", Commands: []string{"tool"}})
	}
	for _, in := range hostile {
		files, n := localtest.NewFS(), localtest.NewNet()
		bare(n)
		r := releases(files, n, "arm64")
		for verb, err := range map[string]error{
			"install": r.Install(context.Background(), []catalog.Install{in}, nil),
			"update":  r.Update(context.Background(), []catalog.Install{in}, nil),
		} {
			var name *catalog.NameError
			if !errors.As(err, &name) || len(n.Requests) > 0 || len(files.Files) > 0 {
				t.Errorf("%s of %q %q gave %v, fetched %v and wrote %v; want it refused, nothing fetched nor written", verb, in.Repo, in.Commands, err, n.Requests, files.Files)
			}
		}
	}
	for _, bad := range contracttest.HostileRepos() {
		files := localtest.NewFS()
		_ = files.WriteFile("/tmp/x.json", []byte(`{"repo":"a/b"}`), 0o644)
		err := releases(files, localtest.NewNet(), "arm64").Remove(context.Background(), []catalog.Install{catalog.GitHubReleaseInstall{Repo: bad}}, nil)
		var name *catalog.NameError
		if _, read := files.ReadFile("/tmp/x.json"); !errors.As(err, &name) || read != nil {
			t.Errorf("removal of %q gave %v; want it refused, nothing removed", bad, err)
		}
	}
}

func TestAProvidedPathNeverLandsOutsideTheSearchPath(t *testing.T) {
	files, n := localtest.NewFS(), localtest.NewNet()
	bare(n)

	err := releases(files, n, "arm64").Install(context.Background(), []catalog.Install{catalog.GitHubReleaseInstall{Repo: repo, Asset: "tool-{os}-{arch}", Commands: []string{"../../.profile"}}}, nil)

	if _, read := files.ReadFile("/home/u/.profile"); err == nil || read == nil {
		t.Errorf("install gave %v and wrote ~/.profile (%v); want it refused, nothing written", err, read)
	}
}

func record(t *testing.T, files *localtest.FS, at string, rec map[string]any) {
	t.Helper()
	b, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := files.WriteFile(store+"/"+at, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestARecordNamingAPathOutsideIsNeverFollowed(t *testing.T) {
	// invariant: a record is read back from the disk, so a command or a
	// repo it names is checked as an install's is: a record that names a
	// path outside its folders removes nothing and holds nothing.
	victim := []byte("ssh-ed25519 AAAA owner@mac\n")
	for name, rec := range map[string]map[string]any{
		"a command outside the search path": {"repo": repo, "tag": "v1", "files": map[string]string{"../../.ssh/authorized_keys": sum(victim)}},
		"an install under way outside it":   {"repo": repo, "pending_tag": "v1", "pending": map[string]string{"../../.ssh/authorized_keys": sum(victim)}},
		"another repo than its place":       {"repo": "acme/other", "tag": "v1", "files": map[string]string{"tool": sum(victim)}},
		"a repo outside the store":          {"repo": "acme/../../../../.ssh/authorized_keys", "tag": "v1", "files": map[string]string{"tool": sum(victim)}},
	} {
		files := localtest.NewFS()
		_ = files.WriteFile("/home/u/.ssh/authorized_keys", victim, 0o600)
		_ = files.WriteFile(bin+"/tool", victim, 0o755)
		record(t, files, repo+".json", rec)
		r := releases(files, localtest.NewNet(), "arm64")

		err := r.Remove(context.Background(), []catalog.Install{tool("")}, nil)

		var bad *githubrelease.RecordError
		if !errors.As(err, &bad) {
			t.Errorf("%s: removal gave %v; want the record refused", name, err)
		}
		for _, kept := range []string{"/home/u/.ssh/authorized_keys", bin + "/tool", store + "/" + repo + ".json"} {
			if _, read := files.ReadFile(kept); read != nil {
				t.Errorf("%s: %s removed", name, kept)
			}
		}
		if held, err := r.Inventory(context.Background()); err != nil || len(held) > 0 {
			t.Errorf("%s: the inventory holds %v (%v); want nothing", name, held, err)
		}
	}
}

func TestARecordThatDoesNotReadLeavesTheOthersHeld(t *testing.T) {
	files, n := localtest.NewFS(), localtest.NewNet()
	serve(n, toolAt(t, "v2.0.0"))
	r := releases(files, n, "arm64")
	if err := r.Install(context.Background(), []catalog.Install{tool("")}, nil); err != nil {
		t.Fatal(err)
	}
	_ = files.WriteFile(store+"/acme/broken.json", []byte("{not json"), 0o644)

	held, err := r.Inventory(context.Background())

	if err != nil || held[repo] != "v2.0.0" || len(held) != 1 {
		t.Errorf("the inventory holds %v (%v); want tool alone, the broken record skipped", held, err)
	}
	var bad *githubrelease.RecordError
	if err := r.Remove(context.Background(), []catalog.Install{catalog.GitHubReleaseInstall{Repo: "acme/broken"}}, nil); !errors.As(err, &bad) {
		t.Errorf("removal of the broken record's release gave %v; want the record named", err)
	}
}
