module spwn.sh/packages/architect

go 1.25.0

require (
	github.com/docker/docker v27.5.1+incompatible
	gopkg.in/yaml.v3 v3.0.1
	spwn.sh/catalog v0.0.0
	spwn.sh/packages/activity v0.0.0
	spwn.sh/packages/agent v0.0.0
	spwn.sh/packages/auth v0.0.0
	spwn.sh/packages/compile v0.0.0
	spwn.sh/packages/dependency v0.0.0
	spwn.sh/packages/image v0.0.0
	spwn.sh/packages/platform v0.0.0
	spwn.sh/packages/runtimes v0.0.0
	spwn.sh/packages/world v0.0.0
)

replace (
	spwn.sh/catalog => ../../catalog
	spwn.sh/packages/activity => ../activity
	spwn.sh/packages/agent => ../agent
	spwn.sh/packages/auth => ../auth
	spwn.sh/packages/compile => ../compile
	spwn.sh/packages/dependency => ../dependency
	spwn.sh/packages/image => ../image
	spwn.sh/packages/platform => ../platform
	spwn.sh/packages/runtimes => ../runtimes
	spwn.sh/packages/world => ../world
)
