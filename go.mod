module github.com/awootton/knotfreeiot

go 1.14

// replace github.com/awootton/knotfreeiot/packets => ./packetsrequire

//replace github.com/awootton/knotfreeiot/badjson => ./badjson

//replace github.com/awootton/knotfreeiot/iot => ./iot

// replace github.com/awootton/knotfreeiot/tokens => ./tokens

// replace github.com/awootton/knotfreeiot/mainhelpers => ./mainhelpers

// replace github.com/awootton/libmqtt => ../libmqtt

require (
	cloud.google.com/go/firestore v1.6.1 // indirect
	firebase.google.com/go v3.13.0+incompatible
	github.com/awootton/libmqtt v0.2.0
	// github.com/awootton/libmqtt v0.0.0-00010101000000-000000000000
	//	github.com/awootton/knotfreeiot/badjson v0.0.0-20221002062330-114974b38c0d // indirect
	//	github.com/awootton/knotfreeiot/iot v0.0.0-20221002062330-114974b38c0d
	//	github.com/awootton/knotfreeiot/packets v0.0.0-20221002062330-114974b38c0d
	////	github.com/awootton/knotfreeiot/tokens v0.0.0-20221002062330-114974b38c0d
	github.com/aws/aws-sdk-go v1.44.109
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/dgryski/go-maglev v0.0.0-20200611225407-8961b9b1b8e6
	github.com/emirpasic/gods v1.18.1
	github.com/gbrlsnchs/jwt/v3 v3.0.1
	github.com/gorilla/websocket v1.5.0
	//github.com/hashicorp/mdns v1.0.5
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.5.0
	github.com/prometheus/client_golang v1.13.0
	github.com/prometheus/client_model v0.2.0
	golang.org/x/crypto v0.0.0-20220926161630-eccd6366d1be
	google.golang.org/api v0.59.0

)
