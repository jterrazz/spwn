package calls

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/wiki"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
)

const (
	None        fail.Code = "wiki.none"
	UnknownPage fail.Code = "wiki.unknown-page"
	Unreadable  fail.Code = "wiki.unreadable"
	TooLarge    fail.Code = "wiki.too-large"
)

type Deps struct {
	Corpus contract.Corpus
	cache  *cache
}

type cache struct {
	// invariant: one corpus, the last read, kept by its signature; a read
	// whose signature differs compiles the corpus anew and replaces it, and
	// a compile started later is never replaced by one started earlier.
	mu        sync.Mutex
	signature string
	corpus    wiki.Corpus
	started   int
	stored    int
	flight    *flight
}

// value: the one compile of a signature in progress, which every ask of
// that signature waits for instead of compiling its own.
type flight struct {
	signature string
	n         int
	done      chan struct{}
	corpus    wiki.Corpus
	err       error
}

func All(d Deps) []call.Handler {
	// invariant: the calls of one Deps share one cache, so wiki.read and
	// wiki.map of an unchanged corpus compile it once between them.
	d.cache = &cache{}
	return []call.Handler{call.Bind(Read, d), call.Bind(Map, d)}
}

func Applies(p machine.Profile) machine.Verdict {
	// invariant: the wiki is read by the window of a Mac, from the corpus
	// that Mac declares, so a VM reads none.
	if !p.Caps.Engine {
		return machine.No(p.Ref.Name + " is a " + p.Kind.Word(p.System.OS) + ": its Mac reads the wiki")
	}
	return machine.Yes
}

type ReadIn struct {
	// value: the page's id, its path under the corpus without .md; empty
	// opens the first seat's home.
	Page string
}

var Read = call.Call[Deps, ReadIn, wiki.Served]{
	Name: "wiki.read", Where: call.InProcess, Applies: Applies,
	Errors: []fail.Code{None, UnknownPage, Unreadable, TooLarge},
	Run: func(ctx context.Context, d Deps, in ReadIn, _ call.Progress) (wiki.Served, error) {
		corpus, err := compiled(ctx, d)
		if err != nil {
			return wiki.Served{}, err
		}
		page := in.Page
		if page == "" {
			page = corpus.Home()
		}
		served, err := corpus.Serve(page)
		if errors.Is(err, wiki.ErrUnknownPage) {
			return wiki.Served{}, fail.New(fail.NotFound, UnknownPage, page, nil)
		}
		return served, err
	},
}

// invariant: the one way a call of the family reaches the compiled corpus:
// through the cache of its Deps, every refusal coded.
func compiled(ctx context.Context, d Deps) (wiki.Corpus, error) {
	corpus, err := d.cache.read(ctx, d.Corpus)
	if err != nil {
		return wiki.Corpus{}, refusal(err)
	}
	return corpus, nil
}

func (c *cache) read(ctx context.Context, corpus contract.Corpus) (wiki.Corpus, error) {
	signature, err := corpus.Signature(ctx)
	if err != nil {
		return wiki.Corpus{}, err
	}
	// invariant: one compile at a time per signature. The first ask of a
	// signature the cache does not hold compiles it under its own context;
	// every ask of it meanwhile waits for that compile and takes its corpus.
	// A waiter abandoned stops waiting; a compile its asker abandoned ends
	// with the context's error, and each waiter still asking runs it again.
	for {
		c.mu.Lock()
		if c.signature == signature {
			kept := c.corpus
			c.mu.Unlock()
			return kept, nil
		}
		f := c.flight
		if f == nil || f.signature != signature {
			c.started++
			f = &flight{signature: signature, n: c.started, done: make(chan struct{})}
			c.flight = f
			c.mu.Unlock()
			c.compile(ctx, corpus, f)
			return f.corpus, f.err
		}
		c.mu.Unlock()
		select {
		case <-f.done:
		case <-ctx.Done():
			return wiki.Corpus{}, ctx.Err()
		}
		if abandoned(f.err) && ctx.Err() == nil {
			continue
		}
		return f.corpus, f.err
	}
}

func (c *cache) compile(ctx context.Context, corpus contract.Corpus, f *flight) {
	defer close(f.done)
	files, err := corpus.Files(ctx)
	if err == nil {
		f.corpus, err = wiki.Read(ctx, files)
	}
	f.err = err
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.flight == f {
		c.flight = nil
	}
	if err == nil && f.n > c.stored {
		c.signature, c.corpus, c.stored = f.signature, f.corpus, f.n
	}
}

func abandoned(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func refusal(err error) error {
	// invariant: every way a corpus fails to read reaches the caller coded;
	// a file the folder refuses is named by its path inside the wiki only.
	var past *wiki.TooLarge
	var unread *contract.Unreadable
	switch {
	case errors.Is(err, contract.ErrNone):
		return fail.New(fail.NotFound, None, "", nil)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.As(err, &past):
		return fail.New(fail.Refused, TooLarge, "", map[string]string{"pages": strconv.Itoa(wiki.MaxPages), "mib": strconv.Itoa(wiki.MaxCorpusBytes >> 20)})
	case errors.As(err, &unread):
		heard := wiki.Audible(unread.Path)
		return fail.New(fail.Failed, Unreadable, heard, map[string]string{"path": heard})
	}
	return fail.New(fail.Failed, Unreadable, err.Error(), nil)
}
