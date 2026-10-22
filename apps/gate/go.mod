module spwn.sh/apps/gatehouse

go 1.25.0

require spwn.sh/packages/gate v0.0.0

require (
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	spwn.sh/packages/authentication v0.0.0 // indirect
	spwn.sh/packages/platform v0.0.0 // indirect
)

replace spwn.sh/packages/gate => ../../packages/gate

replace spwn.sh/packages/authentication => ../../packages/authentication

replace spwn.sh/packages/platform => ../../packages/platform
