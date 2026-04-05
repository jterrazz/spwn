package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultAPIBase = "https://api.github.com"

// Asset is one downloadable file attached to a GitHub release.
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// Release is the subset of the GitHub releases API we care about.
type Release struct {
	TagName    string  `json:"tag_name"`
	Prerelease bool    `json:"prerelease"`
	HTMLURL    string  `json:"html_url"`
	Assets     []Asset `json:"assets"`
}

// Channel decides which releases are considered acceptable.
type Channel string

const (
	// ChannelStable returns only full (non-prerelease) releases.
	ChannelStable Channel = "stable"
	// ChannelBeta returns both stable and prereleases, newest first.
	ChannelBeta Channel = "beta"
)

// ReleaseClient is an interface over GitHub's releases API so tests can
// swap in a fake implementation without hitting the network.
type ReleaseClient interface {
	LatestStable(ctx context.Context) (*Release, error)
	ListReleases(ctx context.Context, limit int) ([]Release, error)
}

// GitHubClient talks to the real GitHub REST API. Zero value is valid
// and uses default endpoints / the default http.Client.
type GitHubClient struct {
	Owner   string        // e.g. "jterrazz"
	Repo    string        // e.g. "spwn"
	BaseURL string        // default: https://api.github.com
	HTTP    *http.Client  // default: http.Client with 10s timeout
	Token   string        // optional, for private repos / higher rate limits
}

func (c *GitHubClient) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (c *GitHubClient) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return defaultAPIBase
}

// LatestStable fetches GitHub's idea of "latest release" (stable only —
// GitHub filters prereleases automatically on this endpoint).
func (c *GitHubClient) LatestStable(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL(), c.Owner, c.Repo)
	var r Release
	if err := c.get(ctx, url, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ListReleases returns up to `limit` releases (most recent first),
// including prereleases. Used for beta/nightly channels.
func (c *GitHubClient) ListReleases(ctx context.Context, limit int) ([]Release, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=%d", c.baseURL(), c.Owner, c.Repo, limit)
	var out []Release
	if err := c.get(ctx, url, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *GitHubClient) get(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("github api: %s not found (is the repo public?)", url)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("github api: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("github api: decode: %w", err)
	}
	return nil
}

// ResolveTarget picks the release that should be offered to a user on the
// given channel. Stable returns GitHub's "latest"; beta walks the release
// list and returns the newest that's either stable or prerelease.
func ResolveTarget(ctx context.Context, client ReleaseClient, channel Channel) (*Release, error) {
	switch channel {
	case ChannelStable, "":
		return client.LatestStable(ctx)
	case ChannelBeta:
		list, err := client.ListReleases(ctx, 10)
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("no releases found")
		}
		// GitHub returns newest first. Just take the first.
		r := list[0]
		return &r, nil
	default:
		return nil, fmt.Errorf("unknown channel: %q", channel)
	}
}

// FindAsset returns the asset whose name matches the given platform token,
// e.g. "darwin_arm64". Returns nil if no match.
func FindAsset(release *Release, platformToken string) *Asset {
	for i := range release.Assets {
		a := &release.Assets[i]
		if strings.Contains(a.Name, platformToken) && !strings.HasSuffix(a.Name, ".txt") && !strings.HasSuffix(a.Name, ".sha256") {
			return a
		}
	}
	return nil
}

// FindChecksumsAsset locates the SHA256SUMS file attached to a release.
// GoReleaser names it "checksums.txt" by default.
func FindChecksumsAsset(release *Release) *Asset {
	for i := range release.Assets {
		a := &release.Assets[i]
		name := strings.ToLower(a.Name)
		if name == "checksums.txt" || strings.HasSuffix(name, "_checksums.txt") || strings.HasSuffix(name, "-checksums.txt") {
			return a
		}
	}
	return nil
}
