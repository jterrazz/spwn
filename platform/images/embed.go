package images

import _ "embed"

//go:embed Dockerfile
var Dockerfile []byte

//go:embed Dockerfile.world
var DockerfileWorld []byte

//go:embed Dockerfile.architect
var DockerfileArchitect []byte

//go:embed architect-entrypoint.sh
var ArchitectEntrypoint []byte
