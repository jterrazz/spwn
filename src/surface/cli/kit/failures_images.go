package kit

import (
	"errors"

	"github.com/jterrazz/studio-station/src/images"

	"github.com/jterrazz/studio-station/src/core/fail"
)

const linuxBinaryHint = "make build-linux, then station images build linux --binary .artifacts/go/station-linux-arm64"

func imagesFailure(err error) (*fail.Error, bool) {
	var version *images.UnknownVersionError
	var notBuilt *images.NotBuiltError
	var noBinary *images.NoLinuxBinaryError
	var missing *images.BinaryNotFoundError
	var noEngine *images.NoEngineError
	var noRecipe *images.NoRecipeError
	var unpublished *images.NotReleasedError
	var registry *images.RegistryError
	var fetch *images.FetchFailedError
	var build *images.BuildFailedError
	switch {
	case errors.As(err, &version):
		return usage(version.Error(), "station images lists the versions"), true
	case errors.As(err, &notBuilt):
		return usage(notBuilt.Error(), "station images build "+notBuilt.Image), true
	case errors.As(err, &noBinary):
		return usage(noBinary.Error(), linuxBinaryHint), true
	case errors.As(err, &missing):
		return usage(missing.Error(), linuxBinaryHint), true
	case errors.As(err, &noEngine):
		return usage(noEngine.Error(), "station apps install "+noEngine.Hypervisor), true
	case errors.As(err, &noRecipe):
		return usage(noRecipe.Error(), "station images update "+noRecipe.Image), true
	case errors.As(err, &unpublished):
		if unpublished.Buildable {
			return failed(unpublished.Error(), "not released · station images build "+unpublished.Image, err), true
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
