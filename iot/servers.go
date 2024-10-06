package iot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/awootton/knotfreeiot/packets"
	"github.com/awootton/knotfreeiot/tokens"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	cache "github.com/victorspringer/http-cache"
	"github.com/victorspringer/http-cache/adapter/memory"
	"golang.org/x/crypto/nacl/box"
)

type ApiHandler struct {
	ce                 *ClusterExecutive
	staticStuffHandler webHandler
	// add long lived mongo connect here?

	// add cache here.
	cacheClient *cache.Client
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
	// serve another way. Serve from memory?

	memcached, err := memory.NewAdapter(
		memory.AdapterWithAlgorithm(memory.LRU),
		memory.AdapterWithCapacity(100*1024),
	)
	if err != nil {
		log.Fatal(err)
	}

	cachettl := 10 * time.Minute
	if DEBUG {
		cachettl = time.Second
	}

	cacheClient, err := cache.NewClient(
		cache.ClientWithAdapter(memcached),
		cache.ClientWithTTL(cachettl),
		cache.ClientWithRefreshKey("opn"),
	)
	if err != nil {
		log.Fatal(err)
	}

	supermux.sub.Handle("/mqtt", wsAPIHandler{ce})

	// the default handler is the ApiHandler
	supermux.sub.Handle("/", ApiHandler{ce, staticStuffHandler, cacheClient}) // add mongo client?

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
	ce  *ClusterExecutive
	fs2 http.Handler
}

