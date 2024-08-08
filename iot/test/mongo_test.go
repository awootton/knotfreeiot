package iot_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/tokens"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func makeSampleWatchedItem() ([]*iot.WatchedTopic, []*iot.SavedToken) {

	results := make([]*iot.WatchedTopic, 10)
	savedTokens := make(map[string]*iot.SavedToken, 0)

	ownerPassPhrase := "myFamousOldeSaying" //
	spublic, sprivate := tokens.GetBoxKeyPairFromPassphrase(ownerPassPhrase)
	fmt.Println(ownerPassPhrase, "makes sender public key ", base64.RawURLEncoding.EncodeToString(spublic[:]))
	fmt.Println(ownerPassPhrase, "makes sender private key ", base64.RawURLEncoding.EncodeToString(sprivate[:]))

	for i := 0; i < 10; i++ {

		JWTID := "i6gezcteajn8o6g9lj4xlzm" + fmt.Sprint(i/2) // ie mi6gezcteajn8o6g9lj4xlzm0 to i6gezcteajn8o6g9lj4xlzm4

		payload := &tokens.KnotFreeTokenPayload{}

		payload.Issuer = tokens.GetPrivateKeyPrefix(0) //"_9sh"
		payload.JWTID = JWTID
		payload.ExpirationTime = uint32(time.Now().Unix()) + 60*60*24*365 // 1 year
		// ?? payload.Pubk = clientPublicKey
		priceThing := tokens.GetTokenStatsAndPrice(tokens.TinyX4)
		payload.KnotFreeContactStats = priceThing.Stats

		savedToken := &iot.SavedToken{}
		savedToken.KnotFreeTokenPayload = *payload
		savedToken.IpAddress = "127.0.0.1-" + fmt.Sprint((i+1)%4)

		savedTokens[JWTID] = savedToken

		topic := &iot.WatchedTopic{}

		topic.Name.HashBytes([]byte("contact_address" + fmt.Sprint(i)))
		// topic.Permanent = true
		topic.Expires = uint32(time.Now().Add(time.Hour * 24 * 365 * 2).Unix())
		topic.Jwtid = JWTID

		topic.SetOption("noack", "y")
		topic.SetOption("pub2self", "0")
		topic.SetOption("A", "123.123.123.123")
		topic.SetOption("AAAA", "2001:0000:130F:0000:0000:09C0:876A:130B")
		topic.SetOption("WEB", "get-unix-time.knotfree.net")
		topic.Owners = []string{base64.RawURLEncoding.EncodeToString(spublic[:])}
		topic.Users = []string{topic.Owners[0]}

		ba := &iot.BillingAccumulator{} // for topics that are "bill" only
		ba.Name = topic.Name.String()[0:4]
		//	ba.
		iot.BucketCopy(&savedToken.KnotFreeContactStats, &ba.Max)
		topic.Bill = ba

		results[i] = topic
	}
	// get values from the map
	vals := make([]*iot.SavedToken, 0)
	for _, v := range savedTokens {
		vals = append(vals, v)
	}
	return results, vals
}

