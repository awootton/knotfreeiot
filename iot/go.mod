module github.com/awootton/knotfreeiot/iot

go 1.13

replace github.com/awootton/knotfreeiot/packets => ../packets

replace github.com/awootton/knotfreeiot/badjson => ../badjson

replace github.com/awootton/knotfreeiot/iot => ../iot

replace github.com/awootton/knotfreeiot/tokens => ../tokens

require (
	github.com/awootton/knotfreeiot/badjson v0.0.0-00010101000000-000000000000
	github.com/awootton/knotfreeiot/packets v0.0.0-00010101000000-000000000000
	github.com/awootton/knotfreeiot/tokens v0.0.0-00010101000000-000000000000
	github.com/dchest/siphash v1.2.1 // indirect
	github.com/dgryski/go-maglev v0.0.0-20170623041913-a123f15678dd
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/emirpasic/gods v1.12.0
	github.com/gbrlsnchs/jwt/v3 v3.0.0-rc.1
	github.com/minio/highwayhash v1.0.0
	github.com/prometheus/client_golang v1.4.1
	github.com/prometheus/client_model v0.2.0
)
