package baseimages

import _ "embed"

//go:embed world.Dockerfile
var WorldDockerfile []byte

//go:embed test.Dockerfile
var TestDockerfile []byte
