package accounts

import (
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

func Fielded(err error) error {
	// invariant: a refusal about an input of the request names it, whichever
	// adapter raised it.
	var e *fail.Error
	if !errors.As(err, &e) || e.Field != "" {
		return err
	}
	switch e.Code {
	case VaultLabelTaken:
		e.Field = fail.FieldLabel
	case VaultEmptyValue:
		e.Field = fail.FieldValue
	}
	return e
}
