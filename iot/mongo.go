// Copyright 2019,2020,2021-2024 Alan Tracey Wootton
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package iot

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/tokens"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TODO: cleanup

// tables
/**
subscription aka  watched item
	index on hash unique
	index on jwtid
	pubk

*/
/**
Tokens
	index on jwtid unique
	fields + pubk
	collection of ip addresses
*/

type SavedToken struct {
	// JWTID is the unique identifier for the token
	tokens.KnotFreeTokenPayload

	// IP is the ip address used when the token was created
	IpAddress string `json:"ip" bson:"ip"`
}

/**
ipaddress // for limiting tokens
	index on ip unique
	index on jwtid
*/
/**
billing

	index on jwtid unique
*/
/**
people
	phone,
	email,
	unique index on email
	unique index on phone
	index on jwtid
*/
/**
payments?

*/

var mongoInited = false
var mongoInitedLock sync.Mutex
var MongoClientOptions *options.ClientOptions

var mongoClient *mongo.Client

var gNonExistingSubsCache *expirable.LRU[string, string]

// var mongoTablesInited = false
// var mongoTablesInitedLock sync.Mutex
var subscriptionsDb *mongo.Collection

func InitMongo() {
	if mongoInited {
		return
	}
	// one gets the lock and finishes. The others wait and see that the init is done and return.
	mongoInitedLock.Lock()
	defer mongoInitedLock.Unlock()
	if mongoInited {
		return
	}
	startTime := time.Now()
	defer func() {
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		fmt.Println("InitMongo took ", duration)
	}()
	fmt.Println("InitMongo starting")

	MongoClientOptions = initMongEnv()
	_ = MongoClientOptions
	initIotTables()

	gNonExistingSubsCache = expirable.NewLRU[string, string](32*1024, nil, time.Second*300)

	mongoInited = true
}

// GetMongoClient returns THE mongo client. Is this safe to just do once ever?
// Is it safe to call this multiple times?
// unused  atw delete me clean up the trash
// func XXlocalGetMongoClient() (*mongo.Client, error) {

// 	// InitMongEnv()
// 	// InitIotTables()

// 	clientConnectLock.Lock()
// 	defer clientConnectLock.Unlock()
// 	if clientConnectDone {
// 		return mongoClient, nil
// 	}
// 	var err error
// 	mongoClient, err = mongo.Connect(context.TODO(), MongoClientOptions)
// 	if err != nil {
// 		return nil, err
// 	}
// 	clientConnectDone = true

// 	// 32K keys that go nowhere and expire after 5 minutes.
// 	// This is for caching the not founds in GetSubscription, so we don't have to hit mongo every time for a non existing topic.
// 	//
// 	gNonExistingSubsCache = expirable.NewLRU[string, string](32*1024, nil, time.Second*300)

// 	return mongoClient, nil
// }

func GetSubscriptionList(ownerPubk string) ([]WatchedTopic, error) {

	// client, err := GetMongoClient()
	// if err != nil {
	// 	fmt.Println("mongo.Connect err", err)
	// 	return nil, err
	// }
	// // defer client.Disconnect(ctx)

	InitMongo()

	filter := bson.D{{Key: "own", Value: ownerPubk}}
	cursor, err := subscriptionsDb.Find(context.TODO(), filter)
	if err != nil {
		fmt.Println("mongo find err", err)
		return nil, err
	}

	var names []WatchedTopic
	if err = cursor.All(context.TODO(), &names); err != nil {
		return nil, err
	}

	// fmt.Println("found watched topic ", len(names))

	return names, nil
}

func GetSubscriptionListCount(ownerPubk string) (int, error) {

	InitMongo()

	filter := bson.D{{Key: "own", Value: ownerPubk}}
	cursor, err := subscriptionsDb.Find(context.TODO(), filter)
	if err != nil {
		fmt.Println("mongo find err", err)
		return 0, err
	}

	// FIXME: find a better way to just get the count.
	// maybe fetch the distinct names and count them?

	var names []WatchedTopic
	if err = cursor.All(context.TODO(), &names); err != nil {
		return 0, err
	}

	fmt.Println("found watched topic count ", len(names))

	return len(names), nil
}

func ClearCachedSubscription(hashedTopicStr string) {
	gNonExistingSubsCache.Remove(hashedTopicStr)
}

// the core of the lookup. Get the subscription from the database.
// The most used function.
func GetSubscription(hashedTopicStr string) (*WatchedTopic, bool) {

	InitMongo()

	cached, ok := gNonExistingSubsCache.Get(hashedTopicStr)
	if ok {
		if cached == "0" {
			// this is a cached "not found" which happens a lot because of the way an OctTree works.
			return nil, false
		}
	}
	topic, ok := getSubscriptionInternal(hashedTopicStr)
	if ok {
		// fmt.Println("GetSubscription found topic for ", hashedTopicStr)
		return topic, true
	}
	// not found. Cache the not found.
	gNonExistingSubsCache.Add(hashedTopicStr, "0")
	return nil, false
}

