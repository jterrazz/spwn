package calls

import (
	"context"
	"errors"

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
)

type Deps struct {
	Corpus contract.Corpus
}

func All(d Deps) []call.Handler {
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
	Errors: []fail.Code{None, UnknownPage, Unreadable},
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
