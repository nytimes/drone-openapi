module github.com/NYTimes/drone-openapi

go 1.12

replace golang.org/x/oauth2 => github.com/wlhee/oauth2 v0.0.0-20190308230854-b33c8d1d8308

require (
	github.com/drone/drone-plugin-go v0.0.0-20160112175251-d6109f644c59
	github.com/ghodss/yaml v1.0.0
	github.com/pkg/errors v0.8.1
	golang.org/x/oauth2 v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v2 v2.2.2 // indirect
)
