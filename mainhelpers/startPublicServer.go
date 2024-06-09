package mainhelpers

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

// type ApiHandler struct {
// 	ce *iot.ClusterExecutive

// 	staticStuffHandler webHandler
// }

// func (api ApiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {

// 	fmt.Println("ApiHandler ServeHTTP", req.RequestURI, req.Host)

// 	w.Header().Add("Access-Control-Allow-Origin", "*")

// 	const proxyApiPath = "/api1/rawgithubusercontentproxy/"

// 	if strings.HasPrefix(req.RequestURI, proxyApiPath) {

// 		path := req.RequestURI[len(proxyApiPath):]

// 		fmt.Println("proxy path", path)
// 		if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") {
// 			w.Header().Set("Content-Type", "image/png")
// 		}

// 		resp, err := http.Get("https://raw.githubusercontent.com/" + path)
// 		if err != nil {
// 			w.Write([]byte("error " + err.Error()))
// 		}
// 		defer resp.Body.Close()

// 		body, err := io.ReadAll(resp.Body)
// 		if err != nil {
// 			w.Write([]byte("error " + err.Error()))
// 		} else {
// 			w.Write(body)
// 		}

// 	} else if req.RequestURI == "/api1/getallstats" {

// 		stats := api.ce.Aides[0].ClusterStatsString

// 		w.Write([]byte(stats))

// 	} else if req.RequestURI == "/api1/getstats" {

// 		stats := api.ce.Aides[0].GetExecutiveStats()
// 		bytes, err := json.Marshal(stats)
// 		if err != nil {
// 			fmt.Println("GetExecutiveStats marshal", err)
// 		}
// 		w.Write(bytes)

// 	} else if req.RequestURI == "/api1/getToken" {

// 		// if isLocal(req) {
// 		// 	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
// 		// } else {
// 		// 	w.Header().Set("Access-Control-Allow-Origin", "http://knotfree.io")
// 		// }
// 		api.ServeMakeToken(w, req)

// 	} else if req.RequestURI == "/api1/getPublicKey" {

// 		//sss := base64.RawURLEncoding.EncodeToString([]byte(tokens.FindPublicKey("yRst")))

// 		sss := base64.RawURLEncoding.EncodeToString(api.ce.PublicKeyTemp[:])

// 		fmt.Println("serve /api1/getPublicKey ", sss)

// 		//w.Header().Set("Access-Control-Allow-Origin", "*")
// 		w.Write([]byte(sss))

// 	} else if req.RequestURI == "/api1/getGiantPassword" {

// 		sss := tokens.MakeRandomPhrase(14)

// 		w.Write([]byte(sss))

// 	} else if req.RequestURI == "/api1/help" {

// 		//	w.Header().Set("Access-Control-Allow-Origin", "*")

// 		sss := "/api1/getallstats\n"
// 		sss += "/api1/getstats\n"
// 		sss += "/api1/getToken\n"
// 		sss += "/api1/getPublicKey\n"
// 		sss += "/api1/getGiantPassword\n"

// 		w.Write([]byte(sss))

// 	} else {
// 		// http.NotFound(w, req)
// 		// fmt.Fprintf(w, "expected known path "+req.RequestURI)
// 		// iot.HTTPServe404.Inc()
// 		api.staticStuffHandler.ServeHTTP(w, req)
// 	}
// }

// type SuperMux struct {
// 	ce *iot.ClusterExecutive
// 	//super,
// 	sub *http.ServeMux
// }

// type pinfo struct {
// 	// these are the reply buffers
// 	buff []byte
// }
// type RequestReplyStruct struct {
// 	originalRequest []byte
// 	firstLine       string
// 	replyParts      []pinfo
// }

// var servedMap = map[string]*RequestReplyStruct{}

func IsLocal(r *http.Request) bool {
	fmt.Println("host is ", r.Host)
	if os.Getenv("KNOT_KUNG_FOO") == "atw" {
		return true
	}
	if strings.Contains(r.Host, "local") {
		return true
	}
	return false
}

