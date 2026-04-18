module spwn.sh/tests/catalog

go 1.25.0

require (
	gopkg.in/yaml.v3 v3.0.1
	spwn.sh/catalog v0.0.0
	spwn.sh/packages/compile v0.0.0
	spwn.sh/packages/dependency v0.0.0
	spwn.sh/packages/project v0.0.0
	spwn.sh/packages/runtimes v0.0.0
	spwn.sh/packages/transpile v0.0.0
)

replace (
	spwn.sh/catalog => ../../catalog
	spwn.sh/packages/compile => ../../packages/compile
	spwn.sh/packages/dependency => ../../packages/dependency
	spwn.sh/packages/project => ../../packages/project
	spwn.sh/packages/runtimes => ../../packages/runtimes
	spwn.sh/packages/transpile => ../../packages/transpile
)
