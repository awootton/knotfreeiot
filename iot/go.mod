module github.com/awootton/knotfreeiot/iot

go 1.13

replace github.com/awootton/knotfreeiot/packets => ../packets

replace github.com/awootton/knotfreeiot/badjson => ../badjson

replace github.com/awootton/knotfreeiot/iot => ../iotxx

replace github.com/awootton/knotfreeiot/tokens => ../tokens

replace github.com/thei4t/libmqtt => ../../libmqtt

require (
	github.com/awootton/knotfreeiot/badjson v0.0.0-00010101000000-000000000000
	github.com/awootton/knotfreeiot/packets v0.0.0-00010101000000-000000000000
	github.com/awootton/knotfreeiot/tokens v0.0.0-00010101000000-000000000000
	github.com/dchest/siphash v1.2.1 // indirect
	github.com/dgryski/go-maglev v0.0.0-20170623041913-a123f15678dd
	github.com/emirpasic/gods v1.12.0
	github.com/gorilla/websocket v1.4.1

	github.com/minio/highwayhash v1.0.0 // indirect
	github.com/prometheus/client_golang v1.5.1
	github.com/prometheus/client_model v0.2.0
	github.com/thei4t/libmqtt v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.0.0-20190927123631-a832865fa7ad
)
