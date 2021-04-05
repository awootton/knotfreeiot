package tokens

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

func GetFirebaseApp(ctx context.Context) (*firebase.App, error) {

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error UserHomeDir: %v", err)
	}

	opt := option.WithCredentialsFile(home + "/atw/fair-theater-238820-firebase-adminsdk-uyr4z-63b4da8ff3.json")

	//ctx := context.Background()
	config := &firebase.Config{
		DatabaseURL: "https://fair-theater-238820-default-rtdb.firebaseio.com/",
	}

	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing app: %v", err)
	}
	return app, nil
}

//CalcTokenPrice figures out how much we would need to pay to get this token.
func CalcTokenPrice(token *KnotFreeTokenPayload) float64 {
	price := 0.0

	return price
}

type TokenLogStruct struct {
	RemoteAddr string

	When uint32 // unix time

	Token *KnotFreeTokenPayload
}

//LogNewToken to make a record that this token was delivered to customer.
// Let's not include the whole jwt.
func LogNewToken(ctx context.Context, token *KnotFreeTokenPayload, remoteAddr string) error {

	//ctx := context.Background()

	app, err := GetFirebaseApp(ctx)
	if err != nil {
		log.Fatalf("app.Firestore: %v", err)
	}

	client, err := app.Database(ctx)
	if err != nil {
		log.Fatalf("app.Firestore: %v", err)
	}

	tokenLogStruct := &TokenLogStruct{}
	tokenLogStruct.RemoteAddr = remoteAddr
	tokenLogStruct.Token = token
	tokenLogStruct.When = uint32(time.Now().Unix())

	jsonbytes, err := json.Marshal(tokenLogStruct)
	_ = jsonbytes

	currentTime := time.Now()
	str := currentTime.Format("2006-01-02")

	dbpath := "tokens/requests/" + str + "/"
	dbref := client.NewRef(dbpath)

	_, seterr := dbref.Push(ctx, tokenLogStruct)
	if seterr != nil {
		log.Fatalf("app.Firestore: set %v", seterr)
	}

	return nil
}
