module spwn.sh/packages/automation

go 1.25.0

require (
	github.com/robfig/cron/v3 v3.0.1
	spwn.sh/packages/project v0.0.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace spwn.sh/packages/project => ../project
