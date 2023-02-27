// Copyright 2019,2020,2021 Alan Tracey Wootton
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

package mainhelpers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"crypto/rand"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Makes a tokens.Medium token which is 32 connections
func MakeMedium32cToken() (string, tokens.KnotFreeTokenPayload) {

	// 20 connections is about Medium
	// see tokens.Medium

	// caller must do this:
	// tokens.LoadPublicKeys()
	// tokens.LoadPrivateKeys("~/atw/privateKeys4.txt")

	fmt.Println("in MakeMedium32cToken")

	// tokenRequest := &tokens.TokenRequest{}
	payload := tokens.KnotFreeTokenPayload{}
	payload.KnotFreeContactStats = tokens.GetTokenStatsAndPrice(tokens.Medium).Stats

	//payload.Connections = 20 // 2 // TODO: move into standard x-small token

	// a year - standard x-small
	payload.ExpirationTime = uint32(time.Now().Unix() + 60*60*24*365)

	//payload.Input = 1024  // 32 * 4  // TODO: move into standard x-small token
	// payload.Output = 1024 // 32 * 4 // TODO: move into standard x-small token

	payload.Issuer = "_9sh"
	payload.JWTID = tokens.GetRandomB36String()
	nonce := payload.JWTID
	_ = nonce

	//payload.Subscriptions = 20 // TODO: move into standard x-small token

	//  Host:"building_bob_bottomline_boldness.knotfree2.com:8085"
	targetSite := "knotfree.net" // "gotohere.com"
	if os.Getenv("KNOT_KUNG_FOO") == "atw" {
		// targetSite = "gotolocal.com"
	}
	payload.URL = targetSite

	exp := payload.ExpirationTime
	// if exp > uint32(time.Now().Unix()+60*60*24*365) {
	// 	// more than a year in the future not allowed now.
	// 	exp = uint32(time.Now().Unix() + 60*60*24*365)
	// 	fmt.Println("had long token ", string(payload.JWTID)) // TODO: store in db
	// }

	cost := tokens.GetTokenStatsAndPrice(tokens.Medium).Price * 12 //tokens.CalcTokenPrice(&payload, uint32(time.Now().Unix()))
	jsonstr, _ := json.Marshal(payload)
	fmt.Println("token cost is "+fmt.Sprintf("%f", cost), string(jsonstr))

	// large32x := ScaleTokenPayload(&payload, 8*32)
	// cost = tokens.CalcTokenPrice(large32x, uint32(time.Now().Unix()))
	// jsonstr, _ = json.Marshal(large32x)
	// fmt.Println("token cost is "+fmt.Sprintf("%f", cost), string(jsonstr))

	signingKey := tokens.GetPrivateKey("_9sh")
	bbb, err := tokens.MakeToken(&payload, []byte(signingKey))
	if err != nil {
		fmt.Println("Make32xLargeToken ", err)
	}
	exptime := time.Unix(int64(exp), 0)
	formatted := exptime.Format("Jan/_2/2006")

	giantToken := string(bbb)
	giantToken = "[32xlarge_token,expires:" + formatted + ",token:" + giantToken + "]"
	return giantToken, payload
}

type publishStreamer struct {
	w        http.ResponseWriter
	contact  iot.ContactStruct
	offset   int
	received []*packets.Send
}

const S3_BUCKET = "gotoherestatic"

// this works.

