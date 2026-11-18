package calls

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/images"
)

const (
	releaseAsset = "station-linux-arm64"
	releaseBase  = "https://github.com/jterrazz/studio-station/releases/download/"
)

var releaseTag = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

func (s *Builder) needsBinary(given string) error {
	if given == "" && !releaseTag.MatchString(s.o.Version) {
		return &images.NoLinuxBinaryError{}
	}
	return nil
}

func (s *Builder) binary(ctx context.Context, image, given string) (string, error) {
	if given != "" {
		if !s.m.Exists(given) {
			return "", &images.BinaryNotFoundError{Path: given}
		}
		return given, nil
	}
	return s.download(ctx, image)
}

func (s *Builder) download(ctx context.Context, image string) (string, error) {
	cached := path.Join(s.o.Cache, releaseAsset+"-"+s.o.Version)
	if s.m.Exists(cached) {
		return cached, nil
	}
	fail := func(cause error) error {
		return &images.BuildFailedError{Image: image, Step: "fetching " + releaseAsset + " " + s.o.Version, Cause: cause}
	}
	base := releaseBase + s.o.Version + "/"
	status, sums, err := s.m.Get(ctx, base+"checksums.txt")
	if err != nil || status != 200 {
		return "", fail(statusError(base+"checksums.txt", status, err))
	}
	want := checksumOf(string(sums), releaseAsset)
	if want == "" {
		return "", fail(fmt.Errorf("checksums.txt lists no %s", releaseAsset))
	}
	status, body, err := s.m.Get(ctx, base+releaseAsset)
	if err != nil || status != 200 {
		return "", fail(statusError(base+releaseAsset, status, err))
	}
	sum := sha256.Sum256(body)
	if got := hex.EncodeToString(sum[:]); got != want {
		return "", fail(fmt.Errorf("checksum mismatch: checksums.txt says %s, the download is %s", want, got))
	}
	if err := s.m.Keep(cached, body); err != nil {
		return "", fail(err)
	}
	return cached, nil
}

func statusError(url string, status int, err error) error {
	if err != nil {
		return err
	}
	return fmt.Errorf("%s answered %d", url, status)
}

func checksumOf(sums, asset string) string {
	for line := range strings.SplitSeq(sums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			return strings.ToLower(fields[0])
		}
	}
	return ""
}