func TestMongo(t *testing.T) {

	topics, savedTokens := makeSampleWatchedItem()

	ctx := context.TODO()

	iot.InitMongEnv()
	iot.InitIotTables()

	client, err := mongo.Connect(ctx, iot.MongoClientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	subscriptions := client.Database("iot").Collection("subscriptions")

	for i := 0; i < len(topics); i++ {
		result, err := subscriptions.InsertOne(context.TODO(), topics[i])
		_ = err
		// check(err) // dup key err expected
		if result != nil {
			println("Inserted a single document: ", result.InsertedID)
		}
	}

	saved_tokens := client.Database("iot").Collection("saved-tokens")

	for i := 0; i < len(savedTokens); i++ {
		result, err := saved_tokens.InsertOne(context.TODO(), savedTokens[i])
		_ = err
		// check(err) // dup key err expectd
		if result != nil {
			println("Inserted a single document: ", result.InsertedID)
		}
	}

	// get the toks for an ip
	filter := bson.D{{Key: "ip", Value: "127.0.0.1-2"}}
	cursor, err := saved_tokens.Find(context.TODO(), filter)
	if err != nil {
		check(err)
	}
	defer cursor.Close(context.TODO())
	gotjwt := ""
	for cursor.Next(context.TODO()) {
		var result iot.SavedToken
		err := cursor.Decode(&result)
		check(err)
		fmt.Println("found saved token ", result.KnotFreeTokenPayload.JWTID, result.IpAddress)

		gotjwt = result.KnotFreeTokenPayload.JWTID
	}
	// get the subs for a jwtid
	filter = bson.D{{Key: "jwtid", Value: gotjwt}}
	cursor, err = subscriptions.Find(context.TODO(), filter)
	if err != nil {
		check(err)
	}
	defer cursor.Close(context.TODO())
	for cursor.Next(context.TODO()) {
		var result iot.WatchedTopic
		err := cursor.Decode(&result)
		check(err)
		fmt.Println("found watched topic ", result.Name, result.Jwtid)
	}

}

func TestMongoTopicBson(t *testing.T) {
	topics, savedTokens := makeSampleWatchedItem()

	_ = savedTokens // todo: bson these too

	ba := topics[0].Bill
	bytes, err := bson.Marshal(ba)
	check(err)
	ba2 := iot.BillingAccumulator{}
	err = bson.Unmarshal(bytes, &ba2)
	check(err)

	it := topics[0].OptionalKeyValues.Iterator()
	for it.Next() {
		s, ok := it.Key().(string)
		_ = ok
		fmt.Println("topic opt kv ", s, reflect.TypeOf(it.Value()), it.Value())
	}
	bytes, err = json.Marshal(topics[0])
	check(err)
	// fmt.Println("topic bytes json", string(bytes))
	t2 := &iot.WatchedTopic{}
	err = json.Unmarshal(bytes, t2)
	check(err)
	// fmt.Println("topic 2 json restored ", t2)

	assert.Equal(t, topics[0].Name[0], t2.Name[0])
	assert.Equal(t, topics[0].Name[1], t2.Name[1])
	assert.Equal(t, topics[0].Name[2], t2.Name[2])

	bytes, err = bson.Marshal(topics[0])
	check(err)
	// fmt.Println("topic bytes ", string(bytes))
	// fmt.Println("topic bytes ", showBson(bytes))
	asJson := showBson(bytes)
	_ = asJson
	fmt.Println("BSON length of encoded topic ", len(bytes))

	newtopic := &iot.WatchedTopic{}
	err = bson.Unmarshal(bytes, newtopic)
	check(err)
	assert.Equal(t, topics[0].Name[0], newtopic.Name[0])
	assert.Equal(t, topics[0].Name[1], newtopic.Name[1])
	assert.Equal(t, topics[0].Name[2], newtopic.Name[2])

	// fmt.Println("newtopic ", newtopic)
	it = newtopic.OptionalKeyValues.Iterator()
	for it.Next() {
		s, ok := it.Key().(string)
		_ = ok
		fmt.Println("newtopic opt kv ", s, reflect.TypeOf(it.Value()), it.Value())
	}
}

type Restaurant struct {
	Name         string
	RestaurantId string        `bson:"restaurant_id,omitempty"`
	Cuisine      string        `bson:"cuisine,omitempty"`
	Address      interface{}   `bson:"address,omitempty"`
	Borough      string        `bson:"borough,omitempty"`
	Grades       []interface{} `bson:"grades,omitempty"`
}

func TestMongoRestarants(t *testing.T) {

	iot.InitMongEnv()

	ctx := context.TODO()
	client, err := mongo.Connect(ctx, iot.MongoClientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	restaurants := client.Database("sample_restaurants").Collection("restaurants")

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	name, err := restaurants.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		check(err)
	}
	fmt.Println("Name of restaurants Index Created: " + name)

	newRestaurant := Restaurant{Name: "828299", Cuisine: "Alan" + tokens.GetRandomB36String()}
	resultInsert, err := restaurants.InsertOne(context.TODO(), newRestaurant)
	_ = resultInsert
	if err != nil {
		check(err) // makes dup key error which is expected
		// [E11000 duplicate key error collection: sample_restaurants.restaurants index: name_1 dup key: { name: "828299" }]
	}

	// try an update
	newCuisine := "Alan" + tokens.GetRandomB36String()
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "cuisine", Value: newCuisine}}}}
	// Updates the first document that has the specified "_id" value
	filter := bson.D{{Key: "name", Value: "828299"}}
	updateResult, err := restaurants.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		check(err)
	}

	fmt.Println("updated ", updateResult)

	filter = bson.D{{Key: "name", Value: "828299"}}
	var result Restaurant
	singleResult := restaurants.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("no documents found")
			return
		} else {
			check(err)
		}
	}
	fmt.Println("singleResult ", singleResult)
	assert.Equal(t, newCuisine, result.Cuisine)

}
