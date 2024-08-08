package tokens

import (
	"log"
	"math"
	"os"
)

/*
	civo:
	Extra Small	  1 GB	1 core	30GB NVMe	1 TB	$5 per month	/ 1 = 1 GB 1  core 30GB   1   TB	$5 per month
	Small	      2 GB	1 core	40GB NVMe	2 TB	$10 per month / 2 = 1 GB .5 core 20GB   1   TB	$5 per month
	Medium	      4 GB	2 cores	50GB NVMe	3 TB	$20 per month / 4 = 1 GB .5 core 12.5GB .75 TB	$5 per month
	Large	      8 GB	4 cores	60GB NVMe	4 TB	$40 per month / 8 = 1 GB 1  core 30GB   1   TB	$5 per month

	/ 10,000 = 100k ram, 3 mb disk, 100mb i/o
	100mb io / 2592000 sec/month = 38 bytes / sec we'll need 4 of those
	$5 / 10000 = $0.0005 and $0.0005 * 4 = $0.002 and $0.002 * 12 = 2.4 cents or about a quarter per decade.
	ima double all this for a margin.
	if a subscription uses 4k bytes across the cluster (totally unknown) then
			1 GB / 4000 = 250k subs per instance and 250k / 10,000 conn/instance = 25

*/

type KnotFreeContactPrices struct {
	Stats KnotFreeContactStats
	Price float64 `json:"pr"`
}

// as per Civo 11/2022.
var OneConnectionToken = KnotFreeContactPrices{
	Stats: KnotFreeContactStats{
		Connections:   1,
		Subscriptions: 2, // was 25
		Input:         38,
		Output:        38,
	},
	Price: 0.0005 * 2, // per month
}

func ScaleTokenPrice(in KnotFreeContactPrices, factor float64) KnotFreeContactPrices {
	res := KnotFreeContactPrices{
		Stats: KnotFreeContactStats{
			Connections:   math.Floor(in.Stats.Connections * factor),
			Subscriptions: math.Floor(in.Stats.Subscriptions * factor),
			Input:         math.Floor(in.Stats.Input * factor),
			Output:        math.Floor(in.Stats.Output * factor),
		},
		Price: in.Price * factor, // per month
	}
	return res
}

func GetTokenStatsAndPrice(ttype TokenType) KnotFreeContactPrices {
	power := 1 << ttype
	return ScaleTokenPrice(OneConnectionToken, float64(power))
}

func GetTokenTenKStatsAndPrice() KnotFreeContactPrices {
	return ScaleTokenPrice(OneConnectionToken, float64(10*2000))
}

type TokenType int

// these are powers of two
const (
	Tiny TokenType = iota
	TinyX2
	TinyX4 // this is the free one , 4 connections, 8 names
	TinyX8
	Small     // 16 connections
	Medium    // 32 connections aka medium32
	MediumX2  // 64 connections
	Large     // 128 connections
	LargeX2   // 256 connections
	LargeX4   // 512 connections
	LargeX8   // 1024 connections
	LargeX16  // 2048 connections
	LargeX32  // 4096 connections
	Giant     // 8192
	GiantX2   // 16384 now it's more than one vn
	GiantX4   // 32768
	GiantX8   // 64k
	GiantX16  // 128k
	GiantX32  // 256k
	GiantX64  // 1m
	GiantX128 // 2m
	GiantX256 // 4m
)

// SampleSmallToken is a small token signed by "_9sh" (below)
// p.Input = 20
// p.Output = 20
// p.Subscriptions = 2
// p.Connections = 2
// and, it's expired.
var XxxxSampleSmallToken = `[My_token_expires:_2021-12-31,{exp:1641023999,iss:_ 9 s h,jti:amXYKIuS4uykvPem9Fml371o,in:32,out:32,su:4,co:2,url:knotfree.net},e y J hbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2NDEwMjM5OTksImlzcyI6Il85c2giLCJqdGkiOiJhbVhZS0l1UzR1eWt2UGVtOUZtbDM3MW8iLCJpbiI6MzIsIm91dCI6MzIsInN1Ijo0LCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.7ElPyX1Vju8Q5uDOEfgAblwvE2gxT78Jl68JPlqLRcFeMJ7if39Ppl2_Jr_JTky371bIXAn6S-pghtWSqTBwAQ]`

