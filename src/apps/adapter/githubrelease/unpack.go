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
	"slices"
	"strings"
)

// value: what an asset may hold: each command at most, and its archive
// read through at most, all its files together, so an archive that
// inflates without end is refused rather than read.
type bounds struct{ command, archive int64 }

var within = bounds{command: 512 << 20, archive: 2 << 30}

type entry struct {
	name       string
	executable bool
	data       []byte
}

// invariant: each command is the one regular file of the asset by that
// name (the executable one, where several are), and a runnable one; else
// the asset itself when it is no archive, nor a compressed file, and lands
// one command. Only those files are read into memory, and nothing else of
// the asset is written.
func unpack(asset string, body []byte, cmds []string) (map[string][]byte, error) {
	return unpackWithin(asset, body, cmds, within)
}

func unpackWithin(asset string, body []byte, cmds []string, b bounds) (map[string][]byte, error) {
	entries, archive, err := entries(asset, body, cmds, b)
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
		var run []entry
		for _, e := range found {
			if e.executable {
				run = append(run, e)
			}
		}
		switch {
		case len(found) == 0:
			return nil, fmt.Errorf("%s holds no %s", asset, cmd)
		case len(run) == 0:
			return nil, fmt.Errorf("%s holds %s, but no runnable one", asset, cmd)
		case len(run) > 1:
			return nil, fmt.Errorf("%s holds %d files named %s", asset, len(run), cmd)
		}
		out[cmd] = run[0].data
	}
	return out, nil
}

func entries(asset string, body []byte, cmds []string, b bounds) ([]entry, bool, error) {
	lower := strings.ToLower(asset)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		z, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, true, fmt.Errorf("%s: %w", asset, err)
		}
		es, err := untar(z, cmds, b)
		return es, true, wrap(asset, err)
	case strings.HasSuffix(lower, ".tar.bz2"), strings.HasSuffix(lower, ".tbz"):
		es, err := untar(bzip2.NewReader(bytes.NewReader(body)), cmds, b)
		return es, true, wrap(asset, err)
	case strings.HasSuffix(lower, ".tar"):
		es, err := untar(bytes.NewReader(body), cmds, b)
		return es, true, wrap(asset, err)
	case strings.HasSuffix(lower, ".zip"):
		es, err := unzip(body, cmds, b)
		return es, true, wrap(asset, err)
	case strings.HasSuffix(lower, ".tar.xz"), strings.HasSuffix(lower, ".txz"), strings.HasSuffix(lower, ".tar.zst"), strings.HasSuffix(lower, ".7z"), strings.HasSuffix(lower, ".deb"), strings.HasSuffix(lower, ".rpm"), strings.HasSuffix(lower, ".dmg"), strings.HasSuffix(lower, ".pkg"):
		return nil, true, fmt.Errorf("%s is an archive station does not open: its method names a .tar.gz, .zip or bare asset", asset)
	case strings.HasSuffix(lower, ".gz"), strings.HasSuffix(lower, ".bz2"), strings.HasSuffix(lower, ".xz"), strings.HasSuffix(lower, ".zst"), strings.HasSuffix(lower, ".lz4"), strings.HasSuffix(lower, ".z"):
		return nil, true, fmt.Errorf("%s is a compressed file, no command: its method names a .tar.gz, .zip or bare asset", asset)
	}
	return nil, false, nil
}

func wrap(asset string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", asset, err)
	}
	return nil
}

// value: a reader that fails past its bound, where a plain limit would end
// the archive quietly.
type capped struct {
	r    io.Reader
	left int64
	most int64
}

func (c *capped) Read(p []byte) (int, error) {
	if c.left <= 0 {
		var one [1]byte
		if n, err := c.r.Read(one[:]); n == 0 && errors.Is(err, io.EOF) {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("an archive past %d MiB", c.most>>20)
	}
	if int64(len(p)) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= int64(n)
	return n, err
}

func untar(r io.Reader, cmds []string, b bounds) ([]entry, error) {
	var out []entry
	t := tar.NewReader(&capped{r: r, left: b.archive, most: b.archive})
	for {
		h, err := t.Next()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg || !slices.Contains(cmds, path.Base(h.Name)) {
			continue
		}
		data, err := bounded(t, b.command)
		if err != nil {
			return nil, err
		}
		out = append(out, entry{name: h.Name, executable: h.Mode&0o111 != 0, data: data})
	}
}

func unzip(body []byte, cmds []string, b bounds) ([]entry, error) {
	z, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	var out []entry
	for _, f := range z.File {
		if !f.Mode().IsRegular() || !slices.Contains(cmds, path.Base(f.Name)) {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := bounded(r, b.command)
		_ = r.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, entry{name: f.Name, executable: f.Mode()&0o111 != 0, data: data})
	}
	return out, nil
}

func bounded(r io.Reader, most int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, most+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > most {
		return nil, fmt.Errorf("a file past %d MiB", most>>20)
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
