package iot

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/nacl/box"
)

type ApiHandler struct {
	ce                 *ClusterExecutive
	staticStuffHandler webHandler
	// add long lived mongo connect here?
}

type SuperMux struct {
	ce *ClusterExecutive
	//super,
	sub *http.ServeMux
}

func StartPublicServer(ce *ClusterExecutive) {
	// an http server and reverse proxy.

	go startPublicServer3100()

	// for prometheus webpage
	go startPublicServer9090() // eliminate?

	// go startPublicServer8000(ce) // was for libra

	go startPublicServer9102() // promhttp.Handler for getting metrics

	go func() { // generate heartbeat
		for {
			now := ce.timegetter()
			ce.Aides[0].Heartbeat(now)
			time.Sleep(10 * time.Second)
		}
	}()

	supermux := &SuperMux{}
	supermux.ce = ce

	supermux.sub = http.NewServeMux()

	staticStuffHandler := webHandler{ce,
		http.FileServer(http.Dir("./docs"))} // FIXME: points to gotohere static assets (a react build)

	supermux.sub.Handle("/mqtt", wsAPIHandler{ce})

	supermux.sub.Handle("/", ApiHandler{ce, staticStuffHandler}) // add mongo client?

	s := &http.Server{
		Addr:           ":8085",
		Handler:        supermux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 13,
	}
	go func(s *http.Server) {
		fmt.Println("http service for " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		fmt.Println("ListenAndServe 8085 returned !!!!!  arrrrg", err)
	}(s)
}

type webHandler struct { // this is the 'staticstuff' handler. It serves the static content.
	ce *ClusterExecutive

	//	fs1 http.Handler
	fs2 http.Handler
}

// webHandler.ServeHTTP serves the static content
func (api webHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Println("webHandler ServeHTTP", r.RequestURI)

	api.fs2.ServeHTTP(w, r)

}

func startPublicServer9102() {
	fmt.Println("http metrics service 9102")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9102", nil)
	fmt.Println("http service 9102 FAIL")
}

func startPublicServer3100() {
	// an http server and reverse proxy.

	mux := http.NewServeMux()

	if true {
		origin, _ := url.Parse("http://grafana.monitoring:3100/")
		director := func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", origin.Host)
			req.URL.Scheme = "http"
			req.URL.Host = origin.Host
			//fmt.Println("fwd graf:", req.URL.Host, req.URL.Port(), req.URL.Path)
			ForwardsCount3100.Inc()
		}
		proxy := &httputil.ReverseProxy{Director: director}
		mux.Handle("/", proxy)
	}

	s := &http.Server{
		Addr:           ":3100",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 13,
	}
	go func(s *http.Server) {
		fmt.Println("http grafana service " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		fmt.Println("ListenAndServe 3100 returned !!!!!  arrrrg", err)
	}(s)

}

func startPublicServer9090() {
	// an http server and reverse proxy.

	mux := http.NewServeMux()

	if true {
		origin, _ := url.Parse("http://prometheus-operated.monitoring:9090/")
		director := func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", origin.Host)
			req.URL.Scheme = "http"
			req.URL.Host = origin.Host
			//fmt.Println("fwd prom:", req.URL.Host, req.URL.Port(), req.URL.Path)
			ForwardsCount9090.Inc()
		}
		proxy := &httputil.ReverseProxy{Director: director}
		mux.Handle("/", proxy)
	}

	s := &http.Server{
		Addr:           ":9090",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 13,
	}
	go func(s *http.Server) {
		fmt.Println("http service " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		fmt.Println("ListenAndServe 9090 returned !!!!!  arrrrg", err)
	}(s)

}

var upgrader = websocket.Upgrader{}

type wsAPIHandler struct { // this is the websocket handler
	ce *ClusterExecutive
}

func (api wsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//fmt.Println("ws ServeHTTP", r.RequestURI)

	allowAll := func(r *http.Request) bool {
		return true
	}
	upgrader.WriteBufferSize = 4096
	upgrader.ReadBufferSize = 4096
	upgrader.CheckOrigin = allowAll
	upgrader.Subprotocols = []string{"mqtt", "mqttv5", "mqttv3.1"}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	WebSocketLoop(wsConn, api.ce.Aides[0].Config)
}

func ParsePayload(httpBytes string) (string, map[string]string, string) {
	headerMap := make(map[string]string)
	pos := strings.Index(httpBytes, "\r\n\r\n")
	if pos <= 0 {
		fmt.Println("isWhat? no header end!!! this is bad ", httpBytes)
		return "", headerMap, ""
	}
	payload := httpBytes[pos+4:]
	headers := httpBytes[0:pos]
	headerParts := strings.Split(headers, "\r\n")

	firstLine := headerParts[0]
	headerParts = headerParts[1:]
	for _, head := range headerParts {
		pos = strings.Index(head, ":")
		if pos > 0 && len(head) > 3 {
			key := strings.Trim(head[0:pos], " ")
			val := strings.Trim(head[pos+1:], " ")
			headerMap[key] = val
		} else {
			fmt.Println("weird header found " + head)
		}
	}
	return firstLine, headerMap, payload
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
	if len(gottokens) > 0 {
		sort.Slice(gottokens, func(i, j int) bool {
			return gottokens[i].ExpirationTime > gottokens[j].ExpirationTime
		})
		threeMonths := uint32(60 * 60 * 24 * 90)
		if gottokens[0].ExpirationTime > (uint32(time.Now().Unix()) + threeMonths) {
			// we have a token that's good for 3 months
			// return it
			nonce := gottokens[0].JWTID
			fmt.Println("returning found token ", gottokens[0].JWTID, gottokens[0].IpAddress, gottokens[0].ExpirationTime)
			api.signAndReturnToken(w, gottokens[0].KnotFreeTokenPayload, gottokens[0].ExpirationTime,
				*tokenRequest, nonce, clientPublicKey)
			return
		}
	}

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
	payload.JWTID = tokens.GetRandomB36String()
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
		result, err := saved_tokens.InsertOne(context.TODO(), saved_token)
		if err != nil {
			BadTokenRequests.Inc()
			http.Error(w, err.Error(), 500)
			return
		}
		_ = result
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

func (superMux *SuperMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	theHost := r.Host
	if !strings.HasPrefix(theHost, "10.") {
		fmt.Println("ServeHTTP from host ", theHost)
	}
	{
		isApiRequest := strings.HasPrefix(r.RequestURI, "/api1/")
		isApiRequest = isApiRequest || r.RequestURI == "/mqtt"
		isApiRequest = isApiRequest || r.RequestURI == "/healthz"
		isApiRequest = isApiRequest || r.RequestURI == "/livez"

		if isApiRequest {
			superMux.sub.ServeHTTP(w, r)
			return
		}
	}
	//let's lose the port
	hh := strings.Split(theHost, ":")
	theHost = hh[0]

	if strings.HasPrefix(r.Host, "216.128.128.195") { // the ip address of knotfree.io
		theHost = "knotfree.io"
	}
	if strings.HasPrefix(r.Host, "216.128.128.195") { // the ip address of knotfree.org at vultr
		theHost = "knotfree.org"
	}

	if r.Host == "127.0.0.1:8085" { // localhost for testing by hackery
		theHost = "get-unix-time.localhost.com"
	}

	if strings.Contains(theHost, ".iot") {
		theHost = strings.ReplaceAll(theHost, ".iot", "_iot.knotfree.net")
	}
	if strings.Contains(theHost, ".vr") {
		theHost = strings.ReplaceAll(theHost, ".vr", "_vr.knotfree.net")
	}
	if strings.Contains(theHost, ".pod") {
		theHost = strings.ReplaceAll(theHost, ".pod", "_pod.knotfree.net")
	}
	if strings.Contains(theHost, ".test") { // for testing only - pretend .test is .iot
		theHost = strings.ReplaceAll(theHost, ".test", "_iot.knotfree.net")
	}

	domainParts := strings.Split(theHost, ".")
	// lose the tld
	tld := domainParts[len(domainParts)-1]
	domainParts = domainParts[0 : len(domainParts)-1]
	_ = tld
	// if len(domainParts) ==  { // dotted quads don't work for what's coming.
	// 	// fmt.Println("unknown host-dotted", r.Host)
	// 	// http.NotFound(w, r)
	// 	// return
	// 	superMux.sub.ServeHTTP(w, r)
	// 	return
	// }

	// eg [knotfree net]
	// eg [subdomain knotfree net]
	// eg [subdomain knotfree io]
	// fmt.Println("serving domainParts ", domainParts)

	if len(domainParts) == 2 && domainParts[0] != "www" {
		// we have a subdomain
		subDomain := domainParts[0]

		HandleHttpSubdomainRequest(w, r, superMux.ce.Aides[0], subDomain, theHost)

	} else if len(domainParts) > 2 {
		// eg get option a get-unix-time_iot knotfree
		// we don't need the h9st
		host := domainParts[len(domainParts)-1] // eg knotfree
		_ = host
		// this is the sub sub domain case and the command goes to the api of the subscription
		// this will go to the subscription aka name api
		args := domainParts[0 : len(domainParts)-2] // eg get option a
		domainParts = domainParts[len(domainParts)-2:]
		subDomain := domainParts[0]
		// subSubDomain := domainParts[0]
		//
		fmt.Println("sub sub domain ", subDomain, args)
		command := strings.Join(args, " ")
		cmd := packets.Lookup{}
		cmd.Address.FromString(subDomain)
		cmd.SetOption("cmd", []byte(command))
		// TODO: handle encoded commands.

		// send it
		reply, err := superMux.ce.ServiceContact.Get(&cmd)
		if err != nil {
			fmt.Println("sub sub domain err", err)
			http.NotFound(w, r)
			return
		}
		thePacket, ok := reply.(*packets.Send)
		if !ok {
			fmt.Println("sub sub domain not a send packet")
			http.NotFound(w, r)
			return
		}
		fmt.Println("sub sub domain reply", string(thePacket.Payload))
		w.Write(thePacket.Payload)
		return

	} //else {
	// it's not a subdomain pass it to the api.
	//}
}

func (api ApiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	if req.RequestURI != "/healthz" && req.RequestURI != "/livez" {
		fmt.Println("ApiHandler ServeHTTP", req.RequestURI, req.Host)
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")

	const proxyApiPath = "/api1/rawgithubusercontentproxy/"

	if strings.HasPrefix(req.RequestURI, proxyApiPath) {

		path := req.RequestURI[len(proxyApiPath):]

		fmt.Println("proxy path", path)
		if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") {
			w.Header().Set("Content-Type", "image/png")
		}

		resp, err := http.Get("https://raw.githubusercontent.com/" + path)
		if err != nil {
			w.Write([]byte("error " + err.Error()))
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			w.Write([]byte("error " + err.Error()))
		} else {
			w.Write(body)
		}

	} else if req.RequestURI == "/api1/getallstats" {

		stats := api.ce.Aides[0].ClusterStatsString

		w.Write([]byte(stats))

	} else if req.RequestURI == "/api1/getstats" {

		stats := api.ce.Aides[0].GetExecutiveStats()
		bytes, err := json.Marshal(stats)
		if err != nil {
			fmt.Println("GetExecutiveStats marshal", err)
		}
		w.Write(bytes)

	} else if req.RequestURI == "/api1/getToken" {

		api.ServeMakeToken(w, req)

	} else if req.RequestURI == "/api1/getPublicKey" {

		//sss := base64.RawURLEncoding.EncodeToString([]byte(tokens.FindPublicKey("yRst")))

		sss := base64.RawURLEncoding.EncodeToString(api.ce.PublicKeyTemp[:])

		fmt.Println("serve /api1/getPublicKey ", sss)

		//w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(sss))

	} else if req.RequestURI == "/api1/getGiantPassword" {

		sss := tokens.MakeRandomPhrase(14)

		w.Write([]byte(sss))

	} else if req.RequestURI == "/api1/help" {

		//	w.Header().Set("Access-Control-Allow-Origin", "*")

		sss := "/api1/getallstats\n"
		sss += "/api1/getstats\n"
		sss += "/api1/getToken\n"
		sss += "/api1/getPublicKey\n"
		sss += "/api1/getGiantPassword\n"

		w.Write([]byte(sss))

	} else if req.RequestURI == "/healthz" {

		w.Write([]byte("ok"))

	} else if req.RequestURI == "/livez" {

		w.Write([]byte("ok"))

	} else {
		// http.NotFound(w, req)
		// fmt.Fprintf(w, "expected known path "+req.RequestURI)
		// iot.HTTPServe404.Inc()
		api.staticStuffHandler.ServeHTTP(w, req)
	}

}
