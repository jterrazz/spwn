package platform

import "os"

const (
	DefaultMaxProcs = 64
	DefaultBackend  = "docker"
)

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
