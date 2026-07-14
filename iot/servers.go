package iot

import (
	"encoding/base64"
	"encoding/json"
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
	staticStuffHandler webHandler // move this to the supermux? or remove it and just use the static handler in the supermux?
	// add long lived mongo connect here?
	// add cache here?
	cacheClient *cache.Client
}

type SuperMux struct {
	ce *ClusterExecutive
	//super,
	sub                        *http.ServeMux
	staticStuffHandlerGotohere *webHandler
}

func StartPublicServer(ce *ClusterExecutive) {
	// an http server and reverse proxy.

	go startPublicServer3100()

	// for prometheus webpage
	// go startPublicServer9090() // eliminate? for now.

	// go startPublicServer8000(ce) // was for libra

	go startPublicServer9102() // promhttp.Handler for getting metrics

	go func() { // generate heartbeat
		for {
			now := ce.timegetter()
			ce.Aides[0].Heartbeat(now)
			time.Sleep(10 * time.Second)
		}
	}()

	// Can someone please clean this up? It's a mess. We have multiple http servers, some for reverse proxy, some for metrics, some for the api, and it's all over the place. We should have one http server and use different handlers for different paths. And we should use a router instead of manually checking the paths. And we should use a proper logging library instead of log.Println. And we should handle errors properly instead of just printing them. And we should... well, you get the idea.
	// every time I touch it, it breaks.
	staticStuffHandlerGotohere := webHandler{ce,
		http.FileServer(http.Dir("./gotohere-static-react-build"))} // FIXME: points to   (a react build)
	// see the missnamed KnotOperator function which does the react builds.
	// and copies assets.
	// serve another way. Serve from memory?

	supermux := &SuperMux{}
	supermux.ce = ce
	supermux.staticStuffHandlerGotohere = &staticStuffHandlerGotohere

	supermux.sub = http.NewServeMux()

	staticStuffHandler := webHandler{ce,
		http.FileServer(http.Dir("./docs"))} // FIXME: points to knotfree.net static assets (a react build)
	// see the missnamed KnotOperator function which does the react builds.
	// and copies assets.
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
		log.Println("StartPublicServer for " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		log.Println("ListenAndServe 8085 returned !!!!!  arrrrg", err)
	}(s)
}

type webHandler struct { // this is the 'staticstuff' handler. It serves the static content.
	ce  *ClusterExecutive
	fs2 http.Handler
}

// webHandler.ServeHTTP serves the static content
func (api webHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	req := r.RequestURI
	// let's kill some easy ones. Move these to supermux. Make a map.
	if strings.HasPrefix(req, "/wp-") ||
		strings.HasPrefix(req, "/zoro") ||
		strings.Contains(req, ".php") {
		http.NotFound(w, r)
		return
	}

	log.Println("webHandler ServeHTTP", r.Host, r.RequestURI)

	// it's only 15 meg. Can we just read it all in and serve it from memory?

	api.fs2.ServeHTTP(w, r)
}

func startPublicServer9102() {
	log.Println("http metrics service 9102")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9102", nil)
	log.Println("startPublicServer9102 9102 FAIL")
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
			//log.Println("fwd graf:", req.URL.Host, req.URL.Port(), req.URL.Path)
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
		log.Println("http grafana service " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		log.Println("ListenAndServe 3100 returned !!!!!  arrrrg", err)
	}(s)

}

func XstartPublicServer9090() {
	// an http server and reverse proxy.

	mux := http.NewServeMux()

	if true {
		origin, _ := url.Parse("http://prometheus-operated.monitoring:9090/")
		director := func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", origin.Host)
			req.URL.Scheme = "http"
			req.URL.Host = origin.Host
			//log.Println("fwd prom:", req.URL.Host, req.URL.Port(), req.URL.Path)
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
		log.Println("http public service 9090 " + s.Addr)
		err := s.ListenAndServe()
		_ = err
		log.Println("ListenAndServe 9090 returned !!!!!  arrrrg", err)
	}(s)

}

var upgrader = websocket.Upgrader{}

type wsAPIHandler struct { // this is the websocket handler
	ce *ClusterExecutive
}

func (api wsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//log.Println("ws ServeHTTP", r.RequestURI)

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
		log.Println("isWhat? no header end!!! this is bad ", httpBytes)
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
			log.Println("weird header found " + head)
		}
	}
	return firstLine, headerMap, payload
}

