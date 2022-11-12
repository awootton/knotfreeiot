package tokens_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/tokens"
)

func not_TestLogTok(t *testing.T) { // fixme

	ctx := context.Background()

	startTime := time.Now().Unix()

	token := GetSampleTokenPayload(uint32(startTime))

	remoteAddr := "10.10.10.10"

	tokens.LogNewToken(ctx, token, remoteAddr)

	_ = ctx
	_ = token

}

// FIXME: dump firebase
func not_TestLoggingTokenCreate(t *testing.T) {

	ctx := context.Background()

	app, err := tokens.GetFirebaseApp(ctx)

	fmt.Println("jello ")

	client, err := app.Database(ctx)
	if err != nil {
		log.Fatalf("app.Firestore: %v", err)
	}

	fmt.Println("cc ", client)

	data := map[string]string{
		"msg": "a message",
		"sum": "Happy Day",
	}

	d2, err := json.Marshal(data)
	want := string(d2)
	fmt.Println("json data ", want)

	dbref := client.NewRef(("k1/k2/k3"))
	seterr := dbref.Set(ctx, data)
	if seterr != nil {
		log.Fatalf("app.Firestore: set %v", seterr)
	}

	var got map[string]string
	err = dbref.Get(ctx, &got)
	if err != nil {
		log.Fatalf("app.Firestore: set %v", err)
	}

	d3, err := json.Marshal(got)
	fmt.Println("got json data ", string(d3))
	if want != string(d3) {
		t.Errorf("got %v, want %v", string(d3), want)
	}

	err = dbref.Delete(ctx)
	if err != nil {
		log.Fatalf("app.Firestore: delete %v", err)
	}

	err = dbref.Get(ctx, &got)
	if err != nil {
		log.Fatalf("app.Firestore: set %v", err)
	}
	d3, err = json.Marshal(got)
	fmt.Println("got removed json data ", string(d3))
	if string(d3) != "null" {
		t.Errorf("got %v, want %v", string(d3), string(d2))
	}

	aPretendNewToken := `["My token expires: 2020-12-31",{"iss":"_9sh","in":32,"out":32,"su":4,"co":2,"url":"knotfree.net"},"eyJhbGciOiJFZDI1NTE5IiwidHlwIjoiSldUIn0.eyJleHAiOjE2MDkzNzI4MDAsImlzcyI6Ii85c2giLCJqdGkiOiJqQ0ZqYVNQRGUrUVVwb3NCc0VGK2Uxa2wiLCJpbiI6MzIsIm91dCI6MzIsInN1Ijo0LCJjbyI6MiwidXJsIjoia25vdGZyZWUubmV0In0.LLTrTcFRpngXlOpgte_F6HaLxkXDf5fz17eRMvR5Ymo5lHDb3zoedRklD_dyr1qMIqZ52cOffVj6EqYu8ah8Dg"]`
	_ = aPretendNewToken

}
