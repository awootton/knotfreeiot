package iot

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/monitor_pod"
	"github.com/awootton/knotfreeiot/tokens"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/nacl/box"
)

func HandleByProxy(w http.ResponseWriter, r *http.Request, ex *Executive, subDomain string, theHost string, proxyName string, requestURI string) {

	requestedType := r.Header.Get("Accept")

	isHtml := strings.HasSuffix(requestURI, ".html") || requestURI == "/"
	if isHtml {
		requestedType = "text/html"
	}
	// this really stinks but it's the only way to get the right content type
	if strings.HasSuffix(requestURI, ".js") {
		requestedType = "application/javascript"
	}

	fmt.Println("HandleByProxy ", subDomain, theHost, proxyName, requestURI)
	// fmt.Println("HandleByProxy ", r.RequestURI, r.URL.Path, r.Host, requestedType)

	url, err := url.Parse(proxyName)
	if err != nil {
		fmt.Println("url.Parse error ", proxyName)
		http.Error(w, "url.Parse "+proxyName, 500)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = url.Host
		if isHtml {
			req.Header.Add("Accept", "text/html")
		}
		if r.RequestURI == "/" {
			req.RequestURI = "/index.html"
			req.URL.Path += "index.html"
		}
		// fmt.Println("Director ", req.RequestURI, req.URL.Path, req.Host, req)
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		// for k, v := range resp.Header {
		// 	fmt.Println("modifyResponse kv", k, v)
		// }
		resp.Header.Set("Content-Type", requestedType)
		// this is for github nastiness. I don't think they want to be proxied
		resp.Header.Del("content-security-policy") // why does github set this?

		return nil
	}

	fmt.Println("HandleByProxy proxy.ServeHTTP")

	proxy.ServeHTTP(w, r)

}

// func AAAotherFail_HandleByProxy(w http.ResponseWriter, r *http.Request, ex *Executive, subDomain string, theHost string, proxyName string, requestURI string) {

// 	// wasHtml := strings.HasSuffix(proxyName, "/index.html")
// 	// _ = wasHtml
// 	url, err := url.Parse(proxyName + requestURI)
// 	if err != nil {
// 		fmt.Println("url.Parse error ", proxyName)
// 		http.Error(w, "url.Parse "+proxyName, 500)
// 		return
// 	}

// 	// r.URL.Path = requestURI
// 	r.RequestURI = requestURI

// 	// proxy := httputil.NewSingleHostReverseProxy(url)
// 	proxy := &httputil.ReverseProxy{
// 		Rewrite: func(r *httputil.ProxyRequest) {
// 			r.SetURL(url)
// 			r.Out.Host = r.In.Host // if desired
// 		},
// 	}
// 	proxy.ServeHTTP(w, r)

// }

func AtwHandleByProxy(w http.ResponseWriter, r *http.Request, ex *Executive, subDomain string, theHost string, proxyName string, requestURI string) {

	urlString := proxyName + requestURI
	wasHtml := strings.HasSuffix(urlString, "/index.html")

	// url, err := url.Parse(urlString)
	// if err != nil {
	// 	fmt.Println("url.Parse error ", urlString)
	// 	http.Error(w, "url.Parse "+urlString, 500)
	// 	return

	//proxy := httputil.NewSingleHostReverseProxy(url)
	// proxy := &httputil.ReverseProxy{
	// 	Rewrite: func(r *httputil.ProxyRequest) {
	// 		r.SetURL(url)
	// 		r.Out.Host = r.In.Host // if desired
	// 	},
	// }
	// proxy.ServeHTTP(w, r)

	// it's a static file. Just do a get.
	// resp, err := http.Get(urlString)
	newRequest, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		fmt.Println("http.Get error ", urlString)
		http.Error(w, "http.Get "+urlString, 500)
		return
	}
	// copy headers
	for key, val := range r.Header {
		for i := 0; i < len(val); i++ {
			newRequest.Header.Add(key, val[i])
		}
	}
	if wasHtml {
		newRequest.Header.Add("Accept", "text/html")
		// 	newRequest.Header.Del("Accept-Encoding")
	}
	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		fmt.Println("http.Get ", urlString)
		http.Error(w, "http.Get "+urlString, 500)
		return
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body) // what if it's huge?
	if err != nil {
		fmt.Println("http.Get io.ReadAllerror ", urlString)
		http.Error(w, "http.Get io.ReadAll "+urlString, 500)
		return
	}
	// if wasHtml {
	// 	resp.Header.Set("Content-Type", "text/html")
	// }
	fmt.Println("subdomain Content-Type ", r.Header.Get("Content-Type"), " for ", urlString)
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Header().Set("Content-Encoding", resp.Header.Get("Content-Encoding"))
	if wasHtml {
		w.Header().Set("Content-Type", "text/html")
	}
	// if wasHtml {
	// 	fmt.Println("subdomain body ", string(body))
	// }
	w.Write(body)
}