func (superMux *SuperMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	theHost := r.Host
	if !strings.HasPrefix(theHost, "10.") {
		// boring log.Println("ServeHTTP from host ", theHost)
	}

	if r.RequestURI == "/healthz" || r.RequestURI == "/livez" {
		w.Write([]byte("ok"))
		return
	}

	// let's kill some easy ones. Move these to supermux. Make a map?
	// keep these out of the rest of the logs and keep them out of the static handlers which
	// may cause a lot of disk seeking. A Prefix Tree (Trie) or an Aho-Corasick state ??

	if strings.HasPrefix(r.RequestURI, "/wp-") ||
		strings.HasPrefix(r.RequestURI, "/zoro") ||
		strings.HasPrefix(r.RequestURI, "/.aws/") ||
		strings.HasPrefix(r.RequestURI, "/.docker/") ||
		strings.HasPrefix(r.RequestURI, "/.gcp/") ||
		strings.HasPrefix(r.RequestURI, "/google") ||
		strings.HasPrefix(r.RequestURI, "/root/") ||
		strings.HasPrefix(r.RequestURI, "/home/") ||
		strings.HasPrefix(r.RequestURI, "/app/") ||
		strings.Contains(r.RequestURI, ".php") {
		http.NotFound(w, r)
		return
	}

	// always the same as ApiHandler? RequestURI,host,", r.RequestURI, r.Host)

	isApiRequest := false
	{ // TODO: use a map.
		isApiRequest = strings.Contains(r.RequestURI, "/api1/")
		isApiRequest = isApiRequest || r.RequestURI == "/mqtt"
		isApiRequest = isApiRequest || r.RequestURI == "/healthz"
		isApiRequest = isApiRequest || r.RequestURI == "/livez"
	}

	if strings.Contains(theHost, "gotohere.") {
		log.Println("ServeHTTP for gotohere ", theHost)
		// is this slow? Is it messing us up?
		// TODO: recurse through the directory and reject everyone else.
		superMux.staticStuffHandlerGotohere.ServeHTTP(w, r)
		return
	}

	if isApiRequest {
		// show the query string parameters for debugging
		// log.Println("ServeHTTP api request ", r.RequestURI, theHost)
		superMux.sub.ServeHTTP(w, r)
		return
	}

	log.Println("ServeHTTP for SUBDOMAIN request ", theHost, r.RequestURI)

	// Let's do this the other way arounnd.
	// we will subdomain ALL the TLDs except knotfree hosts which are literally knotfree.xxx
	domainParts := strings.Split(theHost, ".")
	{ // lose the port
		hh := strings.Split(domainParts[len(domainParts)-1], ":")
		domainParts[len(domainParts)-1] = hh[0]
	}
	if len(domainParts) >= 2 {
		if domainParts[len(domainParts)-2] == "knotfree" {
			// remove the knotfree
			domainParts = domainParts[0 : len(domainParts)-2]
			// if there's nothing before the knotfree.xxx or its www then let it fail in the api.
			// TODO: maybe we don't need the isApiRequest checks.
			if len(domainParts) == 0 || (len(domainParts) == 1 && domainParts[0] == "www") {
				superMux.sub.ServeHTTP(w, r)
				return
			}
		}
	}

	subDomain := strings.Join(domainParts, "_")

	HandleHttpSubdomainRequest(w, r, superMux.ce.Aides[0], subDomain, theHost)

	//}
	// else if false && len(domainParts) > 2 {

	// TODO: cleanup. If we're not using this, remove it.
	// 	// sub sub domain request invokes the lookup api on the name. Discontinued so users can use subdomains.
	// 	// eg get option a get-unix-time_iot knotfree
	// 	// we don't need the host
	// 	host := domainParts[len(domainParts)-1] // eg knotfree
	// 	_ = host
	// 	// this is the sub sub domain case and the command goes to the api of the subscription
	// 	// this will go to the subscription aka name api
	// 	args := domainParts[0 : len(domainParts)-2] // eg get option a
	// 	domainParts = domainParts[len(domainParts)-2:]
	// 	subDomain := domainParts[0]
	// 	// subSubDomain := domainParts[0]
	// 	//
	// 	log.Println("sub sub domain ", subDomain, args)
	// 	command := strings.Join(args, " ")
	// 	cmd := packets.Lookup{}
	// 	cmd.Address.FromString(subDomain)
	// 	cmd.SetOption("cmd", []byte(command))
	// 	// TODO: handle encoded commands.

	// 	// send it
	// 	reply, err := superMux.ce.PacketService.GetPacketReply(&cmd)
	// 	if err != nil {
	// 		log.Println("sub sub domain err", err)
	// 		http.NotFound(w, r)
	// 		return
	// 	}
	// 	thePacket, ok := reply.(*packets.Send)
	// 	if !ok {
	// 		log.Println("sub sub domain not a send packet")
	// 		http.NotFound(w, r)
	// 		return
	// 	}
	// 	// log.Println("sub sub domain reply", string(thePacket.Payload))
	// 	w.Write(thePacket.Payload)
	// 	return

	// }
	//else {
	// it's not a subdomain pass it to the api.
	//}
	// superMux.sub.ServeHTTP(w, r)
}

