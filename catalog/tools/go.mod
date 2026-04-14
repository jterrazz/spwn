module spwn.sh/catalog/tools

go 1.25.0

require (
	spwn.sh/catalog/runtimes v0.0.0
	spwn.sh/packages/imagebuilder v0.0.0
)

replace (
	spwn.sh/catalog/runtimes => ../runtimes
	spwn.sh/packages/imagebuilder => ../../packages/imagebuilder
)
