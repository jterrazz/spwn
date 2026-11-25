package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/images"
)

type ImageStore interface {
	// invariant: a version never put answers store.ErrNotFound, and one
	// whose record names no distro this station knows answers
	// images.OldRecordError, from Get and from List alike.
	Get(ctx context.Context, image, version string) (images.Record, error)
	// invariant: putting a version again replaces its record.
	Put(ctx context.Context, r images.Record) error
	// invariant: every version of the image, the newest first.
	List(ctx context.Context, image string) ([]images.Record, error)
	// invariant: every image name that holds a version on disk, readable or
	// not, in name order: the names a catalog or a Machinefile no longer
	// holds too.
	Names(ctx context.Context) ([]string, error)
	// invariant: deleting a version never put changes nothing.
	Delete(ctx context.Context, image, version string) error
}
