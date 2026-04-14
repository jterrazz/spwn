package update

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DownloadHTTP fetches a URL to dest with a 60s timeout and up to 3 retries.
// dest is created if needed. Returns the total bytes written.
func DownloadHTTP(ctx context.Context, url, dest string) (int64, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		n, err := downloadOnce(ctx, url, dest)
		if err == nil {
			return n, nil
		}
		lastErr = err
		// No retry on 4xx, only transient failures.
		if strings.Contains(err.Error(), "HTTP 4") {
			return 0, err
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return 0, fmt.Errorf("download failed after 3 attempts: %w", lastErr)
}

func downloadOnce(ctx context.Context, url, dest string) (int64, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("download: HTTP %d for %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, resp.Body)
}

// FileSHA256 returns the hex-encoded SHA256 of a file on disk.
func FileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ParseChecksums parses GoReleaser-style "checksums.txt" - one entry per line,
// `{hex-digest}  {filename}`. Returns a map filename → digest.
func ParseChecksums(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Accept either "digest  name" (two spaces) or "digest name" (one).
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("checksums line %d: want 2 fields, got %d: %q", lineNum, len(fields), line)
		}
		digest := strings.ToLower(fields[0])
		name := fields[1]
		if len(digest) != 64 {
			return nil, fmt.Errorf("checksums line %d: invalid sha256 length: %q", lineNum, digest)
		}
		out[name] = digest
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// VerifyChecksum compares the SHA256 of the file at path against expectedHex.
// Returns a descriptive error on mismatch. The comparison is case-insensitive.
func VerifyChecksum(path, expectedHex string) error {
	got, err := FileSHA256(path)
	if err != nil {
		return fmt.Errorf("hash file: %w", err)
	}
	if !strings.EqualFold(got, expectedHex) {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", path, got, expectedHex)
	}
	return nil
}
