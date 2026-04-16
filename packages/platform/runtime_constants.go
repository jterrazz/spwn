package platform

import "os"

// Default runtime constants.
//
// These live in paths/ rather than world/ because architect (a
// sub-package of world) needs them, and having them in world/ would
// create an import cycle with world/architect. paths/ is the natural
// lowest-layer home for "runtime naming conventions" alongside the
// directory-name constants.
const (
	DefaultMaxProcs = 64
	DefaultBackend  = "docker"
)

// Runtime image identifiers.
const (
	WorldImage            = "spwn/world:latest"
	ArchitectImage        = "spwn/architect:latest"
	ArchitectImageVersion = "1.1.0"
	ArchitectMount        = "/home/spwn/.spwn"
)

// ArchitectContainerName returns the name used for the always-on
// architect container. Derived from the SPWN_ARCHITECT_CONTAINER_SUFFIX
// env var so parallel test runs with different fake SPWN_HOMEs get
// separate containers.
func ArchitectContainerName() string {
	suffix := os.Getenv("SPWN_ARCHITECT_CONTAINER_SUFFIX")
	if suffix == "" {
		return "spwn-architect"
	}
	return "spwn-architect-" + suffix
}
