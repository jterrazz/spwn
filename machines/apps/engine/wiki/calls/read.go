package calls

import (
	"context"
	"errors"
	"slices"

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
)

type Deps struct {
	Corpus contract.Corpus
}

func All(d Deps) []call.Handler {
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
	Errors: []fail.Code{None, UnknownPage, Unreadable, BadBudget},
	Run: func(ctx context.Context, d Deps, in ReadIn, _ call.Progress) (wiki.Served, error) {
		if in.Budget != "" && !slices.Contains(wiki.Budgets(), in.Budget) {
			return wiki.Served{}, fail.New(fail.Refused, BadBudget, string(in.Budget), nil)
		}
		files, err := d.Corpus.Files(ctx)
		if errors.Is(err, contract.ErrNone) {
			return wiki.Served{}, fail.New(fail.NotFound, None, "", nil)
		}
		if err != nil {
			return wiki.Served{}, err
		}
		corpus, err := wiki.Read(files)
		if err != nil {
			return wiki.Served{}, fail.New(fail.Failed, Unreadable, err.Error(), nil)
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
