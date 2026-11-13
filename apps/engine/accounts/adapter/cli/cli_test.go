package cli_test

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/cli"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

const sentinel = "SENTINELgho_0123456789abcdefghijklmnopqrstuvwxyzAB"

type files struct{ m fstest.MapFS }

func (f files) ReadFile(p string) ([]byte, error) { return f.m.ReadFile(strings.TrimPrefix(p, "/")) }

func provider(id string) catalog.Provider {
	for _, p := range catalog.Providers {
		if p.ID == id {
			return p
		}
	}
	return catalog.Provider{ID: id}
}

// value: each tool printing what it prints on a signed-in Mac, with the
// sentinel token wherever a tool may leak one.
func printed() map[string]local.Output {
	return map[string]local.Output{
		"gh":                            {Stdout: "github.com\n  Logged in to github.com account jterrazz (keyring)\n  - Git operations protocol: ssh\n  - Token: " + sentinel + "\n  - Token scopes: 'gist', '" + sentinel + "'\n", Stderr: sentinel},
		"npm":                           {Stdout: sentinel + "\n", Stderr: "notice " + sentinel},
		"claude":                        {Stdout: `{"loggedIn":true,"email":"` + sentinel + `","subscriptionType":"max","oauthToken":"` + sentinel + `","authMethod":"claude.ai"}`},
		"wrangler":                      {Stdout: "You are logged in with an OAuth Token, associated with the email " + sentinel + ".\nToken: " + sentinel + "\n| " + sentinel + " | 0123456789abcdef0123456789abcdef |\n"},
		"git":                           {Stdout: "admin@jterrazz.com\n"},
		"ssh-add":                       {Stdout: "256 SHA256:76MCcfWNZv0Ixcc4fM0b6Ue1o8l7W1Xo0n3zKhK1Vy4 " + sentinel + " (ED25519)\n"},
		"docker-credential-osxkeychain": {Stdout: `{"https://registry.jterrazz.com":"` + sentinel + `"}`},
	}
}

func signedIn(on ...string) *cli.Reads {
	run := local.Played(func(c local.Cmd) (local.Output, error) { return printed()[c.Name], nil }, on...)
	config := files{fstest.MapFS{"home/" + catalog.DockerSettingsFile: {Data: []byte(`{"credsStore":"osxkeychain"}`)}}}
	return cli.New(run, config, cli.Options{Home: "/home"})
}

func TestIdentities(t *testing.T) {
	contracttest.Identities(t, func(*testing.T) contracttest.IdentitiesSubject {
		return contracttest.IdentitiesSubject{
			Identities: signedIn("gh"), SignedIn: provider(catalog.ProviderGitHub), Who: "jterrazz",
			Missing: provider(catalog.ProviderClaude),
		}
	})
}

func TestNoReadKeepsATokenItsToolPrinted(t *testing.T) {
	ids := signedIn("gh", "npm", "claude", "wrangler", "git", "ssh-add", "docker-credential-osxkeychain")
	for _, id := range []string{catalog.ProviderGitHub, catalog.ProviderNPM, catalog.ProviderClaude, catalog.ProviderCloudflare, catalog.ProviderGit, catalog.ProviderSSH, catalog.ProviderDocker} {
		got, err := ids.Read(t.Context(), provider(id))
		data, _ := json.Marshal(got)
		if err != nil || got.State != catalog.StateOK {
			t.Errorf("%s: the read itself is lost with the token: %s, %v", id, data, err)
		}
		if strings.Contains(string(data), "SENTINEL") || strings.Contains(string(data), "gho_") {
			t.Errorf("%s kept a token: %s", id, data)
		}
	}
}
