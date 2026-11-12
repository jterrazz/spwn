package githubrelease_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/adapter/githubrelease"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/apps/contract/contracttest"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
	"github.com/jterrazz/studio-station/src/platform/local/localtest"
)

const (
	api   = "https://api.test"
	dl    = "https://dl.test"
	repo  = "acme/tool"
	bin   = "/home/u/.local/bin"
	store = "/home/u/.local/share/station/releases"
)

// value: a release as GitHub serves it: its assets by name, the digest
// GitHub gives each, and the checksums file's lines.
type served struct {
	tag     string
	assets  map[string][]byte
	digests map[string]string
}

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func tarball(t testing.TB, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := gzip.NewWriter(&buf)
	w := tar.NewWriter(z)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(files[name])), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// value: tool at tag, built for both arches, each binary saying its tag
// and arch, with a checksums.txt of both and GitHub's digests.
func toolAt(t testing.TB, tag string) served {
	s := served{tag: tag, assets: map[string][]byte{}, digests: map[string]string{}}
	var lines strings.Builder
	for _, arch := range []string{"arm64", "amd64"} {
		name := "tool-" + tag + "-linux-" + arch + ".tar.gz"
		body := tarball(t, map[string]string{"tool-" + tag + "/tool": "tool " + tag + " " + arch, "tool-" + tag + "/README": "read me"})
		s.assets[name] = body
		s.digests[name] = "sha256:" + sum(body)
		lines.WriteString(sum(body) + "  " + name + "\n")
	}
	s.assets["checksums.txt"] = []byte(lines.String())
	return s
}

func (s served) binary(arch string) []byte { return []byte("tool " + s.tag + " " + arch) }

func serve(n *localtest.Net, latest served, others ...served) {
	for k := range n.Responses {
		delete(n.Responses, k)
	}
	for i, s := range append([]served{latest}, others...) {
		type a struct {
			Name   string `json:"name"`
			Digest string `json:"digest,omitempty"`
			URL    string `json:"browser_download_url"`
		}
		list := make([]a, 0, len(s.assets))
		for _, name := range slices.Sorted(maps.Keys(s.assets)) {
			at := dl + "/" + repo + "/releases/download/" + s.tag + "/" + name
			list = append(list, a{Name: name, Digest: s.digests[name], URL: at})
			n.Responses["GET "+at] = localtest.Response{Status: 200, Body: s.assets[name]}
		}
		doc, _ := json.Marshal(map[string]any{"tag_name": s.tag, "assets": list})
		n.Responses["GET "+api+"/repos/"+repo+"/releases/tags/"+strings.TrimPrefix(s.tag, "v")] = localtest.Response{Status: 404}
		n.Responses["GET "+api+"/repos/"+repo+"/releases/tags/"+s.tag] = localtest.Response{Status: 200, Body: doc}
		if i == 0 {
			n.Responses["GET "+api+"/repos/"+repo+"/releases/latest"] = localtest.Response{Status: 200, Body: doc}
		}
	}
}

// value: a file system that stops at its nth write or rename, as an
// interrupted install does.
type interrupted struct {
	*localtest.FS
	at, done int
}

var errStopped = errors.New("interrupted")

func (f *interrupted) step() error {
	f.done++
	if f.done == f.at {
		return errStopped
	}
	return nil
}

func (f *interrupted) WriteFile(p string, data []byte, perm fs.FileMode) error {
	if err := f.step(); err != nil {
		return err
	}
	return f.FS.WriteFile(p, data, perm)
}

func (f *interrupted) Rename(from, to string) error {
	if err := f.step(); err != nil {
		return err
	}
	return f.FS.Rename(from, to)
}

// value: one clock for every Releases a test builds, an hour on at each
// read, so every latest release asked is asked of GitHub, as a release
// served anew is.
type later struct{ at time.Time }

var hourly = &later{}

func (c *later) Now() time.Time {
	c.at = c.at.Add(61 * time.Minute)
	return c.at
}

func (c *later) Sleep(context.Context, time.Duration) error { return nil }