func StartAServer(name string, personPubk string) {
	c := monitor_pod.ThingContext{}
	c.Topic = name //"get-unix-time"
	c.CommandMap = make(map[string]monitor_pod.Command)
	c.Index = 0
	c.Token, _ = tokens.GetImpromptuGiantTokenLocal(personPubk, "")
	c.LogMeVerbose = true
	c.Host = "localhost" + ":8384" //
	fmt.Println("monitor main c.Host", c.Host)
	monitor_pod.ServeGetTime(c.Token, &c)
}

func GetIPAdress(r *http.Request) string {
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		addresses := strings.Split(r.Header.Get(h), ",")
		// march from right to left until we get a public address
		// that will be the address right before our proxy.
		for i := len(addresses) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(addresses[i])
			// header can contain spaces too, strip those out.
			realIP := net.ParseIP(ip)
			if !realIP.IsGlobalUnicast() || isPrivateSubnet(realIP) {
				// bad address, go to next
				continue
			}
			return ip
		}
	}
	return ""
}

// Thanks to  https://husobee.github.io/golang/ip-address/2015/12/17/remote-ip-go.html

// ipRange - a structure that holds the start and end of a range of ip addresses
type ipRange struct {
	start net.IP
	end   net.IP
}

// inRange - check to see if a given ip address is within a range given
func inRange(r ipRange, ipAddress net.IP) bool {
	// strcmp type byte comparison
	if bytes.Compare(ipAddress, r.start) >= 0 && bytes.Compare(ipAddress, r.end) < 0 {
		return true
	}
	return false
}

var privateRanges = []ipRange{
	{
		start: net.ParseIP("10.0.0.0"),
		end:   net.ParseIP("10.255.255.255"),
	},
	{
		start: net.ParseIP("100.64.0.0"),
		end:   net.ParseIP("100.127.255.255"),
	},
	{
		start: net.ParseIP("172.16.0.0"),
		end:   net.ParseIP("172.31.255.255"),
	},
	{
		start: net.ParseIP("192.0.0.0"),
		end:   net.ParseIP("192.0.0.255"),
	},
	{
		start: net.ParseIP("192.168.0.0"),
		end:   net.ParseIP("192.168.255.255"),
	},
	{
		start: net.ParseIP("198.18.0.0"),
		end:   net.ParseIP("198.19.255.255"),
	},
}

// isPrivateSubnet - check to see if this ip is in a private subnet
func isPrivateSubnet(ipAddress net.IP) bool {
	// my use case is only concerned with ipv4 atm
	if ipCheck := ipAddress.To4(); ipCheck != nil {
		// iterate over all our ranges
		for _, r := range privateRanges {
			// check if this ip is in a private range
			if inRange(r, ipAddress) {
				return true
			}
		}
	}
	return false
}

var bootTimeSec int64 = 0
var tokensServed int64 = 0

