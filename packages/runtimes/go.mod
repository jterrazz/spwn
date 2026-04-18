module spwn.sh/packages/runtimes

go 1.25.0

require (
	spwn.sh/packages/dependency v0.0.0
	spwn.sh/packages/transpile v0.0.0
)

replace (
	spwn.sh/packages/dependency => ../dependency
	spwn.sh/packages/transpile => ../transpile
)
