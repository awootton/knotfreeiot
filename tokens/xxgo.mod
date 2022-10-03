module github.com/awootton/knotfreeiot/tokens

replace github.com/awootton/knotfreeiot/badjson => ../badjson

go 1.13

require (
	cloud.google.com/go/firestore v1.5.0 // indirect
	firebase.google.com/go v3.13.0+incompatible
	github.com/awootton/knotfreeiot/badjson v0.0.0-00010101000000-000000000000
	github.com/gbrlsnchs/jwt/v3 v3.0.0-rc.1
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	google.golang.org/api v0.42.0
)