func (api ApiHandler) ServeMakeToken(w http.ResponseWriter, req *http.Request) {

	if bootTimeSec == 0 {
		bootTimeSec = time.Now().Unix()
	}

	remoteAddr := req.RemoteAddr
	parts := strings.Split(remoteAddr, ":") // eg [::1]:49326
	// lose the port
	parts = parts[0 : len(parts)-1]
	remoteAddr = strings.Join(parts, ":")
	// remoteAddr += req.Header.Get("HTTP_X_FORWARDED_FOR")

	ctx := req.Context()

	tokenRequest := &tokens.TokenRequest{}
	err := json.NewDecoder(req.Body).Decode(tokenRequest)

	if err != nil {
		BadTokenRequests.Inc()
		http.Error(w, err.Error(), 500)
		return
	}

	clientPublicKey := tokenRequest.Pubk
	if len(clientPublicKey) < 43 {
		BadTokenRequests.Inc()
		http.Error(w, "bad client key", 500)
		return
	}

	// what is IP or id of sender?
	tmp := GetIPAdress(req)
	if tmp != "" {
		remoteAddr = tmp
	}
	if remoteAddr == "127.0.0.1" {
		// let's fake address for testing
		remoteAddr += "+" + fmt.Sprint(time.Now().Unix()%10)
	}
	fmt.Println("token req RemoteAddr", remoteAddr)

	// check mongo
	InitMongEnv()
	InitIotTables()

	client, err := mongo.Connect(ctx, MongoClientOptions)
	if err != nil {
		BadTokenRequests.Inc()
		http.Error(w, "bad mongo.Connect", 500)
		return
	}
	defer client.Disconnect(ctx)

	saved_tokens := client.Database("iot").Collection("saved-tokens")
	// get the toks for an ip
	filter := bson.D{{Key: "ip", Value: remoteAddr}}
	cursor, err := saved_tokens.Find(context.TODO(), filter)
	if err != nil {
		BadTokenRequests.Inc()
		fmt.Println("saved_tokens find err", err.Error())
		http.Error(w, err.Error(), 500)
		return
	}
	defer cursor.Close(context.TODO())
	gottokens := make([]*SavedToken, 0)
	for cursor.Next(context.TODO()) {
		var result SavedToken
		err := cursor.Decode(&result)
		if err != nil {
			BadTokenRequests.Inc()
			fmt.Println("saved_tokens cursor err", err.Error())
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Println("found saved token ", result.KnotFreeTokenPayload.JWTID, result.IpAddress, result.ExpirationTime)
		gottokens = append(gottokens, &result)
	}
	foundPrevious := false
	previousJtid := ""
	if len(gottokens) > 0 {
		foundPrevious = true
		sort.Slice(gottokens, func(i, j int) bool {
			return gottokens[i].ExpirationTime > gottokens[j].ExpirationTime
		})
		threeMonths := uint32(60 * 60 * 24 * 90)
		if false && gottokens[0].ExpirationTime > (uint32(time.Now().Unix())+threeMonths) {
			// we have a token that's good for 3 months
			// return it
			nonce := gottokens[0].JWTID
			fmt.Println("returning found token ", gottokens[0].JWTID, gottokens[0].IpAddress, gottokens[0].ExpirationTime)
			api.signAndReturnToken(w, gottokens[0].KnotFreeTokenPayload, gottokens[0].ExpirationTime,
				*tokenRequest, nonce, clientPublicKey)
			return
		}
		previousJtid = gottokens[0].JWTID // use the same one
	}
	// else upgrade it.

	now := time.Now().Unix()
	numberOfMinutesPassed := (now - bootTimeSec) / 6 // now it's 10 sec
	if tokensServed > numberOfMinutesPassed {
		BadTokenRequests.Inc()
		http.Error(w, "Token dispenser is too busy now. Try in a minute, or, you could subscribe and get better tokens", 500)
		return
	}
	if numberOfMinutesPassed > 60 {
		// reset the allocator every hour
		bootTimeSec = now
		tokensServed = 0
	}

	// not using the payload . we always hand out Tiny4

	payload := tokens.KnotFreeTokenPayload{}

	payload.Issuer = tokens.GetPrivateKeyPrefix(0) //"_9sh"
	if foundPrevious {
		payload.JWTID = previousJtid
	} else {
		payload.JWTID = tokens.GetRandomB36String()
	}
	nonce := payload.JWTID
	payload.ExpirationTime = uint32(time.Now().Unix()) + 60*60*24*90 // 3 months
	payload.Pubk = clientPublicKey

	priceThing := tokens.GetTokenStatsAndPrice(tokens.TinyX4)
	payload.KnotFreeContactStats = priceThing.Stats

	parts = strings.Split(req.Host, ".")
	partslen := len(parts)
	if partslen < 2 {
		parts = strings.Split("local.localhost", ".")
		partslen = len(parts)
	}
	targetSite := parts[partslen-2] + "." + parts[partslen-1]

	payload.URL = targetSite

	exp := payload.ExpirationTime
	if exp > uint32(time.Now().Unix()+60*60*24*365) {
		// more than a year in the future not allowed now.
		exp = uint32(time.Now().Unix() + 60*60*24*365)
		fmt.Println("had long token ", string(payload.JWTID)) // TODO: store in db
	}

	cost := priceThing.Price // tokens.CalcTokenPrice(&payload, uint32(time.Now().Unix()))
	fmt.Println("token cost is " + fmt.Sprintf("%f", cost))

	// if cost > 0.012 {
	// 	http.Error(w, "token too expensive at "+fmt.Sprintf("%f", cost), 500)
	// 	return
	// }

	fmt.Println("returning new token ", payload.JWTID, remoteAddr, payload.ExpirationTime)

	err = api.signAndReturnToken(w, payload, exp, *tokenRequest, nonce, clientPublicKey)
	if err == nil {
		saved_token := &SavedToken{}
		saved_token.KnotFreeTokenPayload = payload
		saved_token.IpAddress = remoteAddr
		// filter is: bson.D{{Key: "ip", Value: remoteAddr}}
		if foundPrevious {
			// replace the old one
			_, err = saved_tokens.ReplaceOne(context.TODO(), filter, saved_token)
			fmt.Println("saved_tokens ReplaceOne err", err)
		} else {
			// insert a new one
			_, err = saved_tokens.InsertOne(context.TODO(), saved_token)
			fmt.Println("saved_tokens InsertOne err", err)
		}

		if err != nil {
			fmt.Println("saved_tokens insert/replace err", err.Error())
			BadTokenRequests.Inc()
			// too late for this http.Error(w, err.Error(), 500)
			return
		}
	}
}

// clientPublicKey is base64 encoded
func (api ApiHandler) signAndReturnToken(w http.ResponseWriter, payload tokens.KnotFreeTokenPayload, exp uint32,
	tokenRequest tokens.TokenRequest, nonce string, clientPublicKey string) error {
	signingKey := tokens.GetPrivateKeyWhole(0)
	tokenString, err := tokens.MakeToken(&payload, []byte(signingKey))
	if err != nil {
		return err
	}
	signingKey = "unused now"

	when := time.Unix(int64(exp), 0)
	year, month, day := when.Date()

	comments := make([]interface{}, 3)
	tmp := fmt.Sprintf(" expires: %v-%v-%v", year, int(month), day)
	comments[0] = tokenRequest.Comment + tmp
	comments[1] = "" //payload
	comments[2] = string(tokenString)
	//returnval, err := json.Marshal(comments)
	_ = err
	//returnval = []byte(strings.ReplaceAll(string(returnval), `"`, ``))
	// returnval = []byte(strings.ReplaceAll(string(returnval), ` `, `_`))
	//fmt.Println("sending token package ", string(returnval))

	returnval := tokenString

	// err = tokens.LogNewToken(ctx, &payload, remoteAddr)
	// if err != nil {
	// 	BadTokenRequests.Inc()
	// 	http.Error(w, err.Error(), 500)
	// 	return
	// }
	// box it up
	boxout := make([]byte, len(returnval)+box.Overhead)
	boxout = boxout[:0]
	var jwtid [24]byte
	copy(jwtid[:], []byte(nonce))

	var clipub [32]byte
	temp, err := base64.RawURLEncoding.DecodeString(clientPublicKey)
	_ = err
	if len(temp) != 32 {
		return fmt.Errorf("bad size, need 32 has %v", tmp)
	}
	copy(clipub[:], temp)
	sealed := box.Seal(boxout, returnval, &jwtid, &clipub, api.ce.PrivateKeyTemp)

	reply := tokens.TokenReply{}
	reply.Nonce = nonce
	reply.Pubk = hex.EncodeToString(api.ce.PublicKeyTemp[:])
	reply.Payload = hex.EncodeToString(sealed)
	bytes, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	//time.Sleep(8 * time.Second)
	// time.Sleep(1 * time.Second)
	w.Write(bytes)
	tokensServed++
	return nil
}
