// copyright alan tracey wootton 2021

package mainhelpers_test

// this token comes from topkens.TestMakeToken1connection
var sampleToken1 = `eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDkzNzY0MDAsImlzcyI6Il85c2giLCJqdGkiOiIxMjM0NTYiLCJpbiI6MjAsIm91dCI6MjAsInN1IjoxLCJjbyI6MSwidXJsIjoia25vdGZyZWUubmV0In0.i5-h6Yup6vYVD6HZhzIz_jP0y1FYkqfiM4D56eJi_-L8DWyDB9_6gSozpdF3eNgRHKBexiLVyhAAqLHUHLMZBw`

// TestMakeLargeToken is 4 connections for a year
// func XXXnot_TestMakeLargeToken(t *testing.T) {

// 	tokens.LoadPublicKeys()

// 	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

// 	fmt.Println("in TestMakeLargeToken")

// 	tokenRequest := &tokens.TokenRequest{}
// 	payload := tokens.KnotFreeTokenPayload{}
// 	tokenRequest.Payload = &payload

// 	payload.Connections = 4 // TODO: move into standard x-small token

// 	// a year - standard x-small
// 	payload.ExpirationTime = uint32(time.Now().Unix() + 60*60*24*365)

// 	payload.Input = 32 * 4  // TODO: move into standard x-small token
// 	payload.Output = 32 * 4 // TODO: move into standard x-small token

// 	payload.Issuer = "_9sh"
// 	payload.JWTID = tokens.GetRandomB36String()
// 	nonce := payload.JWTID
// 	_ = nonce

// 	payload.Subscriptions = 20 // TODO: move into standard x-small token

// 	//  Host:"building_bob_bottomline_boldness.knotfree2.com:8085"
// 	targetSite := "knotfree.net" // gotohere.com"
// 	if os.Getenv("KNOT_KUNG_FOO") == "Xatw" {
// 		targetSite = "gotolocal.com"
// 	}
// 	payload.URL = targetSite + "/mqtt"

// 	exp := payload.ExpirationTime
// 	if exp > uint32(time.Now().Unix()+60*60*24*365) {
// 		// more than a year in the future not allowed now.
// 		exp = uint32(time.Now().Unix() + 60*60*24*365)
// 		fmt.Println("had long token ", string(payload.JWTID)) // TODO: store in db
// 	}

// 	cost := tokens.CalcTokenPrice(&payload, uint32(time.Now().Unix()))
// 	jsonstr, _ := json.Marshal(payload)
// 	fmt.Println("token cost is "+fmt.Sprintf("%f", cost), string(jsonstr))

// 	large32x := mainhelpers.ScaleTokenPayload(&payload, 8*32)
// 	cost = tokens.CalcTokenPrice(large32x, uint32(time.Now().Unix()))
// 	jsonstr, _ = json.Marshal(large32x)
// 	fmt.Println("token cost is "+fmt.Sprintf("%f", cost), string(jsonstr))

// 	fmt.Println("token is "+fmt.Sprintf("%f", cost), string(jsonstr))

// 	fmt.Println("token is "+fmt.Sprintf("%f", cost), string(jsonstr))

// }

// func not_TestMakeHugeToken(t *testing.T) {
// 	_ = t
// 	tokens.LoadPublicKeys()

// 	tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

// 	tok := tokens.Get32xTokenLocal() //mainhelpers.MakeMedium32cToken()
// 	fmt.Println(tok)

// }
