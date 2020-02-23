module github.com/awootton/knotfreeiot

go 1.12

replace github.com/awootton/knotfreeiot/packets => ./packets

replace github.com/awootton/knotfreeiot/badjson => ./badjson

replace github.com/awootton/knotfreeiot/iot => ./iot

require (
	github.com/kr/pretty v0.2.0 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.2
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/tools v0.0.0-20200114052453-d31a08c2edf2 // indirect
	gopkg.in/yaml.v2 v2.2.7 // indirect
)
