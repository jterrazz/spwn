module spwn.sh/apps/cli

go 1.25.0

require (
	github.com/docker/docker v27.5.1+incompatible
	github.com/fatih/color v1.18.0
	github.com/spf13/cobra v1.10.2
	golang.org/x/term v0.40.0
	gopkg.in/yaml.v3 v3.0.1
	spwn.sh/catalog v0.0.0
	spwn.sh/packages/activity v0.0.0
	spwn.sh/packages/agent v0.0.0
	spwn.sh/packages/auth v0.0.0
	spwn.sh/packages/transpile v0.0.0
	spwn.sh/packages/compile v0.0.0
	spwn.sh/packages/mailbox v0.0.0
	spwn.sh/packages/platform v0.0.0
	spwn.sh/packages/project v0.0.0
	spwn.sh/packages/world v0.0.0
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
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
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260209200024-4cfbd4190f57 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260209200024-4cfbd4190f57 // indirect
	spwn.sh/packages/architect v0.0.0
	spwn.sh/packages/migration v0.0.0
	spwn.sh/packages/runtimes v0.0.0
	spwn.sh/packages/update v0.0.0
)

replace (
	spwn.sh/catalog => ../../catalog
	spwn.sh/packages/activity => ../../packages/activity
	spwn.sh/packages/agent => ../../packages/agent
	spwn.sh/packages/auth => ../../packages/auth
	spwn.sh/packages/transpile => ../../packages/transpile
	spwn.sh/packages/compile => ../../packages/compile
	spwn.sh/packages/mailbox => ../../packages/mailbox
	spwn.sh/packages/platform => ../../packages/platform
	spwn.sh/packages/project => ../../packages/project
	spwn.sh/packages/world => ../../packages/world
)

replace spwn.sh/packages/runtimes => ../../packages/runtimes

replace spwn.sh/packages/architect => ../../packages/architect

replace spwn.sh/packages/migration => ../../packages/migration

replace spwn.sh/packages/update => ../../packages/update
