module spwn.sh/packages/runtimes

go 1.25.0

require (
	spwn.sh/packages/dependency v0.0.0
	spwn.sh/packages/compile v0.0.0
	spwn.sh/packages/platform v0.0.0
)

replace (
	spwn.sh/packages/dependency => ../dependency
	spwn.sh/packages/compile => ../compile
	spwn.sh/packages/platform => ../platform
)
