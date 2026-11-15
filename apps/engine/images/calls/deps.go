package calls

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/images"
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

const (
	Unknown        fail.Code = "image.unknown"
	NoLinuxBinary  fail.Code = "image.no-linux-binary"
	BinaryNotFound fail.Code = "image.binary-not-found"
	NoRecipe       fail.Code = "image.no-recipe"
	NoHypervisor   fail.Code = "image.no-hypervisor"
	InUse          fail.Code = "image.in-use"
)

// value: the refusals a build declares, each coded by Worded; a missing
// hypervisor is its need, refused before it runs.
func buildRefusals() []fail.Code {
	return []fail.Code{Unknown, NoLinuxBinary, BinaryNotFound, NoRecipe}
}

func Worded(err error) error {
	// invariant: an image the catalogue does not hold is image.unknown,
	// whichever call names it, a create's build included; each refusal of a
	// build is its own code.
	coded := func(t fail.Type, code fail.Code, subject string) error {
		return &fail.Error{Type: t, Code: code, Subject: subject, Cause: err}
	}
	var (
		unknown  *images.UnknownError
		noBinary *images.NoLinuxBinaryError
		missing  *images.BinaryNotFoundError
		noRecipe *images.NoRecipeError
		noEngine *images.NoEngineError
		inUse    *images.InUseError
	)
	switch {
	case errors.As(err, &unknown):
		return coded(fail.NotFound, Unknown, unknown.Image)
	case errors.As(err, &noBinary):
		return coded(fail.Refused, NoLinuxBinary, "")
	case errors.As(err, &missing):
		return coded(fail.Refused, BinaryNotFound, missing.Path)
	case errors.As(err, &noRecipe):
		return coded(fail.Refused, NoRecipe, noRecipe.Image)
	case errors.As(err, &noEngine):
		return coded(fail.Refused, NoHypervisor, noEngine.Hypervisor)
	case errors.As(err, &inUse):
		return &fail.Error{Type: fail.Refused, Code: InUse, Subject: inUse.Image, Cause: err, Detail: map[string]string{
			"version": inUse.Version, "building": strconv.FormatBool(inUse.Building), "newest": strconv.FormatBool(inUse.Newest), "vms": strings.Join(inUse.VMs, ","),
		}}
	}
	return err
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(List, d), call.Bind(Build, d), call.Bind(Update, d), call.Bind(Delete, d)}
}
