module spwn.sh/apps/cli

go 1.25.0

require (
	github.com/fatih/color v1.19.0
	github.com/spf13/cobra v1.10.2
	golang.org/x/term v0.45.0
	gopkg.in/yaml.v3 v3.0.1
	spwn.sh/packages/activity v0.0.0
	spwn.sh/packages/agent v0.0.0
	spwn.sh/packages/authentication v0.0.0
	spwn.sh/packages/compile v0.0.0
	spwn.sh/packages/platform v0.0.0
	spwn.sh/packages/project v0.0.0
	spwn.sh/packages/transpile v0.0.0
	spwn.sh/packages/world v0.0.0
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20261006220739-8c912ac31dd4 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/moby/moby/api v1.56.1 // indirect
	github.com/moby/moby/client v0.6.2 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.42.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	google.golang.org/grpc v1.83.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.8.2 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.16 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/pflag v1.0.10
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.71.0 // indirect
	go.opentelemetry.io/otel v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	spwn.sh/packages/architect v0.0.0
	spwn.sh/packages/automation v0.0.0
	spwn.sh/packages/gate v0.0.0
	spwn.sh/packages/migration v0.0.0
	spwn.sh/packages/runtimes v0.0.0
	spwn.sh/packages/upgrade v0.0.0
)

replace (
	spwn.sh/packages/activity => ../../packages/activity
	spwn.sh/packages/agent => ../../packages/agent
	spwn.sh/packages/authentication => ../../packages/authentication
	spwn.sh/packages/compile => ../../packages/compile
	spwn.sh/packages/platform => ../../packages/platform
	spwn.sh/packages/project => ../../packages/project
	spwn.sh/packages/transpile => ../../packages/transpile
	spwn.sh/packages/world => ../../packages/world
)

replace spwn.sh/packages/runtimes => ../../packages/runtimes

replace spwn.sh/packages/architect => ../../packages/architect

replace spwn.sh/packages/automation => ../../packages/automation

replace spwn.sh/packages/migration => ../../packages/migration

replace spwn.sh/packages/upgrade => ../../packages/upgrade

replace spwn.sh/packages/gate => ../../packages/gate
