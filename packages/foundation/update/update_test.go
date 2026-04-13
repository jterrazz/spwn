package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// End-to-end: a fake GitHub-like server + fake release assets, then
// CheckForUpdate + Apply should update a dummy binary successfully.
func TestCheckAndApply_EndToEnd(t *testing.T) {
	// 1) Build a tar.gz with a "spwn" binary inside.
	archive := makeTarGz(t, map[string][]byte{
		"README.md": []byte("release notes"),
		"spwn":      []byte("NEW_BINARY_BYTES"),
	})
	archiveBytes, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	// 2) SHA256 it for the checksums file.
	sum := sha256.Sum256(archiveBytes)
	archiveDigest := hex.EncodeToString(sum[:])
	checksumsContent := fmt.Sprintf("%s  spwn_linux_amd64.tar.gz\n", archiveDigest)

	// 3) Fake server that serves the release JSON + the two assets.
	var assetURL, checksumsURL string
	var server *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archiveBytes)
	})
	mux.HandleFunc("/checksums", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(checksumsContent))
	})
	mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		body := fmt.Sprintf(`{
            "tag_name": "v2.0.0",
            "prerelease": false,
            "html_url": "%s/release",
            "assets": [
                {"name":"spwn_linux_amd64.tar.gz","browser_download_url":"%s","size":%d},
                {"name":"checksums.txt","browser_download_url":"%s","size":%d}
            ]
        }`, server.URL, assetURL, len(archiveBytes), checksumsURL, len(checksumsContent))
		_, _ = w.Write([]byte(body))
	})
	server = httptest.NewServer(mux)
	defer server.Close()
	assetURL = server.URL + "/asset"
	checksumsURL = server.URL + "/checksums"

	client := &GitHubClient{Owner: "owner", Repo: "repo", BaseURL: server.URL}

	// 4) CheckForUpdate: current version is older, update should be found.
	plan, err := CheckForUpdate(context.Background(), client, "v1.0.0", CheckOpts{
		Channel:  ChannelStable,
		Platform: "linux_amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.UpdateAvail {
		t.Fatal("expected update to be available")
	}
	if plan.Latest.String() != "v2.0.0" {
		t.Errorf("latest = %q", plan.Latest.String())
	}
	if plan.Asset == nil || plan.Asset.Name != "spwn_linux_amd64.tar.gz" {
		t.Fatalf("asset = %+v", plan.Asset)
	}
	if plan.ChecksumAsset == nil {
		t.Fatal("expected checksums asset to be found")
	}

	// 5) Apply: install into a dummy target file.
	target := filepath.Join(t.TempDir(), "spwn")
	if err := os.WriteFile(target, []byte("OLD_BINARY_BYTES"), 0755); err != nil {
		t.Fatal(err)
	}
	var steps []string
	err = Apply(context.Background(), plan, ApplyOpts{
		BinaryName: "spwn",
		TargetPath: target,
		Progress:   func(msg string) { steps = append(steps, msg) },
	})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	// Verify the file on disk is now the new one.
	got, _ := os.ReadFile(target)
	if string(got) != "NEW_BINARY_BYTES" {
		t.Errorf("target after Apply = %q", got)
	}
	// Progress should mention each phase.
	wantSteps := []string{"Downloading spwn_linux_amd64.tar.gz", "Verifying checksum", "Extracting binary", "Installing"}
	if len(steps) != len(wantSteps) {
		t.Fatalf("progress steps = %v", steps)
	}
	for i, s := range wantSteps {
		if steps[i] != s {
			t.Errorf("step %d = %q, want %q", i, steps[i], s)
		}
	}
}

// If the release's checksums.txt does not contain an entry for our asset,
// we refuse to install (treating this as tampering).
func TestApply_RefusesWhenChecksumMissing(t *testing.T) {
	archive := makeTarGz(t, map[string][]byte{"spwn": []byte("x")})
	archiveBytes, _ := os.ReadFile(archive)

	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archiveBytes) })
	mux.HandleFunc("/checksums", func(w http.ResponseWriter, _ *http.Request) {
		// Name doesn't match — should trigger rejection.
		_, _ = w.Write([]byte("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789  unrelated.tar.gz\n"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	plan := &Plan{
		Release:       &Release{TagName: "v2.0.0"},
		Asset:         &Asset{Name: "spwn_linux_amd64.tar.gz", DownloadURL: server.URL + "/asset"},
		ChecksumAsset: &Asset{Name: "checksums.txt", DownloadURL: server.URL + "/checksums"},
		Platform:      "linux_amd64",
	}
	target := filepath.Join(t.TempDir(), "spwn")
	_ = os.WriteFile(target, []byte("old"), 0755)
	err := Apply(context.Background(), plan, ApplyOpts{TargetPath: target})
	if err == nil {
		t.Fatal("expected refusal; got nil error")
	}
	if !contains(err.Error(), "no checksum entry") {
		t.Errorf("expected 'no checksum entry', got %v", err)
	}
}

// If someone flips bits in the archive (checksum mismatch), Apply must fail
// and must NOT replace the target binary.
func TestApply_DetectsChecksumMismatch(t *testing.T) {
	archive := makeTarGz(t, map[string][]byte{"spwn": []byte("x")})
	archiveBytes, _ := os.ReadFile(archive)

	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archiveBytes) })
	mux.HandleFunc("/checksums", func(w http.ResponseWriter, _ *http.Request) {
		// Wrong digest entirely.
		_, _ = w.Write([]byte("0000000000000000000000000000000000000000000000000000000000000000  spwn_linux_amd64.tar.gz\n"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	plan := &Plan{
		Release:       &Release{TagName: "v2.0.0"},
		Asset:         &Asset{Name: "spwn_linux_amd64.tar.gz", DownloadURL: server.URL + "/asset"},
		ChecksumAsset: &Asset{Name: "checksums.txt", DownloadURL: server.URL + "/checksums"},
		Platform:      "linux_amd64",
	}
	target := filepath.Join(t.TempDir(), "spwn")
	_ = os.WriteFile(target, []byte("OLD"), 0755)
	err := Apply(context.Background(), plan, ApplyOpts{TargetPath: target})
	if err == nil || !contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
	// Target must be untouched.
	got, _ := os.ReadFile(target)
	if string(got) != "OLD" {
		t.Errorf("target should be untouched after mismatch, got %q", got)
	}
}

func TestCheckForUpdate_DevIsAlwaysBehind(t *testing.T) {
	client := &fakeReleaseClient{latest: &Release{TagName: "v0.0.1"}}
	plan, err := CheckForUpdate(context.Background(), client, "dev", CheckOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.UpdateAvail {
		t.Errorf("dev should always show update available")
	}
}

func TestCheckForUpdate_AlreadyLatest(t *testing.T) {
	client := &fakeReleaseClient{latest: &Release{TagName: "v1.2.3"}}
	plan, err := CheckForUpdate(context.Background(), client, "v1.2.3", CheckOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if plan.UpdateAvail {
		t.Errorf("no update should be flagged when versions match")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
