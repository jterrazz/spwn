package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadHTTP(t *testing.T) {
	payload := []byte("hello spwn")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	n, err := DownloadHTTP(context.Background(), server.URL, dest)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(payload)) {
		t.Errorf("size = %d, want %d", n, len(payload))
	}
	got, _ := os.ReadFile(dest)
	if string(got) != string(payload) {
		t.Errorf("content mismatch: %q", got)
	}
}

func TestDownloadHTTP_404_NoRetry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(404)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "out.bin")
	_, err := DownloadHTTP(context.Background(), server.URL, dest)
	if err == nil {
		t.Fatal("expected error")
	}
	// 4xx should not retry.
	if calls != 1 {
		t.Errorf("HTTP 404 should not retry, got %d calls", calls)
	}
}

func TestFileSHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data")
	content := []byte("spwn checksum test")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	want := hex.EncodeToString(sum[:])
	got, err := FileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("SHA256 = %q, want %q", got, want)
	}
}

func TestParseChecksums(t *testing.T) {
	input := `# some comment
abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789  spwn_darwin_arm64.tar.gz
fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210  spwn_linux_amd64.tar.gz

1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef  checksums.txt
`
	got, err := ParseChecksums(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d entries: %+v", len(got), got)
	}
	if got["spwn_darwin_arm64.tar.gz"] != "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789" {
		t.Errorf("wrong digest for arm64: %q", got["spwn_darwin_arm64.tar.gz"])
	}
}

func TestParseChecksums_Malformed(t *testing.T) {
	tests := []string{
		"only-one-field\n",
		"too short  name.tar.gz\n",
		"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789 name extra\n",
	}
	for _, in := range tests {
		if _, err := ParseChecksums(strings.NewReader(in)); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}

func TestVerifyChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x")
	content := []byte("verify me")
	_ = os.WriteFile(path, content, 0644)
	sum := sha256.Sum256(content)
	good := hex.EncodeToString(sum[:])

	if err := VerifyChecksum(path, good); err != nil {
		t.Errorf("valid digest rejected: %v", err)
	}
	// Uppercase digest should match (case-insensitive).
	if err := VerifyChecksum(path, strings.ToUpper(good)); err != nil {
		t.Errorf("uppercase digest rejected: %v", err)
	}
	// Wrong digest.
	bad := "0000000000000000000000000000000000000000000000000000000000000000"
	err := VerifyChecksum(path, bad)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("expected mismatch error, got %v", err)
	}
}
