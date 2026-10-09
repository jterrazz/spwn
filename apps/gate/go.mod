module spwn.sh/apps/gatehouse

go 1.25.0

require spwn.sh/packages/gate v0.0.0

require (
	gopkg.in/yaml.v3 v3.0.1 // indirect
	spwn.sh/packages/authentication v0.0.0 // indirect
	spwn.sh/packages/platform v0.0.0 // indirect
)

replace spwn.sh/packages/gate => ../../packages/gate

replace spwn.sh/packages/authentication => ../../packages/authentication

replace spwn.sh/packages/platform => ../../packages/platform
