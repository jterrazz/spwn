package images

import _ "embed"

//go:embed Dockerfile
var Dockerfile []byte

//go:embed Dockerfile.god
var DockerfileGod []byte
