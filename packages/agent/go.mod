module spwn.sh/packages/agent

go 1.25.0

require (
	gopkg.in/yaml.v3 v3.0.1
	spwn.sh/packages/activity v0.0.0
	spwn.sh/packages/platform v0.0.0
)

require github.com/kr/text v0.2.0 // indirect

replace (
	spwn.sh/packages/activity => ../activity
	spwn.sh/packages/platform => ../platform
)
