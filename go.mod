module github.com/awootton/knotfreeiot

go 1.14

replace github.com/awootton/knotfreeiot/packets => ./packets

replace github.com/awootton/knotfreeiot/badjson => ./badjson

replace github.com/awootton/knotfreeiot/iot => ./iot

replace github.com/awootton/knotfreeiot/tokens => ./tokens

//replace github.com/thei4t/libmqtt => ../libmqtt

require (
	github.com/awootton/knotfreeiot/iot v0.0.0-20210322105824-7c2b62c09ca0
	github.com/awootton/knotfreeiot/packets v0.0.0-00010101000000-000000000000
	github.com/awootton/knotfreeiot/tokens v0.0.0-20210322105824-7c2b62c09ca0
	github.com/gorilla/websocket v1.4.2
	github.com/klauspost/compress v1.11.12 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.3.0
	github.com/prometheus/client_golang v1.10.0
	golang.org/x/crypto v0.0.0-20210317152858-513c2a44f670
	golang.org/x/net v0.0.0-20210316092652-d523dce5a7f4 // indirect
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.4.0 // indirect

)
