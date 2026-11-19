package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/images"
)

type BuildMarks interface {
	// invariant: what the Mac knows of the builds it runs, one for each image:
	// a mark replaces the image's earlier one.
	Mark(ctx context.Context, m images.Making) error
	// invariant: clearing an image with no mark changes nothing.
	Clear(ctx context.Context, image string) error
	// invariant: every mark, by image.
	Marked(ctx context.Context) ([]images.Making, error)
}
