module spwn.sh/apps/cli

go 1.25.0

require (
	github.com/docker/docker v27.5.1+incompatible
	github.com/fatih/color v1.18.0
	github.com/spf13/cobra v1.10.2
	golang.org/x/term v0.40.0
	gopkg.in/yaml.v3 v3.0.1
	spwn.sh/packages/agent v0.0.0
	spwn.sh/packages/foundation v0.0.0
	spwn.sh/packages/imagebuilder v0.0.0
	spwn.sh/packages/messenger v0.0.0
	spwn.sh/packages/migration v0.0.0
	spwn.sh/packages/universe v0.0.0
	spwn.sh/examples v0.0.0
)

require (
	github.com/Microsoft/go-winio v0.4.21 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.67.0 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/metric v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.41.0 // indirect
	google.golang.org/grpc v1.79.3 // indirect
)

replace (
	spwn.sh/packages/agent => ../../packages/agent
	spwn.sh/packages/foundation => ../../packages/foundation
	spwn.sh/packages/imagebuilder => ../../packages/imagebuilder
	spwn.sh/packages/messenger => ../../packages/messenger
	spwn.sh/packages/migration => ../../packages/migration
	spwn.sh/packages/universe => ../../packages/universe
	spwn.sh/examples => ../../examples
)
