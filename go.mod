module github.com/awootton/knotfreeiot

go 1.12

replace github.com/awootton/knotfreeiot/packets => ./packets

replace github.com/awootton/knotfreeiot/badjson => ./badjson

replace github.com/awootton/knotfreeiot/iot => ./iot

replace github.com/awootton/knotfreeiot/tokens => ./tokens

require (
	github.com/awootton/knotfreeiot/iot v0.0.0-00010101000000-000000000000
	github.com/awootton/knotfreeiot/tokens v0.0.0-00010101000000-000000000000
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/go-openapi/spec v0.19.7 // indirect
	github.com/go-openapi/swag v0.19.8 // indirect
	github.com/google/go-jsonnet v0.15.0 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/ksonnet/ksonnet-lib v0.1.12 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.2
	github.com/prometheus/client_golang v1.4.1
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/tools v0.0.0-20200114052453-d31a08c2edf2 // indirect
	gopkg.in/yaml.v2 v2.2.7 // indirect
)
