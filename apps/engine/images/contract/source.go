package contract

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/images"
)

var (
	// value: no published version of that image.
	ErrNotReleased = errors.New("not released")
	// value: the image is published, never for that hypervisor.
	ErrNotForHypervisor = errors.New("not released for this hypervisor")
)

type ImageSource interface {
	// invariant: the newest published version of the image; an image never
	// published answers ErrNotPublished.
	Latest(ctx context.Context, name string) (images.Ref, error)
	// invariant: where the hypervisor fetches that version; a hypervisor the
	// image is not published for answers ErrNotForHypervisor.
	Locate(ctx context.Context, r images.Ref, hypervisor string) (ImageLocation, error)
}