// no point loading them all the time.
// ed25519
// one per line.
// _9sh is being used to sign tokens
// 8ZNP is unused
// yRst is used as seed to cluster box keypair
// the others are unused so far and the private part unloaded.
//
//	TODO: move these to a file.
var publicKeys string = ""

func GetPublicKeys() string {
	if publicKeys == "" {
		dirname, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		fname := dirname + "/atw/publicKeys.txt"

		tmp, err := os.ReadFile(fname)
		if err != nil {
			return ""
		}
		publicKeys = string(tmp)
	}
	return publicKeys
}

// Example:
// _9sh+kvk3Nd/oN7nq56ydRaFON0YxQ+qCoBL0H91fV4
// 8ZNPzzn2EEnlFCAH6Z//KNHoIyhnIWDGRcy0Ub6F/mc
// yRst5ig1Zf1iYVvI0q0LltjU8gmT+9ZZBKWijosq2Vg
// JvaLqA2oYU9mZHcYYtCWJ7occcW5BiNpbdR2gSVHCFY
// JIbDPOv+0H2zT6bXlO8oMGWWh9NJf+Mz4d6UXETiPZo
// aNhfKWPWWrCkP8R/BCWUmgwv2gZg2wz9e/FmXdKqNG0
// RHLSR6DdlpwCeYOE7DF/QaUGE3AwMZU4F0/uuM1HYCY
// B30LVkD9TY96cD6S54xrnSoa6j/W14RJ0NH55YPiaMw
// cj2gEtBk0qXrxhjKbwUYlD1naOMMHhX0L3s7qGHMvmw
// wfrQr0IqTuvXwTlNdg4yO0H5d2nmeEV93kwkplVV0Gc
// sJNAsh0yH3sY8Qu56zo9J64kNSju+o662FT+OEaW3sc
// p/ia+nTuOaEbKkp2S8uTyccacmdEKPaxj7AOzIyYPbU
// VdGjvGBES2cBXsk8XvJVj55woUxTDegvR+NB1jfocbU
// IN9yT9wMGTOoLQDgdHK7ue8IOzLHkrw5/0DM06jYYlI
// dmTDblSn2A/gnF1dB6RuFDjMk29G8DziKBH0zOUjqUg
// fr8KVrMqF669rKazI6Vs3OO3dYyGjW+gMgXx/XLiEX8
// V5iE4tUGSeamu/r24XOWsrvzvdt0A8R+O2XArT1lvmQ
// ql/nLaNeSDtl6i9fKofC2WT2H9VqHLj0VCLgWS8oEcM
// cvVrcTKky67XukswYgYdttODLTuh5iwlpCBAKaysFGw
// leePkNZx3ns8LOS5jxjxH0ybjn5E6r5gaEO4fwRXO8g
// 3ZKqO3ppTjjfaGFcgwYAcJ9OvXVF0hyeIu8KQgMVOQw
// SxC+EHhmiVYCAtpvp3HWknAdkwzVhKaRnmj8Gnsic5Y
// ebBVe8AMUIvz6raYozdfAWeRmcjF2a8lvY1dTnjDOOc
// O+x0aSZ42c/AUH4hnb0GNRx2I70R1ncuBAeOSrLaG0k
// rKxTWJhMAvaDtLEmxxB6kYSvpJR7ou80dMCEOOxzrLA
// 6sCrFd/c4Leh6F9WxpSCsuKeANpNN57OJxPcDK5KC68
// 3aTB768a1HYrBb2KA1rXv/A6AgBqZW1F7n3JTK8vpl8
// bkBYvnQqxzCUBNpz8aPGBd8rM4gzdGO+JnNueicecr4
// zGJSXO5/KdqqYMwBtHguHpT14jQdE+OOA6PQZjVphuQ
// gW6CF4WH6dyg3Yx1LqLGpiH707NnQUP8nM9PY15SBKA
// rkDnOkSj6XPNNyH/Vlkaaewwi3q8/ePcvXUOiBIu52o
// ByKGuFQteDJLFizQfW1oPGbyh9rL1Yj7SNE0f/q5Xys
// pwAkPWMNZwjuiTrPg4YR58KJFIjqn204BjVzdaCChtI
// mp+9zBU/kSIAMHiZiZBxXe+DtIuddwsWWao/AtU7fmQ
// tA8lPUJtgDP80ga+bj7XFwn2p6BOSZghk21v5X2jq5s
// JAKlfGDiioDYYZEsq6NtEhZdIkCl83oHQRTe+SiA/bI
// GJIebuse2Q/9T6wRFb7rlPd9uOcom0Wx28C/OCB4wHc
// klitu+aunEGRjMaj2nYbBBS9hoohbDmIToQg+9Oc1pw
// kNQSM4gz+1eXDoHnCOIK3oWvhczEgHuP6fD0ecqjGNo
// f//Mser4Py/e/hIvxyDL9q2vjEdz6+ThZYrmXoVBxKc
// vgpHdHc35hIj/DW+vwIiNbyUYwWsihApFo7Vfjd3z94
// J8u8BnH6o6QOMJ16UwgIhL5Dn/ARB31xqnzvYMoHH6o
// N7ok5KZ3YbgcxHkh8ZdV38yE+2Azq7BzyDDrC+JnMAY
// FFdKDiX45E2RfauLWXVd7xBmFHO9Tu2zJSk6FTWHjbc
// HgfPkJqVvifOEZQsJIdAJGGQVlpRO4JAhtcsp6Fz4lI
// 28Mwm1olWZ0D42IYd0hUlyGeHWN9jf4muiSQWen+WS4
// arS0VuqGXNWssBgGc88n1ZfKA1KEcFYgn+Ox//LH5/w
// 8X80fLAo3Cfct/KqYRutuDLv4uCPZ2i3K7ayO7hYUlc
// TJ6ZGaAfHZIU3T3EQ0L/jvB90L/R9yLjsECNFcFAXPY
// gU51mGgvwB/OkQPY9YB3TSi+eNrBQh4vGLD6fTD4qrE
// n25r4SFrtVsrfMGUw8kWUF4vTCkezgJ8raB4UpSKiTQ
// IElPHV4ShGf0kN9pdKgvJrTT9JspWF2vMTtWBqTqUAY
// sgIKxzYEWre7ZNYT4cfYldcGO3XUmXnIksJJh6+miP4
// WVrL5zNpeO0BFlZr4hyBOdK7tLDyC37JrGbRvvEHhoc
// JNMyq/aR4kQlHp0+x8D+E2caIBypaLUfBBzyXxYqsio
// aoaJoZbc0AzXPCZTcfwVUnr3f5Owrojhh4w/wG9JH1M
// UYsfYekd71ElufJcfJ9PMOyYkPoDgSXvlo3V6LKB5zU
// xvWcpdZGW7GdrmZAIJsbGydcYXx495qSacoTSN1Xdsg
// g+ezyaJgv/ZwBpEr80pLxGweXF1Hn6KIVJCg779+/FY
// nm3TYMVGlIN+tXYoiAvOILjKUsmJ3OdbhGkh9puxguA
// 8cPnPSfE9wy7erZGriwde/R2u46mvDP0IGtfFDXaiJw
// Ditv5v1hDgI5L0rD2dgJN6Iz+hzVqAiB08t7vSFnYxw
// vdreVQjOIrv2o+wW/EJi0g+bQ8S71NHFB45ndKE1Des
// 7hwDiSi9ZOOn4IXVEIbMdTqpRE2ayScY6uogj5aBad0
// `
