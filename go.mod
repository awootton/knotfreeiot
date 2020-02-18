module github.com/awootton/knotfreeiot

go 1.12

replace github.com/awootton/knotfreeiot/packets => ./packets

replace github.com/awootton/knotfreeiot/badjson => ./badjson

replace github.com/awootton/knotfreeiot/iot => ./iot

require (
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.2
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/sys v0.0.0-20200122134326-e047566fdf82 // indirect
	golang.org/x/tools v0.0.0-20200114052453-d31a08c2edf2 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.7 // indirect
)
