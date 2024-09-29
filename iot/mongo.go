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

	"github.com/awootton/knotfreeiot/tokens"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

var clientConnectLock sync.Mutex
var clientConnectDone = false
var mongoClient *mongo.Client

// GetMongoClient returns THE mongo client. Is this safe to just do once ever?
// Is it safe to call this multiple times?
func GetMongoClient() (*mongo.Client, error) {

	InitMongEnv()
	InitIotTables()

	clientConnectLock.Lock()
	defer clientConnectLock.Unlock()
	if clientConnectDone {
		return mongoClient, nil
	}
	var err error
	mongoClient, err = mongo.Connect(context.TODO(), MongoClientOptions)
	if err != nil {
		return nil, err
	}
	clientConnectDone = true
	return mongoClient, nil
}

func GetSubscriptionList(ownerPubk string) ([]WatchedTopic, error) {

	client, err := GetMongoClient()
	if err != nil {
		fmt.Println("mongo.Connect err", err)
		return nil, err
	}
	// defer client.Disconnect(ctx)

	subscriptions := client.Database("iot").Collection("subscriptions")

	filter := bson.D{{Key: "own", Value: ownerPubk}}
	cursor, err := subscriptions.Find(context.TODO(), filter)
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

	client, err := GetMongoClient()
	if err != nil {
		fmt.Println("mongo.Connect err", err)
		return 0, err
	}
	// defer client.Disconnect(ctx)

	subscriptions := client.Database("iot").Collection("subscriptions")

	filter := bson.D{{Key: "own", Value: ownerPubk}}
	cursor, err := subscriptions.Find(context.TODO(), filter)
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

func GetSubscription(hashedTopicStr string) (*WatchedTopic, bool) {

	client, err := GetMongoClient() //:= mongo.Connect(ctx, MongoClientOptions)
	if err != nil {
		fmt.Println("mongo.Connect err", err)
		return nil, false
	}
	// defer client.Disconnect(ctx)

	subscriptions := client.Database("iot").Collection("subscriptions")

	filter := bson.D{{Key: "name", Value: hashedTopicStr}}
	result := subscriptions.FindOne(context.TODO(), filter)
	if result.Err() != nil {
		// fmt.Println("mongo find name err", result.Err())
		return nil, false
	}
	found := WatchedTopic{}
	err = result.Decode(&found)
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

	client, err := GetMongoClient() //:= mongo.Connect(ctx, MongoClientOptions)
	if err != nil {
		fmt.Println("mongo.Connect err", err)
		return err
	}
	// defer client.Disconnect(ctx)

	subscriptions := client.Database("iot").Collection("subscriptions")

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
	InitMongEnv()
	InitIotTables()

	ctx := context.TODO()

	client, err := mongo.Connect(ctx, MongoClientOptions) // is this a new connection each time?
	if err != nil {
		fmt.Println("mongo.Connect err", err)
		return err
	}
	defer client.Disconnect(ctx)

	subscriptions := client.Database("iot").Collection("subscriptions")
	hashedTopicStr := watchedTopic.Name.ToBase64()
	filter := bson.D{{Key: "name", Value: hashedTopicStr}}
	result := subscriptions.FindOne(context.TODO(), filter) // I hate this.
	if result.Err() != nil {
		// not found
		// insert
		result, err := subscriptions.InsertOne(context.TODO(), watchedTopic)
		_ = result
		return err

	} else {
		// found
		// replace
		result, err := subscriptions.ReplaceOne(context.TODO(), filter, watchedTopic)
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

func InitMongEnv() *options.ClientOptions {

	mongoInitedLock.Lock()
	defer mongoInitedLock.Unlock()
	if mongoInited {
		return MongoClientOptions
	}
	mongoInited = true

	url := "mongodb+srv://knot-mongo-cluster-0.dclqni1.mongodb.net/?authSource=%24external&authMechanism=MONGODB-X509&retryWrites=true&w=majority&appName=knot-mongo-cluster-0"

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

var mongoTablesInited = false
var mongoTablesInitedLock sync.Mutex

func InitIotTables() error {

	mongoTablesInitedLock.Lock()
	defer mongoTablesInitedLock.Unlock()
	if mongoTablesInited {
		return nil
	}
	mongoTablesInited = true

	ctx := context.TODO()

	client, err := mongo.Connect(ctx, MongoClientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	subscriptions := client.Database("iot").Collection("subscriptions")

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	name, err := subscriptions.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	_ = name
	// fmt.Println("Name of subscriptions Index Created: " + name)

	indexModel = mongo.IndexModel{
		Keys:    bson.D{{Key: "jwtid", Value: 1}},
		Options: options.Index().SetUnique(false), // many subs can have same jwtid
	}
	name, err = subscriptions.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	_ = name
	// fmt.Println("Name of subscriptions Index Created: " + name)

	indexModel = mongo.IndexModel{
		Keys:    bson.D{{Key: "own", Value: 1}},
		Options: options.Index().SetUnique(false), // many subs can have same owner
	}
	name, err = subscriptions.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	_ = name
	// fmt.Println("Name of subscriptions Index Created: " + name)

	// now do the tokens
	// now do the tokens
	// now do the tokens
	savedTokensColl := client.Database("iot").Collection("saved-tokens")
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

	return nil
}