// func (superMux *SuperMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {

// 	//fmt.Println("giant token ", tokens.GetImpromptuGiantToken())

// 	// if !strings.Contains(r.Host, "knotfree.") {
// 	// 	fmt.Println("unknown host ", r.Host)
// 	// 	http.NotFound(w, r)
// 	// 	return
// 	// }

// 	// fmt.Println("ServeHTTP from host ", r.Host)

// 	if strings.HasPrefix(r.Host, "212.2.245.112") { // the ip address of knotfree.io
// 		r.Host = "knotfree.io"
// 	}

// 	if strings.HasPrefix(r.Host, "216.128.128.195") { // the ip address of knotfree.org at vultr
// 		r.Host = "knotfree.org"
// 	}

// 	domainParts := strings.Split(r.Host, ".")
// 	if len(domainParts) == 4 { // dotted quads don't work for what's coming.
// 		fmt.Println("unknown host-dotted", r.Host)
// 		http.NotFound(w, r)
// 		return
// 	}

// 	// eg [knotfree net]
// 	// eg [subdomain knotfree net]
// 	// eg [subdomain knotfree io]
// 	// fmt.Println("serving domainParts ", domainParts)

// 	isApiRequest := strings.HasPrefix(r.RequestURI, "/api1/") // len(r.RequestURI) >= 9 && r.RequestURI[:9] == "/api1/get"
// 	isApiRequest = isApiRequest || r.RequestURI == "/mqtt"

// 	if !isApiRequest && len(domainParts) > 2 && domainParts[0] != "www" {
// 		// we have a subdomain
// 		subDomain := domainParts[0]

// 		HandleHttpSubdomainRequest(w, r, superMux.ce.Aides[0], subDomain)

// 	} else {
// 		// it's not a subdomain pass it to the api.
// 		superMux.sub.ServeHTTP(w, r)
// 	}
// }

type ProxyHandler struct {
	p *httputil.ReverseProxy
}

func (ph *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL)
	w.Header().Set("X-Ben", "Rad")
	ph.p.ServeHTTP(w, r)
}

// func StartPublicServer(ce *iot.ClusterExecutive) {
// 	// an http server and reverse proxy.

// 	go startPublicServer3100()

// 	// for prometheus webpage
// 	go startPublicServer9090() // eliminate?

// 	// go startPublicServer8000(ce) // was for libra

// 	go startPublicServer9102() // promhttp.Handler for getting metrics

// 	go func() { // generate heartbeat
// 		for {
// 			// mt.Println("Heartbeat ", ce.Aides[0].Name)
// 			ce.Aides[0].Heartbeat(uint32(time.Now().Unix()))
// 			// fmt.Println("Heartbeat DONE", ce.Aides[0].Name)
// 			time.Sleep(10 * time.Second)
// 		}
// 	}()

// 	supermux := &SuperMux{}
// 	supermux.ce = ce

// 	supermux.sub = http.NewServeMux()

// 	staticStuffHandler := webHandler{ce,
// 		http.FileServer(http.Dir("./docs"))} // FIXME: points to gotohere static assets (a react build)

// 	supermux.sub.Handle("/mqtt", wsAPIHandler{ce})

// 	supermux.sub.Handle("/", ApiHandler{ce, staticStuffHandler})

// 	s := &http.Server{
// 		Addr:           ":8085",
// 		Handler:        supermux,
// 		ReadTimeout:    10 * time.Second,
// 		WriteTimeout:   10 * time.Second,
// 		MaxHeaderBytes: 1 << 13,
// 	}
// 	go func(s *http.Server) {
// 		fmt.Println("http service for ws " + s.Addr)
// 		err := s.ListenAndServe()
// 		_ = err
// 		fmt.Println("ListenAndServe 8085 returned !!!!!  arrrrg", err)
// 	}(s)
// }

