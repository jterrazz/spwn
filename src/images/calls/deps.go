package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/images"
)

type Deps struct {
	Images Images
	Kind   machine.Kind
}

type Images interface {
	List(ctx context.Context) ([]images.Report, error)
	// value: builds the named images, or every one missing or stale; each
	// step's label goes to step as it starts.
	Build(ctx context.Context, names []string, binary string, step func(label string)) ([]images.Record, error)
	// value: fetches the newest published version of the named images, or
	// of every one held here.
	Update(ctx context.Context, names []string, step func(label string)) ([]images.Update, error)
	CheckDelete(ctx context.Context, image, version string) error
	Delete(ctx context.Context, image, version string) ([]images.Record, error)
}

const Unknown fail.Code = "image.unknown"

// invariant: an image the catalogue does not hold is image.unknown, whichever
// call names it.
func worded(err error) error {
	if unknown := new(images.UnknownError); errors.As(err, &unknown) {
		return &fail.Error{Type: fail.NotFound, Code: Unknown, Subject: unknown.Image, Cause: err}
	}
	return err
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(List, d), call.Bind(Build, d), call.Bind(Update, d), call.Bind(Delete, d)}
}
