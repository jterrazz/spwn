package githubrelease

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/platform/local"
)

const name = "github-release"

type Options struct {
	FS  local.FS
	Net local.Net
	// value: the GitHub API's root, and the token its calls carry when
	// there is one.
	APIURL, Token string
	// value: the machine's words for an asset's `{os}` and `{arch}`:
	// `linux` or `darwin`, `arm64` or `amd64`.
	OS, Arch string
	// value: the folder the commands land in, on the search path.
	Bin string
	// value: station's own folder of what it installed, one record per
	// release: `<owner>/<name>.json`.
	Store string
}

type Releases struct{ o Options }

func New(o Options) *Releases { return &Releases{o: o} }

func (r *Releases) Method() catalog.Method { return catalog.MethodGitHubRelease }

func (r *Releases) Manager() contract.Manager { return contract.Manager{App: name} }

func (r *Releases) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true, UpdateAll: true}
}

func (r *Releases) Ready(context.Context) error { return nil }

// invariant: what a release installed is its record, written before its
// commands land (the install under way, `pending`) and once they
// are (its tag and files), each command by the hash of the file written.
type record struct {
	Repo   string            `json:"repo"`
	Tag    string            `json:"tag,omitempty"`
	Asset  string            `json:"asset,omitempty"`
	SHA256 string            `json:"sha256,omitempty"`
	Files  map[string]string `json:"files,omitempty"`
	// value: the tag and commands of an install under way: a file left by
	// an interrupted install is station's still.
	PendingTag string            `json:"pending_tag,omitempty"`
	Pending    map[string]string `json:"pending,omitempty"`
}

// value: the tag whose files are every command in place: the finished
// install's, else the one under way once all its commands landed; none
// while the commands in place are of neither, as an interruption between
// two of them leaves them, which the next install mends.
func (r *Releases) heldTag(rec record) string {
	if rec.Pending == nil {
		return rec.Tag
	}
	matches := func(files map[string]string) bool {
		for cmd, h := range files {
			if target, err := r.command(cmd); err != nil || r.hash(target) != h {
				return false
			}
		}
		return len(files) > 0
	}
	switch {
	case matches(rec.Pending):
		return rec.PendingTag
	case rec.Tag != "" && matches(rec.Files):
		return rec.Tag
	}
	return ""
}

func (r *Releases) Inventory(context.Context) (map[string]string, error) {
	held := map[string]string{}
	files, err := r.o.FS.Glob(path.Join(r.o.Store, "*", "*.json"))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		// invariant: a record that does not read, or names what is no
		// repo or no command, holds nothing: the rest are still read, and
		// an install or a removal of its release names it.
		rec, err := r.readRecord(f)
		if err != nil {
			continue
		}
		if tag := r.heldTag(rec); tag != "" {
			held[rec.Repo] = tag
		}
	}
	return held, nil
}

func (r *Releases) Outdated(ctx context.Context, keys []string) (map[string]apps.Currency, error) {
	// invariant: a release is behind when the latest release's tag is
	// another than the one installed; an unanswered read leaves it unknown.
	out := map[string]apps.Currency{}
	for _, k := range keys {
		out[k] = apps.Currency{Verdict: apps.Unknown}
		rec, err := r.record(k)
		held := r.heldTag(rec)
		if err != nil || held == "" {
			continue
		}
		latest, err := r.release(ctx, k, "")
		if err != nil {
			continue
		}
		out[k] = apps.Currency{Verdict: apps.Current}
		if latest.Tag != held {
			out[k] = apps.Currency{Verdict: apps.Behind, Newest: latest.Tag}
		}
	}
	return out, nil
}

