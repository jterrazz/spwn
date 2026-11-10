package images

import "strings"

type UnknownVersionError struct{ Image, Version string }

func (e *UnknownVersionError) Error() string {
	return "image " + e.Image + " has no version " + e.Version
}

type NotBuiltError struct{ Image string }

func (e *NotBuiltError) Error() string { return "image " + e.Image + " is not built here" }

type NoLinuxBinaryError struct{}

func (e *NoLinuxBinaryError) Error() string {
	return "this station is a dev build: there is no release binary to put in the image"
}

type BinaryNotFoundError struct{ Path string }

func (e *BinaryNotFoundError) Error() string { return e.Path + ": no such file" }

type NoEngineError struct {
	Hypervisor string
	Cause      error
}

func (e *NoEngineError) Error() string { return e.Hypervisor + " is not installed or does not answer" }

func (e *NoEngineError) Unwrap() error { return e.Cause }

type NoRecipeError struct{ Image string }

func (e *NoRecipeError) Error() string {
	return "station has no recipe for " + e.Image + ": it is fetched, never built here"
}

// value: Buildable is true when station has a recipe for the image, so the
// owner's way out of "not released" is to build it on this Mac.
type NotReleasedError struct {
	Image     string
	Buildable bool
}

func (e *NotReleasedError) Error() string {
	if e.Buildable {
		return "image " + e.Image + " is not released"
	}
	return "no version of " + e.Image + " is released"
}

type RegistryError struct {
	Image string
	Cause error
}

func (e *RegistryError) Error() string {
	return "the registry did not answer for " + e.Image + ": " + e.Cause.Error()
}

func (e *RegistryError) Unwrap() error { return e.Cause }

type FetchFailedError struct {
	Image, Version, Step string
	Cause                error
}

func (e *FetchFailedError) Error() string {
	return "image " + e.Image + " " + e.Version + ": " + e.Step + " failed: " + strings.TrimSpace(e.Cause.Error())
}

func (e *FetchFailedError) Unwrap() error { return e.Cause }

type BuildFailedError struct {
	Image, Step string
	Cause       error
}

func (e *BuildFailedError) Error() string {
	message := "image " + e.Image + ": " + e.Step + " failed"
	if e.Cause != nil {
		if detail := strings.TrimSpace(e.Cause.Error()); detail != "" {
			message += ": " + detail
		}
	}
	return message
}

func (e *BuildFailedError) Unwrap() error { return e.Cause }
