module github.com/awootton/knotfreeiot/clients

go 1.13

replace github.com/awootton/knotfreeiot/packets => ../packets

replace github.com/awootton/knotfreeiot/badjson => ../badjson

replace github.com/awootton/knotfreeiot/iot => ../iot

replace github.com/awootton/knotfreeiot/tokens => ../tokens

require (
	github.com/awootton/knotfreeiot/packets v0.0.0-00010101000000-000000000000
	github.com/awootton/knotfreeiot/tokens v0.0.0-00010101000000-000000000000
)