// GetNoneSubscription will see if the cache already knows full well that this topic doesn't
// exist. What do I return? lol.
// GetNoneSubscription returns true if the topic is unknown and false if it MIGHT exist.
func GetNoneSubscription(hashedTopicStr string) bool {

	InitMongo()

	cached, ok := gNonExistingSubsCache.Get(hashedTopicStr)
	if ok {
		if cached == "0" {
			// this is a cached "not found" which happens a lot because of the way an OctTree works.
			return true
		}
	}
	return false
}

func getSubscriptionInternal(hashedTopicStr string) (*WatchedTopic, bool) {

	// fmt.Println("Mongo GetSubscription ", hashedTopicStr)
	startTime := time.Now()
	defer func() {
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		if duration > 1000*time.Millisecond {
			fmt.Println("GetSubscription SLOW took ", duration) // I can't live like this. FML.
		}
	}()

	filter := bson.D{{Key: "name", Value: hashedTopicStr}}
	result := subscriptionsDb.FindOne(context.TODO(), filter)
	if result.Err() != nil {
		// fmt.Println("mongo find name err", result.Err())
		return nil, false
	}
	found := WatchedTopic{}
	err := result.Decode(&found)
	if err != nil {
		fmt.Println("mongo find name Decode err", err)
		return nil, false
	}
	// fmt.Println("found watched topic ", found.Name.ToBase64(), found.Jwtid)
	return &found, true
}

// DeleteSubscription deletes a subscription from the database.
// hashedTopicStr is the base64 encoded topic name.
func DeleteSubscription(hashedTopicStr string) error {

	// client, err := GetMongoClient() //:= mongo.Connect(ctx, MongoClientOptions)
	// if err != nil {
	// 	fmt.Println("mongo.Connect err", err)
	// 	return err
	// }
	// defer client.Disconnect(ctx)

	InitMongo()
	subscriptions := subscriptionsDb

	filter := bson.D{{Key: "name", Value: hashedTopicStr}}
	result, err := subscriptions.DeleteOne(context.TODO(), filter)
	if err != nil {
		// fmt.Println("mongo delete name err", result.Err())
		return err
	}
	_ = result
	return nil
}

func SaveSubscription(watchedTopic *WatchedTopic) error {

	InitMongo()

	if watchedTopic == nil {
		return fmt.Errorf("watchedTopic is nil")
	}
	if watchedTopic.Created == 0 {
		watchedTopic.Created = uint32(time.Now().Unix())
	}

	// ctx := context.TODO()

	// client, err := mongo.Connect(ctx, MongoClientOptions) // is this a new connection each time?
	// if err != nil {
	// 	fmt.Println("mongo.Connect err", err)
	// 	return err
	// }
	// defer client.Disconnect(ctx)

	// subscriptions := client.Database("iot").Collection("subscriptions")

	hashedTopicStr := watchedTopic.Name.ToBase64()
	ClearCachedSubscription(hashedTopicStr) // clear the cache for this topic, since it's being updated. This is to prevent the cache from returning a not found for a topic that now exists.
	filter := bson.D{{Key: "name", Value: hashedTopicStr}}
	result := subscriptionsDb.FindOne(context.TODO(), filter) // I hate this.
	if result.Err() != nil {
		// not found
		// insert
		result, err := subscriptionsDb.InsertOne(context.TODO(), watchedTopic)
		_ = result
		return err

	} else {
		// found
		// replace
		result, err := subscriptionsDb.ReplaceOne(context.TODO(), filter, watchedTopic)
		_ = result
		return err
	}

	// result, err := subscriptions.UpdateOne(context.TODO(), filter, watchedTopic)
	// if err != nil {
	// 	fmt.Println("mongo insert err", err)
	// 	return err
	// }
	// _ = result
	// return nil
}

type ChildBitsCache struct {
	WorldName string `bson:"world"`
	Timestamp int64  `bson:"timestamp"`
	Data      []byte `bson:"data"`
}

func SaveChildBitsNameAndData(world string, data string) bool {
	cache := &ChildBitsCache{
		WorldName: world,
		Timestamp: time.Now().Unix(),
		Data:      []byte(data),
	}
	return SaveChildBitsCache(world, cache)
}

func SaveChildBitsCache(world string, cache *ChildBitsCache) bool {
	InitMongo()

	if world == "" {
		return false
	}

	if cache.Timestamp == 0 {
		cache.Timestamp = time.Now().Unix()
	}

	childBitsCacheColl := mongoClient.Database("iot").Collection("child-bits-cache")
	filter := bson.D{{Key: "world", Value: world}}
	result := childBitsCacheColl.FindOne(context.TODO(), filter)
	if result.Err() != nil {
		// not found, insert
		_, err := childBitsCacheColl.InsertOne(context.TODO(), cache)
		if err != nil {
			fmt.Println("mongo insert child bits cache err", err)
			return false
		}
		return true
	} else {
		// found, replace
		_, err := childBitsCacheColl.ReplaceOne(context.TODO(), filter, cache)
		if err != nil {
			fmt.Println("mongo replace child bits cache err", err)
			return false
		}
	}
	return true
}

