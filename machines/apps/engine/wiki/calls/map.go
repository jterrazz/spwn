package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/wiki"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
)

type MapIn struct {
	// value: the page whose sections and items the map holds, its id; empty
	// holds every seat and page with none open.
	Open string
}

var Map = call.Call[Deps, MapIn, wiki.Map]{
	Name: "wiki.map", Where: call.InProcess, Applies: Applies,
	Errors: []fail.Code{None, UnknownPage, Unreadable},
	Run: func(ctx context.Context, d Deps, in MapIn, _ call.Progress) (wiki.Map, error) {
		corpus, err := compiled(ctx, d)
		if err != nil {
			return wiki.Map{}, err
		}
		out, err := corpus.Map(in.Open)
		if errors.Is(err, wiki.ErrUnknownPage) {
			return wiki.Map{}, fail.New(fail.NotFound, UnknownPage, in.Open, nil)
		}
		return out, err
	},
}

// invariant: the one way a call of the family reaches the compiled corpus,
// its refusals coded as wiki.read codes them.
func compiled(ctx context.Context, d Deps) (wiki.Corpus, error) {
	files, err := d.Corpus.Files(ctx)
	if errors.Is(err, contract.ErrNone) {
		return wiki.Corpus{}, fail.New(fail.NotFound, None, "", nil)
	}
	if err != nil {
		return wiki.Corpus{}, err
	}
	corpus, err := wiki.Read(files)
	if err != nil {
		return wiki.Corpus{}, fail.New(fail.Failed, Unreadable, err.Error(), nil)
	}
	return corpus, nil
}
