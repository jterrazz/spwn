package github_test

import (
	"encoding/json"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/adapter/github"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local/localtest"
)

const latest = "GET https://api.example/repos/jterrazz/spwn/releases/latest"

func TestReleases(t *testing.T) {
	contracttest.Releases(t, func(t *testing.T) contracttest.ReleasesSubject {
		net := localtest.NewNet()
		net.Responses[latest] = localtest.Response{Status: 404}
		return contracttest.ReleasesSubject{
			Releases: github.New(github.Options{Net: net, APIURL: "https://api.example/", Repo: "jterrazz/spwn"}),
			Release: func(p engine.Tagged) {
				assets := make([]map[string]string, 0, len(p.Assets))
				for _, a := range p.Assets {
					assets = append(assets, map[string]string{"name": a})
				}
				body, err := json.Marshal(map[string]any{"tag_name": p.Tag, "html_url": p.URL, "assets": assets, "prerelease": false})
				if err != nil {
					t.Fatal(err)
				}
				net.Responses[latest] = localtest.Response{Status: 200, Body: body}
			},
			Down: func() { net.Responses[latest] = localtest.Response{Status: 502} },
		}
	})
}

func TestARefusalNamesWhatGitHubSaid(t *testing.T) {
	net := localtest.NewNet()
	net.Responses[latest] = localtest.Response{Status: 403}
	_, err := github.New(github.Options{Net: net, APIURL: "https://api.example", Repo: "jterrazz/spwn"}).Latest(t.Context())
	if err == nil || err.Error() != "GitHub said 403" {
		t.Errorf("Latest = %v; want GitHub said 403", err)
	}
}
