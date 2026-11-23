package images

import "strings"

type UnknownVersionError struct{ Image, Version string }

func (e *UnknownVersionError) Error() string {
	return "image " + e.Image + " has no version " + e.Version
}

type InUseError struct {
	// value: a version Delete refuses, by the rule of Report.Unused: a build
	// is making it, it is the newest, or machines were made from it.
	Image, Version   string
	Building, Newest bool
	VMs              []string
}

func (e *InUseError) Error() string {
	said := "image " + e.Image + " " + e.Version
	switch {
	case e.Building:
		return said + " is being built"
	case e.Newest:
		return said + " is the newest: the next machines are made from it"
	}
	names := strings.Join(e.VMs, ", ")
	if n := len(e.VMs); n > 1 {
		names = strings.Join(e.VMs[:n-1], ", ") + " and " + e.VMs[n-1]
	}
	return said + " has machines made from it: " + names
}

// value: a record of a version that names no distro this station knows:
// an older station wrote it, and it is read no more; deleting the version
// is the way out.

type OldRecordError struct{ Image, Version, File string }

func (e *OldRecordError) Error() string {
	return e.File + " is a record an older station wrote: it names no distro"
}

type NotBuiltError struct {
	Image string
	// value: Machines only fetches the image: it has no recipe to build it.
	Fetched bool
}

func (e *NotBuiltError) Error() string {
	if e.Fetched {
		return "image " + e.Image + " is not on this Mac"
	}
	return "image " + e.Image + " is not built here"
}

type NoLinuxBinaryError struct{}

func (e *NoLinuxBinaryError) Error() string {
	return "this station is not a released version: there is no release binary to put in the image"
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
	return "Machines has no recipe for " + e.Image + ": it is fetched, never built here"
}

type NotReleasedError struct {
	Image string
	// value: Buildable is true when station has a recipe for the image, so the
	// owner's way out of "not released" is to build it on this Mac.
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