func releases(files local.FS, n local.Net, arch string) *githubrelease.Releases {
	return githubrelease.New(githubrelease.Options{FS: files, Net: n, Clock: hourly, APIURL: api, OS: "linux", Arch: arch, Bin: bin, Store: store})
}

func tool(tag string) catalog.GitHubReleaseInstall {
	return catalog.GitHubReleaseInstall{Repo: repo, Asset: "tool-*-{os}-{arch}.tar.gz", Tag: tag, Commands: []string{"tool"}}
}

func TestInstaller(t *testing.T) {
	contracttest.Installer(t, tool(""), func(_ *testing.T, seed contracttest.Seed) contract.Installer {
		files, n := localtest.NewFS(), localtest.NewNet()
		serve(n, toolAt(t, "v2.0.0"))
		if seed.Fails {
			n.Responses["GET "+api+"/repos/"+repo+"/releases/latest"] = localtest.Response{Status: 502}
		}
		r := releases(files, n, "arm64")
		for range seed.Installed {
			if err := r.Install(context.Background(), []catalog.Install{tool("")}, nil); err != nil {
				t.Fatal(err)
			}
		}
		return r
	})
}

func TestWhatAnInstallFetchesIsWhatItsChecksumVouchesFor(t *testing.T) {
	// invariant: on every path an install takes (fresh, upgrade, re-run,
	// interrupted at each of its writes then run again, a wrong checksum,
	// none, another arch's asset), the command on the search path is a
	// binary a checksum of its release vouched for, or the one it was
	// before; and the inventory never holds a tag whose binary is not the
	// one in place.
	v1, v2 := toolAt(t, "v1.0.0"), toolAt(t, "v2.0.0")
	vouched := map[string]string{sum(v1.binary("arm64")): "v1.0.0", sum(v2.binary("arm64")): "v2.0.0"}
	type path struct {
		name    string
		before  func(files *localtest.FS, n *localtest.Net)
		serving func() served
		in      catalog.GitHubReleaseInstall
		arch    string
		fail    any
	}
	installed := func(at served) func(*localtest.FS, *localtest.Net) {
		return func(files *localtest.FS, n *localtest.Net) {
			serve(n, at)
			if err := releases(files, n, "arm64").Install(context.Background(), []catalog.Install{tool("")}, nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	wrong := func() served {
		s := toolAt(t, "v2.0.0")
		s.assets["checksums.txt"] = []byte(strings.Repeat("0", 64) + "  tool-v2.0.0-linux-arm64.tar.gz\n")
		s.digests = map[string]string{}
		return s
	}
	digestWrong := func() served {
		s := toolAt(t, "v2.0.0")
		s.digests["tool-v2.0.0-linux-arm64.tar.gz"] = "sha256:" + strings.Repeat("1", 64)
		return s
	}
	none := func() served {
		s := toolAt(t, "v2.0.0")
		delete(s.assets, "checksums.txt")
		s.digests = map[string]string{}
		return s
	}
	otherArchOnly := func() served {
		s := toolAt(t, "v2.0.0")
		delete(s.assets, "tool-v2.0.0-linux-arm64.tar.gz")
		return s
	}
	swapped := func() served {
		s := toolAt(t, "v2.0.0")
		s.assets["tool-v2.0.0-linux-arm64.tar.gz"], s.assets["tool-v2.0.0-linux-amd64.tar.gz"] = s.assets["tool-v2.0.0-linux-amd64.tar.gz"], s.assets["tool-v2.0.0-linux-arm64.tar.gz"]
		s.digests = map[string]string{}
		return s
	}
	paths := []path{
		{name: "fresh", serving: func() served { return v2 }, in: tool("")},
		{name: "pinned", serving: func() served { return v2 }, in: tool("1.0.0"), before: func(_ *localtest.FS, n *localtest.Net) { serve(n, v2, v1) }},
		{name: "upgrade", before: installed(v1), serving: func() served { return v2 }, in: tool("")},
		{name: "re-run", before: installed(v2), serving: func() served { return v2 }, in: tool("")},
		{name: "wrong checksum", before: installed(v1), serving: wrong, in: tool(""), fail: new(*githubrelease.MismatchError)},
		{name: "wrong digest", before: installed(v1), serving: digestWrong, in: tool(""), fail: new(*githubrelease.MismatchError)},
		{name: "another arch's asset under this one's name", serving: swapped, in: tool(""), fail: new(*githubrelease.MismatchError)},
		{name: "no checksum", before: installed(v1), serving: none, in: tool(""), fail: new(*githubrelease.UnverifiedError)},
		{name: "no asset for this arch", serving: otherArchOnly, in: tool(""), fail: new(*githubrelease.NoAssetError)},
		{name: "another arch", serving: func() served { return v2 }, in: tool(""), arch: "amd64"},
	}
	check := func(t *testing.T, files *localtest.FS, r *githubrelease.Releases, arch string) {
		t.Helper()
		good := maps.Clone(vouched)
		good[sum(v1.binary("amd64"))], good[sum(v2.binary("amd64"))] = "v1.0.0", "v2.0.0"
		data, err := files.ReadFile(bin + "/tool")
		held, _ := r.Inventory(context.Background())
		if err != nil {
			if _, ok := held[repo]; ok {
				t.Errorf("no tool in place, the inventory holds %s", held[repo])
			}
			return
		}
		tag, ok := good[sum(data)]
		if !ok {
			t.Fatalf("%s holds %q, which no checksum vouched for", bin+"/tool", data)
		}
		if got, listed := held[repo]; listed && got != tag {
			t.Errorf("the inventory holds %s while the binary in place is %s's", got, tag)
		}
		_ = arch
	}
	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			files, n := localtest.NewFS(), localtest.NewNet()
			if p.before != nil {
				p.before(files, n)
			}
			arch := cmpOr(p.arch, "arm64")
			if p.name != "pinned" {
				serve(n, p.serving())
			}
			r := releases(files, n, arch)
			err := r.Install(context.Background(), []catalog.Install{p.in}, nil)
			if p.fail == nil && err != nil {
				t.Fatalf("install: %v", err)
			}
			if p.fail != nil && !errors.As(err, p.fail) {
				t.Fatalf("install gave %v; want %T", err, p.fail)
			}
			check(t, files, r, arch)
			if p.fail == nil {
				data, _ := files.ReadFile(bin + "/tool")
				want := p.serving()
				if p.name == "pinned" {
					want = v1
				}
				if !bytes.Equal(data, want.binary(arch)) {
					t.Errorf("tool is %q; want %q", data, want.binary(arch))
				}
			}
		})
	}
	for at := 1; at <= 6; at++ {
		t.Run("upgrade interrupted at write "+strconv.Itoa(at)+", then run again", func(t *testing.T) {
			files, n := localtest.NewFS(), localtest.NewNet()
			installed(v1)(files, n)
			serve(n, v2)
			stopping := &interrupted{FS: files, at: at}
			err := releases(stopping, n, "arm64").Install(context.Background(), []catalog.Install{tool("")}, nil)
			r := releases(files, n, "arm64")
			check(t, files, r, "arm64")
			if err == nil {
				return
			}
			if err := r.Install(context.Background(), []catalog.Install{tool("")}, nil); err != nil {
				t.Fatalf("the run after the interruption: %v", err)
			}
			check(t, files, r, "arm64")
			if held, _ := r.Inventory(context.Background()); held[repo] != "v2.0.0" {
				t.Errorf("after the run again the inventory holds %v; want v2.0.0", held)
			}
		})
	}
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func TestAReleaseWithNoChecksumInstallsOnlyWhenItsMethodSaysWhy(t *testing.T) {
	files, n := localtest.NewFS(), localtest.NewNet()
	s := toolAt(t, "v2.0.0")
	delete(s.assets, "checksums.txt")
	s.digests = map[string]string{}
	serve(n, s)
	in := tool("")
	in.Unverified = "the project publishes no checksums"

	if err := releases(files, n, "arm64").Install(context.Background(), []catalog.Install{in}, nil); err != nil {
		t.Fatal(err)
	}
	if data, _ := files.ReadFile(bin + "/tool"); !bytes.Equal(data, s.binary("arm64")) {
		t.Errorf("tool is %q", data)
	}
}

func TestAnInstallNeverOverwritesACommandItDidNotPut(t *testing.T) {
	files, n := localtest.NewFS(), localtest.NewNet()
	serve(n, toolAt(t, "v2.0.0"))
	_ = files.WriteFile(bin+"/tool", []byte("the owner's own"), 0o755)

	err := releases(files, n, "arm64").Install(context.Background(), []catalog.Install{tool("")}, nil)

	var theirs *githubrelease.NotOursError
	if !errors.As(err, &theirs) {
		t.Fatalf("install gave %v; want the owner's file refused", err)
	}
	if data, _ := files.ReadFile(bin + "/tool"); string(data) != "the owner's own" {
		t.Errorf("tool is %q; want the owner's", data)
	}
}

func TestARemovalKeepsACommandTheOwnerPutInItsPlace(t *testing.T) {
	files, n := localtest.NewFS(), localtest.NewNet()
	serve(n, toolAt(t, "v2.0.0"))
	r := releases(files, n, "arm64")
	if err := r.Install(context.Background(), []catalog.Install{tool("")}, nil); err != nil {
		t.Fatal(err)
	}
	_ = files.WriteFile(bin+"/tool", []byte("the owner's own"), 0o755)

	if err := r.Remove(context.Background(), []catalog.Install{tool("")}, nil); err != nil {
		t.Fatal(err)
	}
	if data, _ := files.ReadFile(bin + "/tool"); string(data) != "the owner's own" {
		t.Errorf("tool is %q; want the owner's kept", data)
	}
	if held, _ := r.Inventory(context.Background()); slices.Contains(slices.Collect(maps.Keys(held)), repo) {
		t.Errorf("the inventory still holds %v", held)
	}
}

func TestStationsOwnReleaseIsHeldAndNeverTouched(t *testing.T) {
	r := githubrelease.Own(releases(localtest.NewFS(), localtest.NewNet(), "arm64"), githubrelease.Held{Key: "acme/station"})
	self := catalog.GitHubReleaseInstall{Repo: "acme/station", Asset: "station-{os}-{arch}", Commands: []string{"station"}}

	held, _ := r.Inventory(context.Background())
	if _, ok := held["acme/station"]; !ok {
		t.Errorf("the inventory holds %v; want station's own release", held)
	}
	var mine *githubrelease.SelfError
	for _, err := range []error{r.Install(context.Background(), []catalog.Install{self}, nil), r.Remove(context.Background(), []catalog.Install{self}, nil)} {
		if !errors.As(err, &mine) {
			t.Errorf("gave %v; want station's own release refused", err)
		}
	}
	if got, _ := r.Outdated(context.Background(), []string{"acme/station"}); got["acme/station"].Verdict != apps.Current {
		t.Errorf("station's own release reads %v; want current", got)
	}
}

func TestOutdatedComparesTheLatestTagWithTheOneInstalled(t *testing.T) {
	files, n := localtest.NewFS(), localtest.NewNet()
	serve(n, toolAt(t, "v1.0.0"))
	r := releases(files, n, "arm64")
	if err := r.Install(context.Background(), []catalog.Install{tool("")}, nil); err != nil {
		t.Fatal(err)
	}
	serve(n, toolAt(t, "v2.0.0"))

	got, _ := r.Outdated(context.Background(), []string{repo})
	if got[repo] != (apps.Currency{Verdict: apps.Behind, Newest: "v2.0.0"}) {
		t.Errorf("tool reads %+v; want behind v2.0.0", got[repo])
	}
	n.Responses["GET "+api+"/repos/"+repo+"/releases/latest"] = localtest.Response{Status: 403}
	if got, _ := r.Outdated(context.Background(), []string{repo}); got[repo].Verdict != apps.Unknown {
		t.Errorf("an unanswered read gave %+v; want unknown", got[repo])
	}
}

func TestALatestReleaseIsAskedOfGitHubOnceAnHourAndADownloadNever(t *testing.T) {
	files, n := localtest.NewFS(), localtest.NewNet()
	serve(n, toolAt(t, "v2.0.0"))
	clock := &localtest.Clock{At: time.Date(2026, 11, 12, 1, 0, 0, 0, time.UTC)}
	r := githubrelease.New(githubrelease.Options{FS: files, Net: n, Clock: clock, APIURL: api, OS: "linux", Arch: "arm64", Bin: bin, Store: store})
	calls := func() int {
		count := 0
		for _, q := range n.Requests {
			if strings.HasPrefix(q.URL, api) {
				count++
			}
		}
		return count
	}

	if err := r.Install(context.Background(), []catalog.Install{tool("")}, nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		clock.At = clock.At.Add(15 * time.Minute)
		if got, _ := r.Outdated(context.Background(), []string{repo}); got[repo].Verdict != apps.Current {
			t.Fatalf("tool reads %+v; want current", got[repo])
		}
	}
	if got := calls(); got != 1 {
		t.Errorf("an install and three reads within the hour asked the API %d times; want 1 (%v)", got, n.Requests)
	}
	serve(n, toolAt(t, "v3.0.0"))
	clock.At = clock.At.Add(16 * time.Minute)
	if got, _ := r.Outdated(context.Background(), []string{repo}); got[repo] != (apps.Currency{Verdict: apps.Behind, Newest: "v3.0.0"}) {
		t.Errorf("past the hour tool reads %+v; want behind v3.0.0", got[repo])
	}
	if got := calls(); got != 2 {
		t.Errorf("past the hour the API was asked %d times in all; want 2", got)
	}
}

func TestARateRefusalSaysUntilWhenAndWhatRaisesTheLimit(t *testing.T) {
	ends := time.Date(2026, 11, 12, 2, 15, 10, 0, time.UTC)
	spent := localtest.Response{Status: 403, Header: map[string]string{
		"X-Ratelimit-Limit": "60", "X-Ratelimit-Remaining": "0", "X-Ratelimit-Reset": strconv.FormatInt(ends.Unix(), 10),
	}}
	for _, c := range []struct {
		name, token string
		answer      localtest.Response
		want        string
	}{
		{name: "without a token", answer: spent, want: "the release of acme/tool did not answer: GitHub refuses calls without a token until 02:15 UTC: its limit of 60 calls an hour is spent; set GITHUB_TOKEN (or GH_TOKEN) to a GitHub token to raise it"},
		{name: "with a token", token: "t", answer: spent, want: "the release of acme/tool did not answer: GitHub refuses the token's calls until 02:15 UTC: its limit of 60 calls an hour is spent"},
		{name: "a 429 with a wait", answer: localtest.Response{Status: 429, Header: map[string]string{"Retry-After": "120"}}, want: "the release of acme/tool did not answer: GitHub refuses calls without a token until 01:02 UTC: its hourly limit is spent; set GITHUB_TOKEN (or GH_TOKEN) to a GitHub token to raise it"},
	} {
		t.Run(c.name, func(t *testing.T) {
			files, n := localtest.NewFS(), localtest.NewNet()
			n.Responses["GET "+api+"/repos/"+repo+"/releases/latest"] = c.answer
			clock := &localtest.Clock{At: time.Date(2026, 11, 12, 1, 0, 0, 0, time.UTC)}
			r := githubrelease.New(githubrelease.Options{FS: files, Net: n, Clock: clock, APIURL: api, Token: c.token, OS: "linux", Arch: "arm64", Bin: bin, Store: store})

			err := r.Install(context.Background(), []catalog.Install{tool("")}, nil)

			var limited *githubrelease.LimitError
			if !errors.As(err, &limited) || err.Error() != c.want {
				t.Fatalf("install gave %v;\nwant %s", err, c.want)
			}
		})
	}
	files, n := localtest.NewFS(), localtest.NewNet()
	n.Responses["GET "+api+"/repos/"+repo+"/releases/latest"] = localtest.Response{Status: 403}
	err := releases(files, n, "arm64").Install(context.Background(), []catalog.Install{tool("")}, nil)
	if err == nil || err.Error() != "the release of acme/tool did not answer: GitHub said 403" {
		t.Errorf("a 403 over no limit gave %v", err)
	}
}