// func startPublicServer9102() {
// 	fmt.Println("http metrics service 9102")
// 	http.Handle("/metrics", promhttp.Handler())
// 	http.ListenAndServe(":9102", nil)
// 	fmt.Println("http service 9102 FAIL")
// }

// func startPublicServer3100() {
// 	// an http server and reverse proxy.

// 	mux := http.NewServeMux()

// 	if true {
// 		origin, _ := url.Parse("http://grafana.monitoring:3100/")
// 		director := func(req *http.Request) {
// 			req.Header.Add("X-Forwarded-Host", req.Host)
// 			req.Header.Add("X-Origin-Host", origin.Host)
// 			req.URL.Scheme = "http"
// 			req.URL.Host = origin.Host
// 			//fmt.Println("fwd graf:", req.URL.Host, req.URL.Port(), req.URL.Path)
// 			iot.ForwardsCount3100.Inc()
// 		}
// 		proxy := &httputil.ReverseProxy{Director: director}
// 		mux.Handle("/", proxy)
// 	}

// 	s := &http.Server{
// 		Addr:           ":3100",
// 		Handler:        mux,
// 		ReadTimeout:    10 * time.Second,
// 		WriteTimeout:   10 * time.Second,
// 		MaxHeaderBytes: 1 << 13,
// 	}
// 	go func(s *http.Server) {
// 		fmt.Println("http grafana service " + s.Addr)
// 		err := s.ListenAndServe()
// 		_ = err
// 		fmt.Println("ListenAndServe 3100 returned !!!!!  arrrrg", err)
// 	}(s)

// }

// func startPublicServer9090() {
// 	// an http server and reverse proxy.

// 	mux := http.NewServeMux()

// 	if true {
// 		origin, _ := url.Parse("http://prometheus-operated.monitoring:9090/")
// 		director := func(req *http.Request) {
// 			req.Header.Add("X-Forwarded-Host", req.Host)
// 			req.Header.Add("X-Origin-Host", origin.Host)
// 			req.URL.Scheme = "http"
// 			req.URL.Host = origin.Host
// 			//fmt.Println("fwd prom:", req.URL.Host, req.URL.Port(), req.URL.Path)
// 			iot.ForwardsCount9090.Inc()
// 		}
// 		proxy := &httputil.ReverseProxy{Director: director}
// 		mux.Handle("/", proxy)
// 	}

// 	s := &http.Server{
// 		Addr:           ":9090",
// 		Handler:        mux,
// 		ReadTimeout:    10 * time.Second,
// 		WriteTimeout:   10 * time.Second,
// 		MaxHeaderBytes: 1 << 13,
// 	}
// 	go func(s *http.Server) {
// 		fmt.Println("http service " + s.Addr)
// 		err := s.ListenAndServe()
// 		_ = err
// 		fmt.Println("ListenAndServe 9090 returned !!!!!  arrrrg", err)
// 	}(s)

// }

// var upgrader = websocket.Upgrader{}

// type wsAPIHandler struct {
// 	ce *iot.ClusterExecutive
// }

// func (api wsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

// 	//fmt.Println("ws ServeHTTP", r.RequestURI)

// 	allowAll := func(r *http.Request) bool {
// 		return true
// 	}
// 	upgrader.WriteBufferSize = 4096
// 	upgrader.ReadBufferSize = 4096
// 	upgrader.CheckOrigin = allowAll
// 	upgrader.Subprotocols = []string{"mqtt", "mqttv5", "mqttv3.1"}

// 	wsConn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Print("upgrade:", err)
// 		return
// 	}

// 	iot.WebSocketLoop(wsConn, api.ce.Aides[0].Config)
// }

