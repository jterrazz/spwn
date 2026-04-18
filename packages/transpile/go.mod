module spwn.sh/packages/transpile

go 1.25.0

require (
	gopkg.in/yaml.v3 v3.0.1
	spwn.sh/packages/project v0.0.0
	spwn.sh/packages/world v0.0.0
)

require github.com/kr/text v0.2.0 // indirect

replace (
	spwn.sh/packages/project => ../project
	spwn.sh/packages/world => ../world
)
