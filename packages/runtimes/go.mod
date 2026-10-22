module spwn.sh/packages/runtimes

go 1.25.0

require (
	spwn.sh/packages/dependency v0.0.0
	spwn.sh/packages/transpile v0.0.0
)

require (
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	spwn.sh/packages/dependency => ../dependency
	spwn.sh/packages/transpile => ../transpile
)
