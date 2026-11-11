package githubrelease

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// value: the largest command an asset may hold, so an archive that
// inflates without end is refused rather than read.
const most = 512 << 20

type entry struct {
	name       string
	executable bool
	data       []byte
}

// invariant: each command is the one regular file of the asset by that
// name (the executable one, where several are), else the asset itself
// when it is no archive and lands one command; nothing else of the asset
// is written.
func unpack(asset string, body []byte, cmds []string) (map[string][]byte, error) {
	entries, archive, err := entries(asset, body)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	if !archive {
		if len(cmds) != 1 {
			return nil, fmt.Errorf("%s is one file, and lands %d commands", asset, len(cmds))
		}
		out[cmds[0]] = body
		return out, nil
	}
	for _, cmd := range cmds {
		var found []entry
		for _, e := range entries {
			if path.Base(e.name) == cmd {
				found = append(found, e)
			}
		}
		if len(found) > 1 {
			var run []entry
			for _, e := range found {
				if e.executable {
					run = append(run, e)
				}
			}
			found = run
		}
		switch len(found) {
		case 0:
			return nil, fmt.Errorf("%s holds no %s", asset, cmd)
		case 1:
			out[cmd] = found[0].data
		default:
			return nil, fmt.Errorf("%s holds %d files named %s", asset, len(found), cmd)
		}
	}
	return out, nil
}

func entries(asset string, body []byte) ([]entry, bool, error) {
	lower := strings.ToLower(asset)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		z, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, true, fmt.Errorf("%s: %w", asset, err)
		}
		es, err := untar(z)
		return es, true, wrap(asset, err)
	case strings.HasSuffix(lower, ".tar.bz2"), strings.HasSuffix(lower, ".tbz"):
		es, err := untar(bzip2.NewReader(bytes.NewReader(body)))
		return es, true, wrap(asset, err)
	case strings.HasSuffix(lower, ".tar"):
		es, err := untar(bytes.NewReader(body))
		return es, true, wrap(asset, err)
	case strings.HasSuffix(lower, ".zip"):
		es, err := unzip(body)
		return es, true, wrap(asset, err)
	case strings.HasSuffix(lower, ".tar.xz"), strings.HasSuffix(lower, ".txz"), strings.HasSuffix(lower, ".tar.zst"), strings.HasSuffix(lower, ".7z"), strings.HasSuffix(lower, ".deb"), strings.HasSuffix(lower, ".rpm"), strings.HasSuffix(lower, ".dmg"), strings.HasSuffix(lower, ".pkg"):
		return nil, true, fmt.Errorf("%s is an archive station does not open: its method names a .tar.gz, .zip or bare asset", asset)
	}
	return nil, false, nil
}

func wrap(asset string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", asset, err)
	}
	return nil
}

func untar(r io.Reader) ([]entry, error) {
	var out []entry
	t := tar.NewReader(r)
	for {
		h, err := t.Next()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		data, err := bounded(t)
		if err != nil {
			return nil, err
		}
		out = append(out, entry{name: h.Name, executable: h.Mode&0o111 != 0, data: data})
	}
}

func unzip(body []byte) ([]entry, error) {
	z, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	var out []entry
	for _, f := range z.File {
		if !f.Mode().IsRegular() {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := bounded(r)
		_ = r.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, entry{name: f.Name, executable: f.Mode()&0o111 != 0, data: data})
	}
	return out, nil
}

func bounded(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, most+1))
	if err != nil {
		return nil, err
	}
	if len(data) > most {
		return nil, errors.New("a file past 512 MiB")
	}
	return data, nil
}

type NoReleaseError struct{ Repo, Tag string }

func (e *NoReleaseError) Error() string {
	if e.Tag == "" {
		return e.Repo + " has no release"
	}
	return e.Repo + " has no release " + e.Tag
}

type NoAssetError struct{ Repo, Tag, Pattern, OS, Arch string }

func (e *NoAssetError) Error() string {
	return e.Repo + " " + e.Tag + " has no asset for " + e.OS + "/" + e.Arch + " (" + e.Pattern + ")"
}

type UnverifiedError struct{ Repo, Tag, Asset string }

func (e *UnverifiedError) Error() string {
	return e.Repo + " " + e.Tag + " gives no checksum for " + e.Asset + ": nothing installed"
}

type MismatchError struct{ Asset, Source, Want, Got string }

func (e *MismatchError) Error() string {
	return e.Asset + " does not match " + e.Source + " (sha256 " + e.Got + ", not " + e.Want + "): nothing installed"
}

type NotOursError struct{ Path string }

func (e *NotOursError) Error() string {
	return e.Path + " is not station's: kept, nothing installed"
}

type SelfError struct{ Repo string }

func (e *SelfError) Error() string {
	return e.Repo + " is station's own release: its install line updates it"
}

type RecordError struct {
	File string
	Err  error
}

func (e *RecordError) Error() string {
	return e.File + " is no record station reads: " + e.Err.Error() + "; nothing installed nor removed"
}

func (e *RecordError) Unwrap() error { return e.Err }
