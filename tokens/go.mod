module github.com/awootton/knotfreeiot/tokens

replace github.com/awootton/knotfreeiot/badjson => ../badjson

go 1.13

require (
	github.com/awootton/knotfreeiot/badjson v0.0.0-00010101000000-000000000000
	github.com/gbrlsnchs/jwt/v3 v3.0.0-rc.1
	github.com/the729/go-libra v0.0.0-20200305182524-0104ebe4e12b
	golang.org/x/crypto v0.0.0-20190927123631-a832865fa7ad
)