func (r *Releases) Install(ctx context.Context, installs []catalog.Install, _ contract.Logs) error {
	for _, i := range installs {
		v, ok := i.(catalog.GitHubReleaseInstall)
		if !ok {
			return errors.New(i.Key() + " is not a GitHub release")
		}
		if err := r.install(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

func (r *Releases) Update(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return r.Install(ctx, installs, logs)
}

func (r *Releases) Remove(_ context.Context, installs []catalog.Install, _ contract.Logs) error {
	// invariant: a removal deletes a command only while it is the file
	// station wrote there; one the owner put in its place is kept.
	for _, i := range installs {
		rec, err := r.record(i.Key())
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		for _, cmd := range commands(rec) {
			target, bad := r.command(cmd)
			if bad != nil {
				return bad
			}
			if r.ours(rec, cmd, target) {
				if gone := r.o.FS.Remove(target); gone != nil && !errors.Is(gone, fs.ErrNotExist) {
					return gone
				}
			}
		}
		file, err := recordPath(r.o.Store, rec.Repo)
		if err != nil {
			return err
		}
		if err := r.o.FS.Remove(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// invariant: what an install fetches is what a checksum vouches for, on
// every path: the asset is read whole, hashed, and checked against every
// checksum the release gives for it (its digest, its checksums file)
// before anything is written; none is a refusal unless the method says
// why, one that differs always is. Each command lands by a rename over
// its old file, so an interruption leaves the old command or the new
// one, never a part of either, and the record is written last.
func (r *Releases) install(ctx context.Context, v catalog.GitHubReleaseInstall) error {
	if err := catalog.CheckRepo(v.Repo); err != nil {
		return err
	}
	for _, cmd := range v.Commands {
		if err := catalog.CheckCommand(cmd); err != nil {
			return err
		}
	}
	rel, err := r.release(ctx, v.Repo, v.Tag)
	if err != nil {
		return err
	}
	rec, err := r.record(v.Repo)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		rec = record{Repo: v.Repo}
	case err != nil:
		return err
	}
	if rec.Pending == nil && rec.Tag == rel.Tag && r.whole(rec, v.Commands) {
		return nil
	}
	pattern := r.expand(v.Asset)
	asset, err := rel.only(pattern, r.o.OS, r.o.Arch, v.Repo)
	if err != nil {
		return err
	}
	body, err := r.download(ctx, v.Repo, asset)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	got := hex.EncodeToString(sum[:])
	vouchers, err := r.vouchers(ctx, v, rel, asset)
	if err != nil {
		return err
	}
	if len(vouchers) == 0 && v.Unverified == "" {
		return &UnverifiedError{Repo: v.Repo, Tag: rel.Tag, Asset: asset.Name}
	}
	for _, w := range vouchers {
		if !strings.EqualFold(w.sum, got) {
			return &MismatchError{Asset: asset.Name, Source: w.source, Want: strings.ToLower(w.sum), Got: got}
		}
	}
	files, err := unpack(asset.Name, body, v.Commands)
	if err != nil {
		return err
	}
	written := map[string]string{}
	for cmd, data := range files {
		target, err := r.command(cmd)
		if err != nil {
			return err
		}
		if _, err := r.o.FS.Stat(target); err == nil && !r.ours(rec, cmd, target) {
			return &NotOursError{Path: target}
		}
		h := sha256.Sum256(data)
		written[cmd] = hex.EncodeToString(h[:])
	}
	rec.PendingTag, rec.Pending = rel.Tag, written
	if err := r.writeRecord(rec); err != nil {
		return err
	}
	if err := r.o.FS.MkdirAll(r.o.Bin, 0o755); err != nil {
		return err
	}
	for cmd, data := range files {
		target, err := r.command(cmd)
		if err != nil {
			return err
		}
		if err := r.land(target, data, 0o755); err != nil {
			return err
		}
	}
	if rec.Files == nil {
		rec.Files = map[string]string{}
	}
	for cmd, h := range written {
		rec.Files[cmd] = h
	}
	rec.Tag, rec.Asset, rec.SHA256, rec.PendingTag, rec.Pending = rel.Tag, asset.Name, got, "", nil
	return r.writeRecord(rec)
}

func (r *Releases) expand(pattern string) string {
	return strings.NewReplacer("{os}", r.o.OS, "{arch}", r.o.Arch).Replace(pattern)
}

// value: whether every command of a release is the file its record says.
func (r *Releases) whole(rec record, cmds []string) bool {
	for _, cmd := range cmds {
		target, err := r.command(cmd)
		if h, ok := rec.Files[cmd]; !ok || err != nil || r.hash(target) != h {
			return false
		}
	}
	return len(cmds) > 0
}

func (r *Releases) ours(rec record, cmd, target string) bool {
	h := r.hash(target)
	return h != "" && (rec.Files[cmd] == h || rec.Pending[cmd] == h)
}

func (r *Releases) hash(file string) string {
	data, err := r.o.FS.ReadFile(file)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func commands(rec record) []string {
	var out []string
	for cmd := range rec.Files {
		out = append(out, cmd)
	}
	for cmd := range rec.Pending {
		if _, ok := rec.Files[cmd]; !ok {
			out = append(out, cmd)
		}
	}
	return out
}

func (r *Releases) land(target string, data []byte, perm fs.FileMode) error {
	// invariant: the file written before the rename is made anew, so a
	// link left at its name is removed, never followed to what it names.
	next := path.Join(path.Dir(target), "."+path.Base(target)+".station-new")
	if err := r.o.FS.Remove(next); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := r.o.FS.WriteFile(next, data, perm); err != nil {
		return err
	}
	if err := r.o.FS.Chmod(next, perm); err != nil {
		return err
	}
	return r.o.FS.Rename(next, target)
}

// invariant: a command's path is one file of the folder on the search
// path, and a record's one file two folders down the store: the name that
// builds either is checked by the catalog's rule here, where the path is
// made, whatever checked it before (the decoder, a record's read), so
// nothing station writes or removes is outside those folders.
func (r *Releases) command(cmd string) (string, error) {
	if err := catalog.CheckCommand(cmd); err != nil {
		return "", err
	}
	return path.Join(r.o.Bin, cmd), nil
}

func recordPath(store, repo string) (string, error) {
	if err := catalog.CheckRepo(repo); err != nil {
		return "", err
	}
	return path.Join(store, repo+".json"), nil
}

func (r *Releases) record(repo string) (record, error) {
	file, err := recordPath(r.o.Store, repo)
	if err != nil {
		return record{}, err
	}
	return r.readRecord(file)
}

func (r *Releases) readRecord(file string) (record, error) {
	data, err := r.o.FS.ReadFile(file)
	if err != nil {
		return record{}, err
	}
	var rec record
	if err = json.Unmarshal(data, &rec); err != nil {
		return record{}, &RecordError{File: file, Err: err}
	}
	at, err := recordPath(r.o.Store, rec.Repo)
	if err != nil {
		return record{}, &RecordError{File: file, Err: err}
	}
	if at != file {
		return record{}, &RecordError{File: file, Err: errors.New("it names " + rec.Repo + ", whose record is elsewhere")}
	}
	for _, cmd := range commands(rec) {
		if err := catalog.CheckCommand(cmd); err != nil {
			return record{}, &RecordError{File: file, Err: err}
		}
	}
	return rec, nil
}

func (r *Releases) writeRecord(rec record) error {
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	file, err := recordPath(r.o.Store, rec.Repo)
	if err != nil {
		return err
	}
	if err := r.o.FS.MkdirAll(path.Dir(file), 0o755); err != nil {
		return err
	}
	return r.land(file, append(data, '\n'), 0o644)
}

// outside: GitHub's release object, the fields station reads: its tag and
// each asset's id, name and digest (`sha256:<hex>`, given by GitHub for
// an asset uploaded since 2025).
type release struct {
	Tag    string  `json:"tag_name"`
	Assets []asset `json:"assets"`
}

type asset struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

func (r *Releases) release(ctx context.Context, repo, tag string) (release, error) {
	// invariant: a tag asked is the release's tag, or that tag with a `v`
	// before it (`1.4.4` finds `v1.4.4`); none asked, the latest release.
	tries := []string{"/releases/latest"}
	if tag != "" {
		tries = []string{"/releases/tags/" + tag}
		if !strings.HasPrefix(tag, "v") {
			tries = append(tries, "/releases/tags/v"+tag)
		}
	}
	for _, t := range tries {
		status, body, err := r.get(ctx, "/repos/"+repo+t, "application/vnd.github+json")
		if err != nil {
			return release{}, err
		}
		if status == 404 {
			continue
		}
		if status != 200 {
			return release{}, fmt.Errorf("the release of %s did not answer: GitHub said %d", repo, status)
		}
		var rel release
		if err := json.Unmarshal(body, &rel); err != nil {
			return release{}, fmt.Errorf("the release of %s: %w", repo, err)
		}
		return rel, nil
	}
	return release{}, &NoReleaseError{Repo: repo, Tag: tag}
}

func (rel release) only(pattern, os, arch, repo string) (asset, error) {
	var found []asset
	for _, a := range rel.Assets {
		if ok, _ := path.Match(pattern, a.Name); ok {
			found = append(found, a)
		}
	}
	switch len(found) {
	case 0:
		return asset{}, &NoAssetError{Repo: repo, Tag: rel.Tag, Pattern: pattern, OS: os, Arch: arch}
	case 1:
		return found[0], nil
	}
	return asset{}, fmt.Errorf("%d assets of %s %s match %s: the method names one", len(found), repo, rel.Tag, pattern)
}

func (r *Releases) download(ctx context.Context, repo string, a asset) ([]byte, error) {
	status, body, err := r.get(ctx, "/repos/"+repo+"/releases/assets/"+strconv.FormatInt(a.ID, 10), "application/octet-stream")
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("%s did not download: GitHub said %d", a.Name, status)
	}
	return body, nil
}

func (r *Releases) get(ctx context.Context, route, accept string) (status int, body []byte, err error) {
	header := map[string]string{"Accept": accept}
	if r.o.Token != "" {
		header["Authorization"] = "Bearer " + r.o.Token
	}
	status, body, err = r.o.Net.Get(ctx, strings.TrimSuffix(r.o.APIURL, "/")+route, header)
	if err != nil {
		return 0, nil, fmt.Errorf("GitHub: %w", err)
	}
	return status, body, nil
}

type voucher struct{ source, sum string }

// value: every checksum the release gives for the asset: its digest, and
// each line of a checksums file that names it (a file the method names,
// else the asset's own `.sha256` and the release's checksums files).
func (r *Releases) vouchers(ctx context.Context, v catalog.GitHubReleaseInstall, rel release, a asset) ([]voucher, error) {
	var out []voucher
	if hexSum, ok := strings.CutPrefix(a.Digest, "sha256:"); ok && isSum(hexSum) {
		out = append(out, voucher{source: "the release's digest", sum: hexSum})
	}
	var files []asset
	for _, c := range rel.Assets {
		if c.Name == a.Name {
			continue
		}
		if v.Checksums != "" {
			if ok, _ := path.Match(r.expand(v.Checksums), c.Name); ok {
				files = append(files, c)
			}
			continue
		}
		if checksums(c.Name, a.Name) {
			files = append(files, c)
		}
	}
	if v.Checksums != "" && len(files) == 0 {
		return nil, fmt.Errorf("%s %s has no asset %s, the checksums its method names", v.Repo, rel.Tag, v.Checksums)
	}
	for _, f := range files {
		body, err := r.download(ctx, v.Repo, f)
		if err != nil {
			return nil, err
		}
		if sum, ok := listed(string(body), a.Name, f.Name == a.Name+".sha256" || f.Name == a.Name+".sha256sum"); ok {
			out = append(out, voucher{source: f.Name, sum: sum})
		}
	}
	return out, nil
}

func checksums(file, asset string) bool {
	lower := strings.ToLower(file)
	for _, signature := range []string{".sig", ".asc", ".pem", ".sigstore", ".bundle", ".intoto.jsonl"} {
		if strings.HasSuffix(lower, signature) {
			return false
		}
	}
	return file == asset+".sha256" || file == asset+".sha256sum" || strings.Contains(lower, "checksum") || strings.Contains(lower, "sha256sum")
}

// value: the sum a checksums file gives the asset: the line `<sum>  <name>`
// (`*<name>` in binary mode, `./<name>`), or the lone sum of the asset's
// own file.
func listed(body, asset string, own bool) (string, bool) {
	for line := range strings.Lines(body) {
		f := strings.Fields(line)
		switch {
		case len(f) >= 2 && isSum(f[0]) && strings.TrimPrefix(strings.TrimPrefix(f[1], "*"), "./") == asset:
			return f[0], true
		case len(f) == 1 && own && isSum(f[0]):
			return f[0], true
		}
	}
	return "", false
}

func isSum(s string) bool {
	if len(s) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
