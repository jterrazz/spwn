module spwn.sh/packages/auth

go 1.25.0

require spwn.sh/packages/platform v0.0.0

require (
	github.com/kardianos/service v1.2.4 // indirect
	golang.org/x/sys v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace spwn.sh/packages/platform => ../platform
