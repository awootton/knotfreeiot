package iot

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// MakeTCPExecutive is a thing like a server, not the exec
func MakeTCPExecutive(ex *Executive, serverName string) *Executive {

	go listenForPacketsConnect(ex, serverName)

	return ex
}

type apiHandler struct { // lose this?
	ex *Executive
}

func (api apiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//log.Println("req.RequestURI", req.RequestURI)
	switch req.RequestURI {
	case "/api2/getstats": // GET
		// return the stats for just me.
		// log.Println("GetStats /api2/getstats", api.ex.Name, api.ex.httpAddress)
		stats := api.ex.GetExecutiveStats()
		stats.Limits = api.ex.Limits
		bytes, err := json.Marshal(stats)
		if err != nil {
			log.Println("GetExecutiveStats marshal", err)
		}
		w.Write(bytes)

		API1GetStats.Inc()

	case "/api2/set": // POST
		decoder := json.NewDecoder(req.Body)
		args := &UpstreamNamesArg{}
		err := decoder.Decode(args)
		if err != nil {
			http.Error(w, "decode error", 500)
			API1PostGurusFail.Inc()
			return
		}

		API1PostGurus.Inc()
		if len(args.Names) > 0 && len(args.Names) == len(args.Addresses) {
			// log.Println("SetUpstreamNames ", args.Names, args.Addresses, api.ex.Name, api.ex.tcpAddress)
			api.ex.Looker.SetUpstreamNames(args.Names, args.Addresses)
		} else {
			log.Println("SetUpstreamNames bad names sent", args.Names, args.Addresses, args)
		}
		//log.Println("/api2/set done")

	case "/api2/clusterstats": // POST

		// todo: add security. no - just keep port 8080 unavailable to the world

		data, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "read error 2", 500)
			API1PostGurusFail.Inc()
			return
		}
		stats := &ClusterStats{}
		err = json.Unmarshal(data, stats)
		if err != nil {
			http.Error(w, "decode error 2", 500)
			API1PostGurusFail.Inc()
			return
		}
		str := ""
		for _, stat := range stats.Stats {
			str += stat.Name + " " + stat.TCPAddress + "  "
		}
		api.ex.statsmu.Lock()
		api.ex.ClusterStats = stats
		api.ex.ClusterStatsString = string(data)
		api.ex.statsmu.Unlock()

		// log.Println("api2/clusterstats", str, api.ex.Name)

	default:
		http.NotFound(w, req)
		IotHTTP404.Inc()
	}
}

// MakeHTTPExecutive sets up a http server for serving api1 and api2
func MakeHTTPExecutive(ex *Executive, serverName string) *Executive {

	mux := http.NewServeMux()
	mux.Handle("/api1/", apiHandler{ex})
	mux.Handle("/api2/", apiHandler{ex})

	s := &http.Server{
		Addr:           serverName,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func(s *http.Server) {
		log.Println("MakeHTTPExecutive http service, name, port ", ex.Name, s.Addr)
		err := s.ListenAndServe()
		if err != nil {
			log.Println("ListenAndServe ERROR", err)
		}
		log.Println("ListenAndServe returned !!!!!  arrrrg", err)
	}(s)
	return ex
}

// GetServerStats asks nicely over http
func GetServerStats(addr string) (*ExecutiveStats, error) {

	stats := &ExecutiveStats{}

	if len(addr) < 4 {
		return stats, errors.New("missing stats address")
	}
	if strings.HasPrefix(addr, ":") {
		return stats, errors.New("only port")
	}

	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/api2/getstats")

	if err == nil && resp.StatusCode == 200 {

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return stats, err
		}
		err = json.Unmarshal(bytes, &stats)
		if err != nil {
			return stats, err
		}
	} else {
		log.Println("GetServerStats failed ", addr, err)
	}
	return stats, err
}

// UpstreamNamesArg just has the one job
type UpstreamNamesArg struct {
	Names     []string
	Addresses []string
}

// PostUpstreamNames does SetUpstreamNames the hard way
// we are not going over the internet. Inside a ns should ba well under 1000 ms.
func PostUpstreamNames(guruList []string, addressList []string, addr string) error {

	arg := &UpstreamNamesArg{}
	arg.Names = guruList
	arg.Addresses = addressList

	if len(guruList) != len(addressList) {
		return errors.New("PostUpstreamNames len(guruList) != len(addressList)")
	}

	jbytes, err := json.Marshal(arg)
	if err != nil {
		log.Println("unreachable ?? bb")
		return errors.New("upstreamNamesArg marshal fail")
	}

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Post("http://"+addr+"/api2/set", "application/json", bytes.NewReader(jbytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("upstreamNamesArg not 200")
	}
	return nil
}

// PostClusterStats sends some stats to
func PostClusterStats(ex *Executive, stats *ClusterStats, addr string) error {

	log.Println("PostClusterStats sending to ", addr, "from", ex.Name)

	jbytes, err := json.Marshal(stats)
	if err != nil {
		log.Println("unreachable ? PostClusterStats marshal fail")
		return errors.New("PostClusterStats marshal fail")
	}

	addstr := "http://" + addr + "/api2/clusterstats"
	log.Println("PostClusterStats sending to ", addstr, "from", ex.Name)
	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Post(addstr, "application/json", bytes.NewReader(jbytes))
	if err != nil {
		log.Println("PostClusterStats err", err, addstr, "from", ex.Name)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("PostClusterStats not 200", resp.StatusCode, addstr, "from", ex.Name)
		return errors.New("PostClusterStats not 200")
	}
	return nil
}

// Copyright 2019,2020,2021,2026 Alan Tracey Wootton
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
