package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type Options struct {
	Net local.Net
	// value: the GitHub API's root, the token its calls carry when there is
	// one, and the repository whose latest release is read.
	APIURL, Token, Repo string
}

type Releases struct{ o Options }

var _ contract.Releases = (*Releases)(nil)

func New(o Options) *Releases { return &Releases{o: o} }

type answer struct {
	Tag    string `json:"tag_name"`
	URL    string `json:"html_url"`
	Assets []struct {
		Name string `json:"name"`
	} `json:"assets"`
}

func (r *Releases) Latest(ctx context.Context) (engine.Tagged, error) {
	// outside: GitHub's /releases/latest answers the newest release that is
	// neither a draft nor a pre-release, and a 404 when the repository has
	// none.
	header := map[string]string{"Accept": "application/vnd.github+json"}
	if r.o.Token != "" {
		header["Authorization"] = "Bearer " + r.o.Token
	}
	got, err := r.o.Net.Fetch(ctx, strings.TrimSuffix(r.o.APIURL, "/")+"/repos/"+r.o.Repo+"/releases/latest", header)
	switch {
	case err != nil:
		return engine.Tagged{}, fmt.Errorf("GitHub: %w", err)
	case got.Status == 404:
		return engine.Tagged{}, nil
	case got.Status != 200:
		return engine.Tagged{}, fmt.Errorf("GitHub said %d", got.Status)
	}
	var a answer
	if err := json.Unmarshal(got.Body, &a); err != nil {
		return engine.Tagged{}, fmt.Errorf("the latest release of %s: %w", r.o.Repo, err)
	}
	out := engine.Tagged{Tag: a.Tag, URL: a.URL}
	for _, asset := range a.Assets {
		out.Assets = append(out.Assets, asset.Name)
	}
	return out, nil
}