// func ParsePayload(httpBytes string) (string, map[string]string, string) {
// 	headerMap := make(map[string]string)
// 	pos := strings.Index(httpBytes, "\r\n\r\n")
// 	if pos <= 0 {
// 		fmt.Println("isWhat? no header end!!! this is bad ", httpBytes)
// 		return "", headerMap, ""
// 	}
// 	payload := httpBytes[pos+4:]
// 	headers := httpBytes[0:pos]
// 	headerParts := strings.Split(headers, "\r\n")

// 	firstLine := headerParts[0]
// 	headerParts = headerParts[1:]
// 	for _, head := range headerParts {
// 		pos = strings.Index(head, ":")
// 		if pos > 0 && len(head) > 3 {
// 			key := strings.Trim(head[0:pos], " ")
// 			val := strings.Trim(head[pos+1:], " ")
// 			headerMap[key] = val
// 		} else {
// 			fmt.Println("weird header found " + head)
// 		}
// 	}
// 	return firstLine, headerMap, payload
// }

// var bootTimeSec int64 = 0
// var tokensServed int64 = 0

// func (api ApiHandler) ServeMakeToken(w http.ResponseWriter, req *http.Request) {

// 	if bootTimeSec == 0 {
// 		bootTimeSec = time.Now().Unix()
// 	}

// 	remoteAddr := req.RemoteAddr
// 	parts := strings.Split(remoteAddr, ":") // eg [::1]:49326
// 	// lose the port
// 	parts = parts[0 : len(parts)-1]
// 	remoteAddr = strings.Join(parts, ":")
// 	// remoteAddr += req.Header.Get("HTTP_X_FORWARDED_FOR")

// 	ctx := req.Context()

// 	// what is IP or id of sender?
// 	fmt.Println("token req RemoteAddr", remoteAddr) // F I X M E: use db see below

// 	now := time.Now().Unix()
// 	numberOfMinutesPassed := (now - bootTimeSec) / 6 // now it's 10 sec
// 	if tokensServed > numberOfMinutesPassed {
// 		iot.BadTokenRequests.Inc()
// 		http.Error(w, "Token dispenser is too busy now. Try in a minute, or, you could subscribe and get better tokens", 500)
// 		return
// 	}
// 	if numberOfMinutesPassed > 60 {
// 		// reset the allocator every hour
// 		bootTimeSec = now
// 		tokensServed = 0
// 	}

// 	var buff1024 [1024]byte
// 	n, err := req.Body.Read(buff1024[:])
// 	if err != nil {
// 		iot.BadTokenRequests.Inc()
// 	}
// 	buf := buff1024[:n]
// 	fmt.Println("token request read body", string(buf), n)
// 	_ = buf

// 	tokenRequest := &tokens.TokenRequest{}
// 	err = json.Unmarshal(buf, tokenRequest)
// 	if err != nil {
// 		iot.BadTokenRequests.Inc()
// 		fmt.Println("TokenRequest err", err.Error())
// 		http.Error(w, err.Error(), 500)
// 		return
// 	} else {
// 		// todo: calc cost of this token and have limit.
// 		// move this phat routine somewhere else TODO:

// 		clientPublicKey := tokenRequest.Pkey
// 		if len(clientPublicKey) != 64 {
// 			iot.BadTokenRequests.Inc()
// 			http.Error(w, "bad client key", 500)
// 			return
// 		}

// 		// not using the payload . we always hand out Tiny4

// 		payload := tokens.KnotFreeTokenPayload{}

// 		payload.Issuer = "_9sh"
// 		payload.JWTID = tokens.GetRandomB36String()
// 		nonce := payload.JWTID
// 		payload.ExpirationTime = uint32(time.Now().Unix()) + 60*60*24*30*2 // two months

// 		priceThing := tokens.GetTokenStatsAndPrice(tokens.TinyX4)
// 		payload.KnotFreeContactStats = priceThing.Stats

// 		parts := strings.Split(req.Host, ".")
// 		partslen := len(parts)
// 		if partslen < 2 {
// 			parts = strings.Split("local.localhost", ".")
// 			partslen = len(parts)
// 		}
// 		targetSite := parts[partslen-2] + "." + parts[partslen-1]

