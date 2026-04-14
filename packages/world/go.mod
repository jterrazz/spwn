module spwn.sh/packages/world

go 1.25.0

require (
	github.com/docker/docker v27.5.1+incompatible
	gopkg.in/yaml.v3 v3.0.1
	spwn.sh/packages/activity v0.0.0
	spwn.sh/packages/agent v0.0.0
	spwn.sh/packages/auth v0.0.0
	spwn.sh/packages/base v0.0.0
	spwn.sh/packages/catalog v0.0.0
	spwn.sh/packages/ids v0.0.0
	spwn.sh/packages/image v0.0.0
	spwn.sh/packages/paths v0.0.0
	spwn.sh/packages/version v0.0.0
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.67.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.40.0 // indirect
)

replace (
	spwn.sh/packages/activity => ../activity
	spwn.sh/packages/agent => ../agent
	spwn.sh/packages/auth => ../auth
	spwn.sh/packages/base => ../base
	spwn.sh/packages/catalog => ../catalog
	spwn.sh/packages/ids => ../ids
	spwn.sh/packages/image => ../image
	spwn.sh/packages/paths => ../paths
	spwn.sh/packages/version => ../version
)