// webHandler.ServeHTTP serves the static content
func (api webHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Println("webHandler ServeHTTP", r.Host, r.RequestURI)

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

	// should we just do all the TLDs here?
	if strings.Contains(theHost, ".xyz") {
		theHost = strings.ReplaceAll(theHost, ".xyz", "_xyz.knotfree.net")
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

		return

	} else if len(domainParts) > 2 {

		// sub sub domain request invokes the lookup api on the name.
		// eg get option a get-unix-time_iot knotfree
		// we don't need the host
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
		reply, err := superMux.ce.PacketService.GetPacketReply(&cmd)
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
	superMux.sub.ServeHTTP(w, r)
}

func (api ApiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	if req.RequestURI != "/healthz" && req.RequestURI != "/livez" {
		tmp := req.RequestURI
		if len(tmp) > 100 {
			tmp = tmp[0:100]
		}
		fmt.Println("ApiHandler ServeHTTP", tmp, req.Host)
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")

	const proxyApiPath = "/api1/rawgithubusercontentproxy/"

	if strings.HasPrefix(req.RequestURI, proxyApiPath) {

		path := req.RequestURI[len(proxyApiPath):]

		// fmt.Println("proxy path", path)
		if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") {
			w.Header().Set("Content-Type", "image/png")
		}

		// DONE: build a cache and don't fetch the same thing twice in the same 10 minutes.

		wholeUrl := "https://raw.githubusercontent.com/" + path
		fmt.Println("proxying to ", wholeUrl)

		if true {

			handler2 := api.cacheClient.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// do something with the response
				fmt.Println("cacheClient.Middleware", r.RequestURI)
				// fmt.Println("cacheClient.Middleware", r.URL)
				resp, err := http.Get(wholeUrl)
				if err != nil {
					fmt.Println("rawgithubusercontentproxy failed to fetch ", wholeUrl)
					w.Write([]byte("error " + err.Error()))
					return
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					w.Write([]byte("error " + err.Error()))
				} else {
					w.Write(body)
				}

			}))
			req.URL, _ = url.Parse(wholeUrl)
			handler2.ServeHTTP(w, req)

		} else {
			// old way
			resp, err := http.Get(wholeUrl)
			if err != nil {
				fmt.Println("rawgithubusercontentproxy failed to fetch ", wholeUrl)
				w.Write([]byte("error " + err.Error()))
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				w.Write([]byte("error " + err.Error()))
			} else {
				w.Write(body)
			}
		}

		return
	}
	path := strings.Split(req.RequestURI, "?")[0]
	// switch here? TODO: switch
	// mo., really. make this into a switch statement
	if path == "/api1/getallstats" {

		stats := api.ce.Aides[0].ClusterStatsString

		w.Write([]byte(stats))

	} else if path == "/api1/getstats" {

		stats := api.ce.Aides[0].GetExecutiveStats()
		bytes, err := json.Marshal(stats)
		if err != nil {
			fmt.Println("GetExecutiveStats marshal", err)
		}
		w.Write(bytes)

	} else if path == "/api1/getToken" {

		api.ServeMakeToken(w, req)

	} else if path == "/api1/getPublicKey" {

		//sss := base64.RawURLEncoding.EncodeToString([]byte(tokens.FindPublicKey("yRst")))

		sss := base64.RawURLEncoding.EncodeToString(api.ce.PublicKeyTemp[:])

		fmt.Println("serve /api1/getPublicKey ", sss)

		//w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(sss))

	} else if path == "/api1/getGiantPassword" {

		sss := tokens.MakeRandomPhrase(14)

		w.Write([]byte(sss))

	} else if path == "/api1/help" {

		//	w.Header().Set("Access-Control-Allow-Origin", "*")

		sss := "/api1/getallstats\n"
		sss += "/api1/getstats\n"
		sss += "/api1/getToken\n"
		sss += "/api1/getPublicKey\n"
		sss += "/api1/getGiantPassword\n"
		sss += "/api1/getNames\n"
		sss += "/api1/getNameStatus\n"
		sss += "/api1/getNameDetail\n"

		w.Write([]byte(sss))

	} else if path == "/healthz" {

		w.Write([]byte("ok"))

	} else if path == "/livez" {

		w.Write([]byte("ok"))

	} else if path == "/api1/getNameStatus" {

		// don't use this. use the nameService
		name := req.URL.Query().Get("name")
		fmt.Println("have getNameStatus", name)

		look := packets.Lookup{}
		look.Address.FromString(name)
		look.SetOption("cmd", []byte("exists"))
		val, err := api.ce.PacketService.GetPacketReply(&look)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if val == nil {
			http.Error(w, "no reply", 500)
			return
		}
		str := string(val.(*packets.Send).Payload)
		w.Write([]byte(str))

	} else if path == "/api1/getNames" {

		//  see nameServices.go

		// get a list of WatchedItems for an owner pubk, from the mongo db
		cmd := req.URL.Query().Get("cmd")
		nonceStr := req.URL.Query().Get("nonce")
		ourPrivK := api.ce.PrivateKeyTemp
		theirPubk := req.URL.Query().Get("pubk")
		_ = ourPrivK

		fmt.Println("getNames cmd", cmd)
		fmt.Println("getNames theirPubk", theirPubk)

		// we need to unbox this
		bincmd, err := base64.RawURLEncoding.DecodeString(cmd)
		if err != nil {
			fmt.Println("getNames decode cmd", err)
			http.Error(w, err.Error(), 500)
			return
		}
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])
		openbuffer := make([]byte, 0, (len(cmd))) // - box.Overhead))
		tmp, err := base64.RawURLEncoding.DecodeString(theirPubk)
		if err != nil {
			fmt.Println("getNames decode pubk", err)
			http.Error(w, err.Error(), 500)
			return
		}
		pubk := new([32]byte)
		copy(pubk[:], tmp[:])
		opened, ok := box.Open(openbuffer, bincmd, nonce, pubk, api.ce.PrivateKeyTemp)
		if !ok {
			fmt.Println("getNames box open failed", nonceStr, theirPubk, ourPrivK)
			http.Error(w, "box open failed", 500)
			return
		}
		parts := strings.Split(string(opened), "#")
		if len(parts) != 2 {
			fmt.Println("getNames parts len != 2")
			http.Error(w, "parts len != 2", 500)
			return
		}
		if parts[0] != theirPubk {
			fmt.Println("pubk not match")
			http.Error(w, "pubk not match", 500)
			return
		}
		timeStr := parts[1]
		seconds, err := strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			fmt.Println("time not int")
			http.Error(w, "time not int", 500)
			return
		}
		delta := time.Now().Unix() - seconds
		if delta < 0 {
			delta = -delta
		}
		if delta > 10 {
			fmt.Println("time not match")
			http.Error(w, "time not match", 500)
			return
		}

		list, err := GetSubscriptionList(theirPubk)
		if err != nil {
			fmt.Println("getNames GetSubscriptionList", err)
			http.Error(w, err.Error(), 500)
			return
		}

		jsonList, err := json.Marshal(list)
		if err != nil {
			fmt.Println("getNames json.Marshal", err)
			http.Error(w, err.Error(), 500)
			return
		}

		// fmt.Println("getNames found ", string(jsonList)) //
		//now we must encrypt the answer

		payload := string(jsonList)
		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		privk := api.ce.PrivateKeyTemp
		devicePublicKey := pubk
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, privk)

		sealedb64 := base64.RawURLEncoding.EncodeToString(sealed)
		w.Write([]byte(sealedb64)) // agile rules say no binary

	} else if path == "/api1/nameService" {

		api.NameService(w, req)

	} else { // default:
		//  This might be unnecessary but I want to see the path if it fails.
		if req.RequestURI == "/index.html" || req.RequestURI == "/" {
			indexHtml := getIndexHtml()
			w.Write([]byte(indexHtml))
		} else {
			api.staticStuffHandler.ServeHTTP(w, req)
		}
	}
}

var indexHtml []byte
var indexHtmlLock sync.Mutex

func getIndexHtml() []byte {
	indexHtmlLock.Lock()
	defer indexHtmlLock.Unlock()
	if len(indexHtml) != 0 {
		return indexHtml
	}
	cwd, _ := os.Getwd()
	fmt.Println("getIndexHtml cwd", cwd)
	var err error
	indexHtml, err = os.ReadFile("./docs/index.html")
	if err != nil {
		fmt.Println("getIndexHtml err", err)
	}
	return indexHtml
}
