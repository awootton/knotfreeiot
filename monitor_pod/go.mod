module knotfree.net/monitor_pod

go 1.19

// when develping locally:
//
// replace github.com/awootton/knotfreeiot => ../../knotfreeiot

replace github.com/awootton/knotfreeiot/monitor_pod => ./

require (
	github.com/awootton/knotfreeiot v0.1.6
	golang.org/x/crypto v0.6.0
)

require (
	cloud.google.com/go v0.110.0 // indirect
	cloud.google.com/go/compute v1.18.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/firestore v1.9.0 // indirect
	cloud.google.com/go/iam v0.12.0 // indirect
	cloud.google.com/go/longrunning v0.4.1 // indirect
	cloud.google.com/go/storage v1.29.0 // indirect
	firebase.google.com/go v3.13.0+incompatible // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/gbrlsnchs/jwt/v3 v3.0.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.8 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/magefile/mage v1.9.0 // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.59.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230227214838-9b19f0bdc514 // indirect
	google.golang.org/grpc v1.53.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