func TrySomeS3Stuff() {

	fmt.Println("in TrySomeS3Stuff")

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	credsName := dirname + "/atw/credentials"
	fmt.Println("Using", dirname)

	// the default location ~/.aws/credentials is not mapped by k8s
	// we use this path:
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credsName) // don't work
	//os.Setenv("AWS_CONFIG_FILE", credsName)

	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})

	fmt.Println("in TrySomeS3Stuff bottom   err ", err)

	ccc, err2 := sess.Config.Credentials.Get()
	fmt.Println("in TrySomeS3Stuff ccc ", ccc, err2)

	svc := s3.New(sess)

	input := &s3.ListBucketsInput{}
	result, err := svc.ListBuckets(input)
	_ = result
	fmt.Println("in TrySomeS3Stuff get bucket list  ", err, result)

	// S3_BUCKET aka gotoherestatic will be our cache.

	// let's amke a file
	fname := "testingFile" + getRandomString() + ".txt"
	sampleText := fname + " aaa nnn lpl tttaaa nnn lpl ttt aaa nnn lpl ttt aaa nnn lpl ttt aaa nnn lpl ttt aaa nnn lpl ttt aaa nnn lpl ttt  "

	AddBytesToS3(sess, fname, []byte(sampleText))

	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(S3_BUCKET),
		Key:    aws.String(fname),
	})
	urlStr, err := req.Presign(15 * time.Minute)

	if err != nil {
		log.Println("Failed to sign request", err)
	}

	log.Println("The URL is", urlStr)

	fmt.Println("in TrySomeS3Stuff bottom ", fname)
}

func getRandomString() string {
	var tmp [16]byte
	rand.Read(tmp[:])
	return base64.RawURLEncoding.EncodeToString(tmp[:])
}

func AddBytesToS3(s *session.Session, destFileName string, buffer []byte) error {

	// Config settings: this is where you choose the bucket, filename, content-type etc.
	// of the file you're uploading.
	_, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:             aws.String(S3_BUCKET),
		Key:                aws.String(destFileName),
		ACL:                aws.String("private"),
		Body:               bytes.NewReader(buffer),
		ContentLength:      aws.Int64(int64(len(buffer))),
		ContentType:        aws.String(http.DetectContentType(buffer)),
		ContentDisposition: aws.String("attachment"),
		//	ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileFileToS3(s *session.Session, fileDir string) error {

	// Open the file for use
	file, err := os.Open(fileDir)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)
	file.Read(buffer)

	// Config settings: this is where you choose the bucket, filename, content-type etc.
	// of the file you're uploading.
	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:             aws.String(S3_BUCKET),
		Key:                aws.String(fileDir),
		ACL:                aws.String("private"),
		Body:               bytes.NewReader(buffer),
		ContentLength:      aws.Int64(int64(len(buffer))),
		ContentType:        aws.String(http.DetectContentType(buffer)),
		ContentDisposition: aws.String("attachment"),
		//ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

func XXXScaleTokenPayload(token *tokens.KnotFreeTokenPayload, scale float64) *tokens.KnotFreeTokenPayload {
	scaled := tokens.KnotFreeTokenPayload{}

	scaled.ExpirationTime = token.ExpirationTime // unix seconds
	scaled.Issuer = token.Issuer                 // first 4 bytes (or more) of base64 public key of issuer
	scaled.JWTID = token.JWTID                   // a unique serial number for this Issuer

	scaled.KnotFreeContactStats.Input = token.KnotFreeContactStats.Input                 // bytes per sec
	scaled.KnotFreeContactStats.Output = token.KnotFreeContactStats.Output               // bytes per sec
	scaled.KnotFreeContactStats.Subscriptions = token.KnotFreeContactStats.Subscriptions // seconds per sec
	scaled.KnotFreeContactStats.Connections = token.KnotFreeContactStats.Connections     // limits on what we're allowed to do.

	scaled.URL = token.URL // address of the service eg. "knotfree.net" or knotfree0.com for localhost

	// the meat:
	scaled.KnotFreeContactStats.Input *= scale
	scaled.KnotFreeContactStats.Output *= scale
	scaled.KnotFreeContactStats.Subscriptions *= scale
	scaled.KnotFreeContactStats.Connections *= scale

	scaled.KnotFreeContactStats.Input = float64(math.Floor(float64(scaled.KnotFreeContactStats.Input)))
	scaled.KnotFreeContactStats.Output = float64(math.Floor(float64(scaled.KnotFreeContactStats.Output)))
	scaled.KnotFreeContactStats.Subscriptions = float64(math.Floor(float64(scaled.KnotFreeContactStats.Subscriptions)))
	scaled.KnotFreeContactStats.Connections = float64(math.Floor(float64(scaled.KnotFreeContactStats.Connections)))

	return &scaled
}