func (api ApiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	// I think nginx might be spamming me.
	// show it.

	log.Println("ApiHandler ServeHTTP RequestURI,host,", req.RequestURI, req.Host)

	if req.RequestURI != "/healthz" && req.RequestURI != "/livez" {
		tmp := req.RequestURI
		if len(tmp) > 100 {
			tmp = tmp[0:100] // why?
		}
		// this gets tedious log.Println("ApiHandler ServeHTTP", tmp, req.Host)
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")

	const proxyApiPath = "/api1/rawgithubusercontentproxy/"
	// move this proxy stuff to a less annoying place.
	if strings.HasPrefix(req.RequestURI, proxyApiPath) {

		path := req.RequestURI[len(proxyApiPath):]

		// log.Println("proxy path", path)
		if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") {
			w.Header().Set("Content-Type", "image/png")
		}

		// DONE: build a cache and don't fetch the same thing twice in the same 10 minutes.

		wholeUrl := "https://raw.githubusercontent.com/" + path
		log.Println("proxying to ", wholeUrl)

		if true {

			handler2 := api.cacheClient.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// do something with the response
				log.Println("cacheClient.Middleware", r.RequestURI)
				// log.Println("cacheClient.Middleware", r.URL)
				resp, err := http.Get(wholeUrl)
				if err != nil {
					log.Println("rawgithubusercontentproxy failed to fetch ", wholeUrl)
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
				log.Println("rawgithubusercontentproxy failed to fetch ", wholeUrl)
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

	switch path {
	case "/api1/getallstats":

		stats := api.ce.Aides[0].ClusterStatsString

		w.Write([]byte(stats))

	case "/api1/dns-query":

		// this is a dns over http query. We will parse the query and return a dns response.
		// note that this will optionally take an array of domains, and return an array of responses. This is to allow for multiple queries in one request, which is a common use case for dns over http.
		// otherwise it's supposed to look like the cloudflair one and the google one, which only allows one query per request. But we want to allow multiple queries per request to reduce latency and increase cache hits.
		domains := req.URL.Query().Get("name")
		domainList := strings.Split(domains, ",")
		if len(domainList) == 0 {
			http.Error(w, "no domains provided", 400)
			return
		}
		if len(domainList) > 64 {
			http.Error(w, "too many domains provided, 64 is the maximum", 400)
			return
		}

		recordType := req.URL.Query().Get("type")
		if recordType != "A" && recordType != "TXT" {
			http.Error(w, "unsupported type "+recordType, 400)
			return
		}

		recordTypeInt := 0
		switch recordType {
		case "A":
			recordTypeInt = 1
		case "TXT":
			recordTypeInt = 16
		}
		dnsServer := req.URL.Query().Get("dnsserver") // required unless knotfree is used, e.g., dns.gotohere.com or 1.1.1.1 etc
		// don't use dns.gotohere.com, set the knotfree=1 flag instead.
		// can we just default to cloudflair?
		if dnsServer == "" {
			dnsServer = "1.1.1.1"
		}
		isKnotfree := req.URL.Query().Get("knotfree")
		if dnsServer == "" && isKnotfree != "1" {
			http.Error(w, "no dns server provided", 400)
			return
		}
		var responses []DnsResponse
		var err error
		if isKnotfree == "1" {
			responses, err = LookupDnsOverHttpKnotfree(api.ce, domainList, recordTypeInt)
		} else {
			responses, err = LookupDnsOverHttp(domainList, recordTypeInt, dnsServer)
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if len(responses) != len(domainList) {
			http.Error(w, "responses length mismatch", 500)
			return
		}
		if len(responses) == 1 {
			jsonBytes, err := json.Marshal(responses[0])
			// log.Println("dns-query response", string(jsonBytes))
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write(jsonBytes)
		} else {
			// multiple responses. return an array of responses.
			jsonBytes, err := json.Marshal(responses)
			// too big to print
			log.Println("dns-query response bytes len", len(jsonBytes), " for", len(responses), "responses")
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Write(jsonBytes)
		}
	case "/api1/getstats":

		stats := api.ce.Aides[0].GetExecutiveStats()
		bytes, err := json.Marshal(stats)
		if err != nil {
			log.Println("GetExecutiveStats marshal", err)
		}
		w.Write(bytes)

	case "/api1/getToken":

		api.ServeMakeToken(w, req)

	case "/api1/getPublicKey":

		//sss := base64.RawURLEncoding.EncodeToString([]byte(tokens.FindPublicKey("yRst")))

		sss := base64.RawURLEncoding.EncodeToString(api.ce.PublicKeyTemp[:])

		log.Println("serve /api1/getPublicKey ", sss)

		//w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(sss))

	case "/api1/getGiantPassword":

		sss := tokens.MakeRandomPhrase(14)

		w.Write([]byte(sss))

	case "/api1/help":

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

	case "/healthz":

		// FIXME: we should send something to the service contact.
		// like the help command! or not.

		w.Write([]byte("ok"))

	case "/livez":

		// FIXME: we should send something to the service contact. or not.

		w.Write([]byte("ok"))

	case "/api1/getNameStatus":

		// don't use this. use the nameService
		name := req.URL.Query().Get("name")
		log.Println("have getNameStatus", name)

		look := packets.Lookup{}
		look.Address.FromString(name)
		look.SetOption("cmd", []byte("exists"))
		val, err := api.ce.GetPacketService().GetPacketReply(&look)
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

	case "/api1/getNames":

		//  see nameServices.go

		// get a list of WatchedItems for an owner pubk, from the mongo db
		// what if it's HUGE !!
		cmd := req.URL.Query().Get("cmd")
		nonceStr := req.URL.Query().Get("nonce")
		ourPrivK := api.ce.PrivateKeyTemp
		theirPubk := req.URL.Query().Get("pubk")
		_ = ourPrivK

		log.Println("getNames cmd", cmd)
		log.Println("getNames theirPubk", theirPubk)

		// we need to unbox this
		bincmd, err := base64.RawURLEncoding.DecodeString(cmd)
		if err != nil {
			log.Println("getNames decode cmd", err)
			http.Error(w, err.Error(), 500)
			return
		}
		nonce := new([24]byte)
		copy(nonce[:], nonceStr[:])
		openbuffer := make([]byte, 0, (len(cmd))) // - box.Overhead))
		tmp, err := base64.RawURLEncoding.DecodeString(theirPubk)
		if err != nil {
			log.Println("getNames decode pubk", err)
			http.Error(w, err.Error(), 500)
			return
		}
		pubk := new([32]byte)
		copy(pubk[:], tmp[:])
		opened, ok := box.Open(openbuffer, bincmd, nonce, pubk, api.ce.PrivateKeyTemp)
		if !ok {
			log.Println("getNames box open failed", nonceStr, theirPubk, ourPrivK)
			http.Error(w, "box open failed", 500)
			return
		}
		parts := strings.Split(string(opened), "#")
		if len(parts) != 2 {
			log.Println("getNames parts len != 2")
			http.Error(w, "parts len != 2", 500)
			return
		}
		if parts[0] != theirPubk {
			log.Println("pubk not match")
			http.Error(w, "pubk not match", 500)
			return
		}
		timeStr := parts[1]
		seconds, err := strconv.ParseInt(timeStr, 10, 64)
		if err != nil {
			log.Println("time not int")
			http.Error(w, "time not int", 500)
			return
		}
		delta := time.Now().Unix() - seconds
		if delta < 0 {
			delta = -delta
		}
		if delta > 10 {
			log.Println("time not match")
			http.Error(w, "time not match", 500)
			return
		}

		list, err := GetSubscriptionList(theirPubk)
		if err != nil {
			log.Println("getNames GetSubscriptionList", err)
			http.Error(w, err.Error(), 500)
			return
		}

		jsonList, err := json.Marshal(list)
		if err != nil {
			log.Println("getNames json.Marshal", err)
			http.Error(w, err.Error(), 500)
			return
		}

		// log.Println("getNames found ", string(jsonList)) //
		//now we must encrypt the answer

		payload := string(jsonList)
		buffer := make([]byte, 0, (len(payload) + box.Overhead))
		privk := api.ce.PrivateKeyTemp
		devicePublicKey := pubk
		sealed := box.Seal(buffer, []byte(payload), nonce, devicePublicKey, privk)

		sealedb64 := base64.RawURLEncoding.EncodeToString(sealed)
		w.Write([]byte(sealedb64)) // agile rules say no binary

	case "/api1/nameService":

		// show the query string parameters for debugging
		//log.Println("nameService query", req.URL.Query())
		api.NameService(w, req)

	case "/api1/setAllChildBitCache":

		// this is a post of a large string (36k right now).
		// remember it.

		worldName := req.URL.Query().Get("world")

		log.Println("/api1/setAllChildBitCache for world=", worldName)

		defer req.Body.Close()

		body, err := io.ReadAll(req.Body)
		if err != nil {
			w.Write([]byte("error reading all of setAllChildBitCache " + err.Error()))
		} else {
			// w.Write(body)
		}

		// save it in memory.
		WorldName2ChildbitsLock.Lock()
		WorldName2ChildbitsWholeString[worldName] = body
		whenWasLocalDataSet := WorldName2lastset[worldName]
		WorldName2lastset[worldName] = time.Now().Unix() // pretend it's brand new.
		WorldName2ChildbitsLock.Unlock()

		// how often do we save it to the db? every 3 minutes max
		// we should check if it json parses?
		if time.Now().Unix()-whenWasLocalDataSet > 60*3 {
			ok := SaveChildBitsNameAndData(worldName, string(body))
			if !ok {
				log.Println("failed to save child bits for world=", worldName)
			}
		}
		w.Write([]byte("ok"))

	case "/api1/getAllChildBitCache":

		// this is a send of a large string (36k right now).
		// send it.

		worldName := req.URL.Query().Get("world")
		_ = worldName

		log.Println("/api1/getAllChildBitCache for world=", worldName)

		WorldName2ChildbitsLock.Lock()
		wholeString, ok := WorldName2ChildbitsWholeString[worldName]
		when, ok2 := WorldName2lastset[worldName]
		WorldName2ChildbitsLock.Unlock()

		// need to call the DB. I know there's one there.
		var needDb = false
		if !ok || !ok2 { // nothing locally, need to call the DB
			needDb = true
		}
		if time.Now().Unix()-when > 5*60 {
			// it's been 5 minutes
			needDb = true
		}
		if needDb {
			// call the DB to get the latest child bits for this world
			cachedata, ok := GetChildBitsCache(worldName)
			if ok {
				log.Println("/api1/getAllChildBitCache got data from DB for world=", worldName, "len=", len(cachedata.Data))
				wholeString = []byte(cachedata.Data)
				WorldName2ChildbitsLock.Lock()
				WorldName2ChildbitsWholeString[worldName] = wholeString
				// we pretend that the dbdata is new or else we would be it every time
				// if we don't keep posting it.
				WorldName2lastset[worldName] = time.Now().Unix()
				WorldName2ChildbitsLock.Unlock()
			} else {
				wholeString = []byte{} // empty
				log.Println("/api1/getAllChildBitCache no data from DB for world=", worldName)
			}
		}
		if len(wholeString) == 0 {
			// w.Write([]byte("error no childbit cache for world " + worldName))
			// make an error here.
			http.Error(w, "no childbit cache for world "+worldName, 404)
			return
		}

		// and also, lol, this.
		log.Println("/api1/getAllChildBitCache data written will be len =", len(wholeString))

		// do I need to be working it this hard?
		n, err := w.Write(wholeString)
		for n < len(wholeString) {
			n2, err2 := w.Write(wholeString[n:])
			if err2 != nil {
				log.Println("/api1/getAllChildBitCache error writing response for world=", worldName, "err=", err2)
				break
			}
			n += n2
		}
		if err != nil {
			log.Println("/api1/getAllChildBitCache error writing response for world=", worldName, "err=", err)
		} else {
			log.Println("/api1/getAllChildBitCache sent", n, "bytes for world=", worldName)
		}

	default: // default:
		log.Println("ApiHandler ServeHTTP default query", req.RequestURI, req.Host)
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
	log.Println("getIndexHtml cwd", cwd)
	var err error
	indexHtml, err = os.ReadFile("./docs/index.html")
	if err != nil {
		log.Println("getIndexHtml err", err)
	}
	return indexHtml
}
