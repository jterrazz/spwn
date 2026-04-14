package base

import _ "embed"

//go:embed world.Dockerfile
var WorldDockerfile []byte

//go:embed architect.Dockerfile
var ArchitectDockerfile []byte

//go:embed test.Dockerfile
var TestDockerfile []byte

//go:embed entrypoint.sh
var ArchitectEntrypoint []byte
