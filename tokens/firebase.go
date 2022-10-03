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

// CalcTokenPrice figures out how much we would need to pay to get this token.
// TODO: move out of firebase
func CalcTokenPrice(token *KnotFreeTokenPayload, unixIssueTime uint32) float64 {
	price := float64(0.0)
	// at DigitalOcean 4/2021:
	// $5
	// 0.5 cpu
	// 1 Gb so 500k subs
	// 1 tb io
	// say 10k connections
	// $0.01/GB. when over

	// so, divide by 5000
	// tinyCost := float32(5) / float32(5000) // 100 m$
	// tinyConnectAmt := float32(10*1000) / float32(5000)
	// tinyIO := float32(1000 * 1000 * 1000 * 1000 / 5000) // per month

	// fmt.Println("tinyCost ", tinyCost)               // 0.001 or 1 m$ or 1000 u$
	// fmt.Println("tinyConnectAmt ", tinyConnectAmt)   // 2
	// fmt.Println("tinyIO/sec ", tinyIO/secsInMonth) // 77 bytes/sec

	greaterPrice := float64(-1.0)

	secsInMonth := float64(60 * 60 * 24 * 30)

	connectionPrice := token.Connections * float64(5) / float64(10*1000)
	subscriptionPrice := token.Subscriptions * float64(5) / float64(500*1000)
	subscriptionPrice *= 2                                                             // because it's on two layers
	ioPrice1 := token.Input * secsInMonth * float64(5) / float64(1000*1000*1000*1000)  // per month
	ioPrice2 := token.Output * secsInMonth * float64(5) / float64(1000*1000*1000*1000) // per month

	greaterPrice = connectionPrice
	if subscriptionPrice > greaterPrice {
		greaterPrice = subscriptionPrice
	}
	if ioPrice1 > greaterPrice {
		greaterPrice = ioPrice1
	}
	if ioPrice2 > greaterPrice {
		greaterPrice = ioPrice2
	}

	nowSeconds := unixIssueTime //uint32(time.Now().Unix())
	deltaTime := float64(uint32(token.ExpirationTime - nowSeconds))
	deltaTime = deltaTime / secsInMonth // now in months

	price = greaterPrice * deltaTime

	return price
}

type TokenLogStruct struct {
	RemoteAddr string

	When uint32 // unix time

	Token *KnotFreeTokenPayload
}

// LogNewToken to make a record that this token was delivered to customer.
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

	log.Println(" starting fire push : ")

	_, seterr := dbref.Push(ctx, tokenLogStruct)
	if seterr != nil {
		fmt.Println("about to die from app.Firestore error ", seterr)
		//log.Fatalf("app.Firestore: set %v", seterr)
		log.Println(" ERROR app.Firestore: ", seterr)
	}

	return nil
}
