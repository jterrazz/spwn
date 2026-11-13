package kit

import (
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/images"
	imagescalls "github.com/jterrazz/studio-station/apps/engine/images/calls"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

func imagesFailure(err error) (*fail.Error, bool) {
	var version *images.UnknownVersionError
	var notBuilt *images.NotBuiltError
	var unpublished *images.NotReleasedError
	var registry *images.RegistryError
	var fetch *images.FetchFailedError
	var build *images.BuildFailedError
	var template *images.NotTemplateError
	var noBinary *images.NoLinuxBinaryError
	var missing *images.BinaryNotFoundError
	var noEngine *images.NoEngineError
	var noRecipe *images.NoRecipeError
	switch {
	case errors.As(err, &noBinary), errors.As(err, &missing), errors.As(err, &noEngine), errors.As(err, &noRecipe):
		return Translate(imagescalls.Worded(err), ""), true
	case errors.As(err, &template):
		return usage(template.Error(), "station explain --machinefile "+template.Image), true
	case errors.As(err, &version):
		return usage(version.Error(), "station images lists the versions"), true
	case errors.As(err, &notBuilt):
		return usage(notBuilt.Error(), "station images build "+notBuilt.Image), true
	case errors.As(err, &unpublished):
		if unpublished.Buildable {
			return failed(unpublished.Error(), "station images build "+unpublished.Image, err), true
		}
		return failed(unpublished.Error(), "station images", err), true
	case errors.As(err, &registry):
		return failed(registry.Error(), "station images update "+registry.Image, err), true
	case errors.As(err, &fetch):
		return failed(fetch.Error(), "station images update "+fetch.Image, err), true
	case errors.As(err, &build):
		return failed(build.Error(), "station images build "+build.Image, err), true
	}
	return nil, false
}
