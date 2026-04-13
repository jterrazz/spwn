// Package backend re-exports types from spwn.sh/packages/imagebuilder/backend.
// This package exists so that universe's internal code can continue importing
// "spwn.sh/packages/universe/internal/backend" without changes.
package backend

import ib "spwn.sh/packages/imagebuilder/backend"

// Type aliases — all forwarded from imagebuilder/backend.
type Backend = ib.Backend
type ContainerConfig = ib.ContainerConfig
type ExecConfig = ib.ExecConfig
type ImageInfo = ib.ImageInfo
type ContainerInfo = ib.ContainerInfo
type Docker = ib.Docker

// NewDocker creates a Docker backend from the environment.
var NewDocker = ib.NewDocker
