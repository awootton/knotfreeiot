module github.com/awootton/knotfreeiot

go 1.12

replace github.com/awootton/knotfreeiot/packets => ./packets

replace github.com/awootton/knotfreeiot/badjson => ./badjson

replace github.com/awootton/knotfreeiot/iot => ./iot

replace github.com/awootton/knotfreeiot/tokens => ./tokens

replace github.com/thei4t/libmqtt => ../libmqtt

require (
	github.com/awootton/knotfreeiot/iot v0.0.0-20200321134716-0d0a9b663fb5
	github.com/awootton/knotfreeiot/kubectl v0.0.0-20200313101551-e4bec42208fd // indirect
	github.com/awootton/knotfreeiot/tokens v0.0.0-00010101000000-000000000000
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/go-openapi/spec v0.19.7 // indirect
	github.com/go-openapi/swag v0.19.8 // indirect

	github.com/google/go-jsonnet v0.15.0 // indirect
	github.com/gorilla/websocket v1.4.1
	github.com/kr/pretty v0.2.0 // indirect
	github.com/ksonnet/ksonnet-lib v0.1.12 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.2
	github.com/prometheus/client_golang v1.5.1
	github.com/thei4t/libmqtt v0.9.5 // indirect
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/tools v0.0.0-20200114052453-d31a08c2edf2 // indirect
	gopkg.in/yaml.v2 v2.2.7 // indirect
)