// 		payload.URL = targetSite

// 		exp := payload.ExpirationTime
// 		if exp > uint32(time.Now().Unix()+60*60*24*365) {
// 			// more than a year in the future not allowed now.
// 			exp = uint32(time.Now().Unix() + 60*60*24*365)
// 			fmt.Println("had long token ", string(payload.JWTID)) // TODO: store in db
// 		}

// 		cost := priceThing.Price // tokens.CalcTokenPrice(&payload, uint32(time.Now().Unix()))
// 		fmt.Println("token cost is " + fmt.Sprintf("%f", cost))

// 		// if cost > 0.012 {
// 		// 	http.Error(w, "token too expensive at "+fmt.Sprintf("%f", cost), 500)
// 		// 	return
// 		// }

// 		signingKey := tokens.GetPrivateKey(0)
// 		tokenString, err := tokens.MakeToken(&payload, []byte(signingKey))
// 		if err != nil {
// 			iot.BadTokenRequests.Inc()
// 			http.Error(w, err.Error(), 500)
// 			return
// 		}
// 		signingKey = "unused now"

// 		when := time.Unix(int64(exp), 0)
// 		year, month, day := when.Date()

// 		comments := make([]interface{}, 3)
// 		tmp := fmt.Sprintf(" expires: %v-%v-%v", year, int(month), day)
// 		comments[0] = tokenRequest.Comment + tmp
// 		comments[1] = "" //payload
// 		comments[2] = string(tokenString)
// 		//returnval, err := json.Marshal(comments)
// 		_ = err
// 		//returnval = []byte(strings.ReplaceAll(string(returnval), `"`, ``))
// 		// returnval = []byte(strings.ReplaceAll(string(returnval), ` `, `_`))
// 		//fmt.Println("sending token package ", string(returnval))

// 		returnval := tokenString

// 		err = tokens.LogNewToken(ctx, &payload, remoteAddr)
// 		if err != nil {
// 			iot.BadTokenRequests.Inc()
// 			http.Error(w, err.Error(), 500)
// 			return
// 		}
// 		// box it up
// 		boxout := make([]byte, len(returnval)+box.Overhead)
// 		boxout = boxout[:0]
// 		var jwtid [24]byte
// 		copy(jwtid[:], []byte(nonce))

// 		var clipub [32]byte
// 		n, err := hex.Decode(clipub[:], []byte(clientPublicKey))
// 		_ = err
// 		if n != 32 {
// 			iot.BadTokenRequests.Inc()
// 			http.Error(w, "bad size2", 500)
// 			return
// 		}
// 		sealed := box.Seal(boxout, returnval, &jwtid, &clipub, api.ce.PrivateKeyTemp)

// 		reply := tokens.TokenReply{}
// 		reply.Nonce = nonce
// 		reply.Pkey = hex.EncodeToString(api.ce.PublicKeyTemp[:])
// 		reply.Payload = hex.EncodeToString(sealed)
// 		bytes, err := json.Marshal(reply)
// 		if err != nil {
// 			iot.BadTokenRequests.Inc()
// 			http.Error(w, err.Error(), 500)
// 			return
// 		}
// 		//time.Sleep(8 * time.Second)
// 		time.Sleep(1 * time.Second)
// 		w.Write(bytes)
// 		tokensServed++
// 		fmt.Println("done sending free token")
// 	}
// }

// type webHandler struct { // is this even used?
// 	ce *iot.ClusterExecutive

// 	//	fs1 http.Handler
// 	fs2 http.Handler

// 	//fs := http.FileServer(http.Dir("./docs/_site"))
// 	// supermux.sub.Handle("/", fs)

// }

// // webHandler.ServeHTTP serves the static content
// func (api webHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

// 	fmt.Println("webHandler ServeHTTP", r.RequestURI)

// 	api.fs2.ServeHTTP(w, r)

// }
