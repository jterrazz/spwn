package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"path/filepath"
	"strconv"

	"spwn.sh/packages/platform"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
)

type Corpus struct {
	// value: a corpus read from a folder of this machine's disk; an empty
	// root is a machine that declares none.
	root string
}

var _ contract.Corpus = (*Corpus)(nil)

func New(root string) *Corpus { return &Corpus{root: root} }

type reading struct {
	tree  *platform.Tree
	out   map[string]wiki.File
	pages int
	bytes int64
}

func (c *Corpus) Files(ctx context.Context) (map[string]wiki.File, error) {
	if c.root == "" {
		return nil, contract.ErrNone
	}
	// invariant: every read of the corpus goes through one platform.Tree of
	// its root, so a page that is a link out of the folder, a ".." or a FIFO
	// is refused before anything outside the corpus is opened or waited on.
	// A file is sized before it is read, and the walk stops once the corpus
	// is past its limits, so no read holds more than the limits allow.
	r := &reading{tree: platform.Within(c.root), out: map[string]wiki.File{}}
	defer func() { _ = r.tree.Close() }()
	walked := r.tree.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return &contract.Unreadable{Path: filepath.ToSlash(p), Err: err}
		}
		if stopped := ctx.Err(); stopped != nil {
			return stopped
		}
		return r.take(p, d)
	})
	if walked != nil {
		return nil, walked
	}
	return r.out, nil
}

func (c *Corpus) Signature(ctx context.Context) (string, error) {
	if c.root == "" {
		return "", contract.ErrNone
	}
	tree := platform.Within(c.root)
	defer func() { _ = tree.Close() }()
	h := sha256.New()
	files := 0
	walked := tree.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return &contract.Unreadable{Path: filepath.ToSlash(p), Err: err}
		}
		if stopped := ctx.Err(); stopped != nil {
			return stopped
		}
		if read, skip := skipped(p, d); !read {
			return skip
		}
		info, err := tree.Stat(p)
		if err != nil {
			return &contract.Unreadable{Path: filepath.ToSlash(p), Err: err}
		}
		files++
		if files > wiki.MaxPages+1 {
			return &wiki.TooLarge{Pages: files}
		}
		_, _ = h.Write([]byte(filepath.ToSlash(p) + " " + strconv.FormatInt(info.Size(), 10) + " " + strconv.FormatInt(info.ModTime().UnixNano(), 10) + "\n"))
		return nil
	})
	if walked != nil {
		return "", walked
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// value: whether the walk reads the file at p, and else what it does there:
// a hidden folder skipped, any other folder or file passed.
func skipped(p string, d fs.DirEntry) (bool, error) {
	slash := filepath.ToSlash(p)
	if d.IsDir() {
		if slash != "." && !contract.Reads(slash+"/page.md") {
			return false, fs.SkipDir
		}
		return false, nil
	}
	return contract.Reads(slash), nil
}

func (r *reading) take(p string, d fs.DirEntry) error {
	slash := filepath.ToSlash(p)
	if read, skip := skipped(p, d); !read {
		return skip
	}
	if slash != "wiki.yaml" {
		r.pages++
	}
	f, err := r.tree.Open(p)
	if err != nil {
		return &contract.Unreadable{Path: slash, Err: err}
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return &contract.Unreadable{Path: slash, Err: err}
	}
	if info.Size() > wiki.MaxPageBytes {
		r.out[slash] = wiki.File{Size: info.Size()}
		return r.within()
	}
	// invariant: the read stops one byte past the limit, so a file that
	// grows after it was sized is never held whole.
	body, err := io.ReadAll(io.LimitReader(f, wiki.MaxPageBytes+1))
	if err != nil {
		return &contract.Unreadable{Path: slash, Err: err}
	}
	size := max(info.Size(), int64(len(body)))
	if size > wiki.MaxPageBytes {
		body = nil
	}
	r.bytes += int64(len(body))
	r.out[slash] = wiki.File{Size: size, Body: body}
	return r.within()
}

func (r *reading) within() error {
	if wiki.Past(r.pages, r.bytes) {
		return &wiki.TooLarge{Pages: r.pages, Bytes: r.bytes}
	}
	return nil
}