func GetChildBitsCache(world string) (*ChildBitsCache, bool) {
	// implementation goes here
	InitMongo()

	if world == "" {
		return nil, false
	}

	childBitsCacheColl := mongoClient.Database("iot").Collection("child-bits-cache")
	filter := bson.D{{Key: "world", Value: world}}
	result := childBitsCacheColl.FindOne(context.TODO(), filter)
	if result.Err() != nil {
		return nil, false
	}
	found := ChildBitsCache{}
	err := result.Decode(&found)
	if err != nil {
		fmt.Println("mongo find child bits cache Decode err", err)
		return nil, false
	}
	return &found, true
}

func initMongEnv() *options.ClientOptions {

	// old url := "mongodb+srv://knot-mongo-cluster-0.dclqni1.mongodb.net/?authSource=%24external&authMechanism=MONGODB-X509&retryWrites=true&w=majority&appName=knot-mongo-cluster-0"
	// and i replaced the cert 6/11/26
	url := "mongodb+srv://knot-mongo-cluster-0.dclqni1.mongodb.net/?authSource=%24external&authMechanism=MONGODB-X509&appName=knot-mongo-cluster-0"
	err := os.Setenv("MONGODB_URI", url)
	if err != nil {
		log.Println("Setenv err", err)
	}

	credential := options.Credential{
		AuthMechanism: "MONGODB-X509",
	}

	hh, _ := os.UserHomeDir()
	dir := hh + "/atw/"

	certificateKeyFilePath := dir + "mongo-cert.pem"

	url = url + "&tlsCertificateKeyFile=" + certificateKeyFilePath

	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	MongoClientOptions = options.Client().
		ApplyURI(url).
		SetServerAPIOptions(serverAPIOptions)
	MongoClientOptions.SetAuth(credential)

	return MongoClientOptions
}

func initIotTables() error {

	// mongoTablesInitedLock.Lock()
	// defer mongoTablesInitedLock.Unlock()
	// if mongoTablesInited {
	// 	return nil
	// }
	// mongoTablesInited = true

	ctx := context.TODO()

	var err error
	mongoClient, err = mongo.Connect(ctx, MongoClientOptions)
	if err != nil {
		log.Fatal(err)
	}
	// defer mongoClient.Disconnect(ctx)

	subscriptionsDb = mongoClient.Database("iot").Collection("subscriptions")

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	name, err := subscriptionsDb.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	_ = name
	// fmt.Println("Name of subscriptions Index Created: " + name)

	indexModel = mongo.IndexModel{
		Keys:    bson.D{{Key: "jwtid", Value: 1}},
		Options: options.Index().SetUnique(false), // many subs can have same jwtid
	}
	name, err = subscriptionsDb.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	_ = name
	// fmt.Println("Name of subscriptions Index Created: " + name)

	indexModel = mongo.IndexModel{
		Keys:    bson.D{{Key: "own", Value: 1}},
		Options: options.Index().SetUnique(false), // many subs can have same owner
	}
	name, err = subscriptionsDb.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	_ = name
	// fmt.Println("Name of subscriptions Index Created: " + name)

	// now do the tokens
	// now do the tokens
	// now do the tokens
	savedTokensColl := mongoClient.Database("iot").Collection("saved-tokens")
	indexModel = mongo.IndexModel{
		Keys:    bson.D{{Key: "knotfreetokenpayload.jti", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	name, err = savedTokensColl.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	_ = name
	// fmt.Println("Name of tokens Index Created: " + name)

	indexModel = mongo.IndexModel{
		Keys:    bson.D{{Key: "knotfreetokenpayload.pubk", Value: 1}},
		Options: options.Index().SetUnique(false), // many tokens can have same pubk !!
	}
	name, err = savedTokensColl.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	_ = name
	// fmt.Println("Name of tokens Index Created: " + name)

	//  this is the IP address from which a free token was created.
	indexModel = mongo.IndexModel{
		Keys:    bson.D{{Key: "ip", Value: 1}},
		Options: options.Index().SetUnique(false),
	}
	name, err = savedTokensColl.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	_ = name
	// fmt.Println("Name of tokens Index Created: " + name)

	// now do the child bits cash
	// now do the child bits cash
	// now do the child bits cash
	childBitsCacheColl := mongoClient.Database("iot").Collection("child-bits-cache")
	// the schema is
	// a world name, a timestamp, and an big array of mongo bytes.

	indexModel = mongo.IndexModel{
		Keys:    bson.D{{Key: "world", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	name, err = childBitsCacheColl.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	_ = name
	// fmt.Println("Name of child bits cache Index Created: " + name)

	// that might be it.

	return nil
}
