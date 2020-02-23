module github.com/awootton/knotfreeiot/tokens

replace github.com/awootton/knotfreeiot/badjson => ../badjson

go 1.13

require (
	github.com/awootton/knotfreeiot/tickets v0.0.0-20200221084641-b45967e53fda
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gbrlsnchs/jwt/v3 v3.0.0-rc.1
)
