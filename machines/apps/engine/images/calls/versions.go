package calls

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/images"
)

func (s *Builder) Latest(ctx context.Context, image string) (images.Record, error) {
	img, err := s.image(ctx, image)
	var none *images.NoRecipeError
	if err != nil && !errors.As(err, &none) {
		return images.Record{}, err
	}
	all, err := s.records(img)
	if err != nil {
		return images.Record{}, err
	}
	if len(all) == 0 {
		return images.Record{}, &images.NotBuiltError{Image: image}
	}
	return all[0], nil
}
