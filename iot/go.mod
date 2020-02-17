module github.com/awootton/knotfreeiot/iot

go 1.13

replace github.com/awootton/knotfreeiot/packets => ../packets

replace github.com/awootton/knotfreeiot/badjson => ../badjson

replace github.com/awootton/knotfreeiot/iot => ../iot

require (
	github.com/awootton/knotfreeiot/badjson v0.0.0-00010101000000-000000000000
	github.com/awootton/knotfreeiot/packets v0.0.0-00010101000000-000000000000
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/emirpasic/gods v1.12.0
	github.com/minio/highwayhash v1.0.0
	github.com/prometheus/client_golang v1.4.1
	github.com/prometheus/client_model v0.2.0
)
