package upgrade

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitHubClient_LatestStable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("wrong Accept header: %s", got)
		}
		_ = json.NewEncoder(w).Encode(Release{
			TagName: "v1.2.3",
			Assets: []Asset{
				{Name: "spwn_darwin_arm64.tar.gz", DownloadURL: "https://dl/arm64", Size: 1234},
				{Name: "checksums.txt", DownloadURL: "https://dl/checksums", Size: 256},
			},
		})
	}))
	defer server.Close()

	c := &GitHubClient{Owner: "jt", Repo: "spwn", BaseURL: server.URL}
	rel, err := c.LatestStable(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.TagName != "v1.2.3" {
		t.Errorf("tag = %q", rel.TagName)
	}
	if len(rel.Assets) != 2 {
		t.Fatalf("assets = %d", len(rel.Assets))
	}
}

func TestGitHubClient_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()
	c := &GitHubClient{Owner: "jt", Repo: "spwn", BaseURL: server.URL}
	_, err := c.LatestStable(context.Background())
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 error, got %v", err)
	}
}

// fakeReleaseClient lets ResolveTarget tests avoid any HTTP.
type fakeReleaseClient struct {
	latest *Release
	list   []Release
}

func (f *fakeReleaseClient) LatestStable(context.Context) (*Release, error) {
	return f.latest, nil
}

func (f *fakeReleaseClient) ListReleases(context.Context, int) ([]Release, error) {
	return f.list, nil
}

func TestResolveTarget_Stable(t *testing.T) {
	client := &fakeReleaseClient{latest: &Release{TagName: "v1.0.0"}}
	r, err := ResolveTarget(context.Background(), client, ChannelStable)
	if err != nil || r.TagName != "v1.0.0" {
		t.Errorf("stable = %+v, %v", r, err)
	}
	// Empty channel defaults to stable.
	r, _ = ResolveTarget(context.Background(), client, "")
	if r.TagName != "v1.0.0" {
		t.Errorf("empty channel defaults to stable")
	}
}

func TestResolveTarget_Beta(t *testing.T) {
	client := &fakeReleaseClient{list: []Release{
		{TagName: "v1.1.0-beta.2", Prerelease: true},
		{TagName: "v1.0.0"},
	}}
	r, err := ResolveTarget(context.Background(), client, ChannelBeta)
	if err != nil || r.TagName != "v1.1.0-beta.2" {
		t.Errorf("beta should pick newest (prerelease) = %+v", r)
	}
}

func TestResolveTarget_UnknownChannel(t *testing.T) {
	_, err := ResolveTarget(context.Background(), &fakeReleaseClient{}, "bogus")
	if err == nil {
		t.Error("expected error for unknown channel")
	}
}

func TestFindAsset(t *testing.T) {
	rel := &Release{Assets: []Asset{
		{Name: "spwn_darwin_arm64.tar.gz"},
		{Name: "spwn_darwin_amd64.tar.gz"},
		{Name: "spwn_linux_amd64.tar.gz"},
		{Name: "checksums.txt"},
	}}
	if got := FindAsset(rel, "darwin_arm64"); got == nil || got.Name != "spwn_darwin_arm64.tar.gz" {
		t.Errorf("arm64 lookup: %+v", got)
	}
	if got := FindAsset(rel, "linux_amd64"); got == nil || got.Name != "spwn_linux_amd64.tar.gz" {
		t.Errorf("linux_amd64 lookup: %+v", got)
	}
	if got := FindAsset(rel, "windows_arm64"); got != nil {
		t.Errorf("should not match missing platform, got %+v", got)
	}
	// Must never return the checksums file as a binary asset.
	if got := FindAsset(rel, "checksums"); got != nil {
		t.Errorf("must skip checksums file, got %+v", got)
	}
}

func TestFindChecksumsAsset(t *testing.T) {
	rel := &Release{Assets: []Asset{
		{Name: "spwn_darwin_arm64.tar.gz"},
		{Name: "checksums.txt"},
	}}
	got := FindChecksumsAsset(rel)
	if got == nil || got.Name != "checksums.txt" {
		t.Errorf("expected checksums.txt, got %+v", got)
	}
	rel2 := &Release{Assets: []Asset{{Name: "spwn_darwin_arm64.tar.gz"}}}
	if got := FindChecksumsAsset(rel2); got != nil {
		t.Errorf("no checksums file should return nil, got %+v", got)
	}
}
