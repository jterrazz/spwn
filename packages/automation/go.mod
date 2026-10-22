module spwn.sh/packages/automation

go 1.25.0

require (
	github.com/fsnotify/fsnotify v1.10.1
	github.com/robfig/cron/v3 v3.0.1
	spwn.sh/packages/project v0.0.0
)

require (
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace spwn.sh/packages/project => ../project
