package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/images"
)

type ImageStore interface {
	// invariant: a version never put answers store.ErrNotFound.
	Get(ctx context.Context, image, version string) (images.Record, error)
	// invariant: putting a version again replaces its record.
	Put(ctx context.Context, r images.Record) error
	// invariant: every version of the image, the newest first.
	List(ctx context.Context, image string) ([]images.Record, error)
	// invariant: deleting a version never put changes nothing.
	Delete(ctx context.Context, image, version string) error
}
