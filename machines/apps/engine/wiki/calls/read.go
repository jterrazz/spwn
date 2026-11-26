package calls

import (
	"context"
	"errors"
	"slices"
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
	BadBudget   fail.Code = "wiki.bad-budget"
	TooLarge    fail.Code = "wiki.too-large"
)

type Deps struct {
	Corpus contract.Corpus
	cache  *cache
}

type cache struct {
	// invariant: one corpus, the last read, kept by its signature; a read
	// whose signature differs compiles the corpus anew and replaces it.
	mu        sync.Mutex
	signature string
	corpus    wiki.Corpus
}

func All(d Deps) []call.Handler {
	d.cache = &cache{}
	return []call.Handler{call.Bind(Read, d)}
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
	// value: 30s, 2min or 10min; empty reads at the corpus's own budget.
	Budget wiki.Budget
}

var Read = call.Call[Deps, ReadIn, wiki.Served]{
	Name: "wiki.read", Where: call.InProcess, Applies: Applies,
	Errors: []fail.Code{None, UnknownPage, Unreadable, BadBudget, TooLarge},
	Run: func(ctx context.Context, d Deps, in ReadIn, _ call.Progress) (wiki.Served, error) {
		if in.Budget != "" && !slices.Contains(wiki.Budgets(), in.Budget) {
			return wiki.Served{}, fail.New(fail.Refused, BadBudget, string(in.Budget), nil)
		}
		corpus, err := d.cache.read(ctx, d.Corpus)
		if err != nil {
			return wiki.Served{}, refusal(err)
		}
		page := in.Page
		if page == "" {
			page = corpus.Home()
		}
		served, err := corpus.Serve(page, in.Budget)
		if errors.Is(err, wiki.ErrUnknownPage) {
			return wiki.Served{}, fail.New(fail.NotFound, UnknownPage, page, nil)
		}
		return served, err
	},
}

func (c *cache) read(ctx context.Context, corpus contract.Corpus) (wiki.Corpus, error) {
	signature, err := corpus.Signature(ctx)
	if err != nil {
		return wiki.Corpus{}, err
	}
	c.mu.Lock()
	if c.signature == signature {
		defer c.mu.Unlock()
		return c.corpus, nil
	}
	c.mu.Unlock()
	files, err := corpus.Files(ctx)
	if err != nil {
		return wiki.Corpus{}, err
	}
	read, err := wiki.Read(ctx, files)
	if err != nil {
		return wiki.Corpus{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.signature, c.corpus = signature, read
	return read, nil
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
		return fail.New(fail.Failed, Unreadable, unread.Path, map[string]string{"path": unread.Path})
	}
	return fail.New(fail.Failed, Unreadable, err.Error(), nil)
}
