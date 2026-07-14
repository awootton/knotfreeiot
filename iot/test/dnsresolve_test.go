package iot_test

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/stretchr/testify/assert"
)

// we have to be awake? ok.
func TestQueryCallOverHttp3native(t *testing.T) {
	url := "http://knotfree.com:8085/api1/dns-query?name=alan-t-wootton.iot,NOTtestmain-0n0u0e16p-0.vr,get-unix-time.iot&type=A&knotfree=1"

	resp, err := http.Get(url)
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("Error fetching URL: %v", err)
		return
	}
	defer resp.Body.Close()

	// a buffer
	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	assert.NoError(t, err)
	result := string(buf[:n])
	// log.Printf("Response body: %s\n", result)
	assert.Equal(t, 200, resp.StatusCode)
	// assert.Equal(t, "-muxcABH_pTsuNqT3yaYfQj-3krwM6XmEu47vTZLSHM", result)
	{
		var responses []iot.DnsResponse
		err = json.Unmarshal([]byte(result), &responses)
		assert.NoError(t, err)
		if err != nil {
			t.Errorf("Error unmarshalling JSON: %v", err)
			return
		}
		assert.Equal(t, 3, len(responses))
		assert.Equal(t, 0, responses[0].Status)
		assert.Equal(t, "216.128.128.195", responses[0].Answer[0].Data)

		assert.Equal(t, 3, responses[2].Status) // not found
		// answer array is empty
	}
}

// start a server for this one
func TestQueryCallOverHttp3(t *testing.T) {
	url := "http://knotfree.com:8085/api1/dns-query?name=example.com,gotohere.com,xgtohere.com,testmain-0n0u0e12p-0.xyz&type=A&dnsserver=1.1.1.1"

	resp, err := http.Get(url)
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("Error fetching URL: %v", err)
		return
	}
	defer resp.Body.Close()

	// a buffer
	buf := make([]byte, 1024*64)
	n, err := resp.Body.Read(buf)
	result := string(buf[:n])
	log.Printf("Response body: %s\n", result)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	// assert.Equal(t, "-muxcABH_pTsuNqT3yaYfQj-3krwM6XmEu47vTZLSHM", result)
	{
		var responses []iot.DnsResponse
		err = json.Unmarshal([]byte(result), &responses)
		assert.NoError(t, err)
		if err != nil {
			log.Println(err)
			t.Errorf("Error unmarshalling JSON: %v", err)
			return
		}
		assert.Equal(t, 4, len(responses))
		assert.Equal(t, 0, responses[0].Status)

		assert.Equal(t, 3, responses[2].Status) // not found
		assert.Equal(t, 0, responses[4].Status) // testmain-0n0u0e12p-0.xyz found
		// answer array is empty
	}
}

// do the native endpoint.
func TestLookupDnsOverHttpNative(t *testing.T) {
	ce := makeClusterWithServiceContact()

	recordType := 1 // A record
	response, err := iot.LookupDnsOverHttpKnotfreeOnce(ce, "testmain-0n0u0e16p-0.vr", recordType)
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("Error looking up DNS over HTTP: %v", err)
		return
	}

	assert.Equal(t, 0, response.Status)
	assert.Equal(t, recordType, response.Answer[0].Type)
	assert.Equal(t, "216.128.128.195", response.Answer[0].Data)
}

func TestLookupDnsOverHttpNativeNotFound(t *testing.T) {
	ce := makeClusterWithServiceContact()

	recordType := 1 // A record
	response, err := iot.LookupDnsOverHttpKnotfreeOnce(ce, "999-testmain-0n0u0e16p-0.vr", recordType)
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("Error looking up DNS over HTTP: %v", err)
		return
	}

	log.Printf("Response: %+v\n", response.Answer[0].Data)
	log.Printf("Response: %+v\n", response.Comment)

	assert.Equal(t, 3, response.Status) // NXDOMAIN not found
	assert.Equal(t, recordType, response.Answer[0].Type)
	assert.Equal(t, "status: topic not found", response.Answer[0].Data)
}

// now let's do the endpoint.
// the server must be running.
// knotfree.com is in hosts as localhost.
// don't use others, it will screw up.

func TestQueryCallOverHttp(t *testing.T) {
	url := "http://knotfree.com:8085/api1/dns-query?name=testmain-0n0u0e12p-0.xyz&type=A&dnsserver=1.1.1.1"

	resp, err := http.Get(url)
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("Error fetching URL: %v", err)
		return
	}
	defer resp.Body.Close()

	// a buffer
	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	assert.NoError(t, err)
	result := string(buf[:n])
	// log.Printf("Response body: %s\n", result)
	assert.Equal(t, 200, resp.StatusCode)
	// assert.Equal(t, "-muxcABH_pTsuNqT3yaYfQj-3krwM6XmEu47vTZLSHM", result)
	if strings.HasPrefix(result, "[") {
		// just one response this time.
		t.Errorf("Unexpected result: %s", result)
	} else {
		// this is a one reply test.
		// marshal the result into a DnsResponse struct and check the fields.
		println("Result: " + result)
		var response iot.DnsResponse
		err = json.Unmarshal([]byte(result), &response)
		assert.NoError(t, err)
		if err != nil {
			t.Errorf("Error unmarshalling JSON: %v", err)
			return
		}
		assert.Equal(t, 0, response.Status)
		assert.Equal(t, 1, response.Answer[0].Type)
		// the fourth address of testmain-0n0u0e12p-0.xyz
		assert.Equal(t, "104.20.23.154", response.Answer[3].Data)
	}
}

// sanity check that works.
func TestAnyCallOverHttp(t *testing.T) {

	url := "http://knotfree.com:8085/api1/getPublicKey"

	resp, err := http.Get(url)
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("Error fetching URL: %v", err)
		return
	}
	defer resp.Body.Close()

	// a buffer
	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	assert.NoError(t, err)
	result := string(buf[:n])
	// log.Printf("Response body: %s\n", result)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "-muxcABH_pTsuNqT3yaYfQj-3krwM6XmEu47vTZLSHM", result)
}

// bypass the server and do the lookup directly to test the lookup function.
func TestLookupTextDnsOverHttp(t *testing.T) {
	// domains := []string{"example.com", "gotohere.com", "dummy.gotohere.com", "gotohereX.com"}
	domains := []string{"example.com"}
	recordType := 16 // TXT record
	dnsServer := "1.1.1.1"

	responses, err := iot.LookupDnsOverHttp(domains, recordType, dnsServer)
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("Error looking up DNS over HTTP: %v", err)
		return
	}

	assert.Equal(t, 0, responses[0].Status)
	assert.Equal(t, recordType, responses[0].Answer[0].Type)
	assert.Equal(t, "_k2n1y4vw3qtb4skdx9e7dxt97qrmmq9", responses[0].Answer[1].Data)
}

// TestLookupADnsOverHttpNotFound is a most important case.
func TestLookupADnsOverHttpNotFound(t *testing.T) {
	// domains := []string{"example.com", "gotohere.com", "dummy.gotohere.com", "gotohereX.com"}
	domains := []string{"exampleNOT-FOUND.com"}
	recordType := 1 // A record
	dnsServer := "1.1.1.1"

	responses, err := iot.LookupDnsOverHttp(domains, recordType, dnsServer)
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("Error looking up DNS over HTTP: %v", err)
		return
	}

	assert.Equal(t, 3, responses[0].Status) // not found. This is the most important assertion in this test, because it verifies that we correctly distinguish between a domain not existing (NXDOMAIN) and an actual error (SERVFAIL). If this assertion fails, it means that we are treating all errors as SERVFAIL, which is not correct. We need to be able to distinguish between these two cases, because they have different implications for how we handle the response. For example, if we get a SERVFAIL, we might want to retry the query with a different resolver or return an error to the user. But if we get an NXDOMAIN, we know that the domain simply doesn't exist, and we can return that information to the user without retrying.
	assert.Equal(t, 0, len(responses[0].Answer))
}

func TestLookupDnsOverHttp(t *testing.T) {
	domains := []string{"example.com", "gotoNOThere.com", "dummy.gotohere.com", "gotohere.com", "testmain-0n0u0e12p-0.xyz"}
	recordType := 1 // A record
	dnsServer := "1.1.1.1"

	responses, err := iot.LookupDnsOverHttp(domains, recordType, dnsServer)
	assert.NoError(t, err)
	if err != nil {
		t.Errorf("Error looking up DNS over HTTP: %v", err)
		return
	}

	assert.Equal(t, 0, responses[0].Status)
	assert.Equal(t, recordType, responses[0].Answer[0].Type)
	// this isn't going to hold still - assert.Equal(t, "104.20.23.154", responses[0].Answer[3].Data)

	assert.Equal(t, 3, responses[1].Status) // not found
	assert.Equal(t, 0, len(responses[1].Answer))

	assert.Equal(t, 3, responses[2].Status) // subdomain not found.
	// dummy.gotohere.com , a subdomain.
	// should be found but hase no A record, so it should return an empty answer with status 0, not an error. This is a common case for domains that exist but don't have the requested record type, and we want to handle it gracefully without treating it as an error.
	// it'll happen in the parent nodes of The Metaverse.
	// it won't let me make the subdomain without at least a 0.0.0.0 address.
	// assert.Equal(t, "0.0.0.0", responses[2].Answer[0].Data)

	assert.Equal(t, 0, responses[3].Status)
	// https endpoint of gotohere.com
	assert.Equal(t, "216.128.128.128", responses[3].Answer[0].Data)

	assert.Equal(t, 0, responses[4].Status) // testmain-0n0u0e12p-0.xyz IS found

}

// dude, that was way too easy.

func TestDNSResolve(t *testing.T) {
	// Initialize a resolver tied to a specific upstream address
	resolver := &net.Resolver{
		PreferGo: true, // Forces the native Go resolver instead of cgo
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 3 * time.Second,
			}
			// Route all UDP/TCP queries to Cloudflare's public DNS server
			return d.DialContext(ctx, network, "1.1.1.1:53")
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := resolver.LookupHost(ctx, "example.com")
	if err != nil {
		print("Custom lookup failed: %v", err)
		return
	}
	println("Resolved example hosts: " + strings.Join(ips, ", "))

	ips, err = resolver.LookupTXT(ctx, "example.com")
	if err != nil {
		print("Custom lookup failed: %v", err)
		return
	}
	println("Resolved example txt: " + strings.Join(ips, ", "))

	// Query an A/AAAA record using the custom resolver
	ips, err = resolver.LookupHost(ctx, "gotohere.com")
	if err != nil {
		print("Custom lookup failed: %v", err)
		return
	}

	println("Resolved gotohere hosts: " + strings.Join(ips, ", "))

	ips, err = resolver.LookupTXT(ctx, "gotohere.com")
	if err != nil {
		print("Custom lookup failed: %v", err)
		return
	}

	println("Resolved txt: " + strings.Join(ips, ", "))

	ips, err = resolver.LookupTXT(ctx, "dummy.gotohere.com")
	if err != nil {
		print("Custom lookup failed: %v", err)
		return
	}

	println("Resolved dummy txt: " + strings.Join(ips, ", "))

	ips, err = resolver.LookupHost(ctx, "gotohereX.com")
	if err != nil {
		print("Custom lookup failed: %v", err)
		return
	}

	println("Resolved host: " + strings.Join(ips, ", "))

}

func TestDNSResolveGotohere(t *testing.T) {
	// Initialize a resolver tied to a specific upstream address
	resolver := &net.Resolver{
		PreferGo: true, // Forces the native Go resolver instead of cgo
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 3 * time.Second,
			}
			// Route all UDP/TCP queries to Cloudflare's public DNS server
			return d.DialContext(ctx, network, "dns.gotohere.com"+":53")
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query an A/AAAA record using the custom resolver
	ips, err := resolver.LookupHost(ctx, "example.com")
	if err != nil {
		print("Custom lookup failed: %v", err)
		return
	}

	println("Resolved IPs using dns.gotohere.com: " + strings.Join(ips, ", "))

	ips, err = resolver.LookupTXT(ctx, "example.com")
	if err != nil {
		print("Custom lookup failed: %v", err)
		return
	}

	println("Resolved txt using dns.gotohere.com: " + strings.Join(ips, ", "))

	// and the weird one. Only in the gotohere dns server,
	ips, err = resolver.LookupTXT(ctx, "meta_group_id.testmain-0n0u0e16p-0.vr")
	if err != nil {
		print("Custom lookup failed: %v", err)
		return
	}

	// expect meta_group_id-no-leading-underscore

	println("Resolved meta_group_id.testmain-0n0u0e16p-0.vr txt using dns.gotohere.com: " + strings.Join(ips, ", "))

}

// I want to copy this format for the url and the results.
// results from node typescript
//   let url = `https://one.one.one.one/dns-query?name=${domain}&type=TXT`
//     {
//   "Status": 0,
//   "TC": false,  TrunCation
//   "RD": true,   Recursion Desired
//   "RA": true,   Recursion Available
//   "AD": false,  Authenticated Data
//   "CD": false,  Checking Disabled
//   "Question": [
//     {
//       "name": "gotohere.com",
//       "type": 16
//     }
//   ],
//   "Answer": [
//     {
//       "name": "gotohere.com",
//       "type": 16,
//       "TTL": 300,
//       "data": "\"default-txt-at-vultr\""
//     }
//   ]
// }
// or, note multiple answers for multiple TXT records:
//     {
//   "Status": 0,
//   "TC": false,
//   "RD": true,
//   "RA": true,
//   "AD": true,
//   "CD": false,
//   "Question": [
//     {
//       "name": "example.com",
//       "type": 16
//     }
//   ],
//   "Answer": [
//     {
//       "name": "example.com",
//       "type": 16,
//       "TTL": 300,
//       "data": "\"v=spf1 -all\""
//     },
//     {
//       "name": "example.com",
//       "type": 16,
//       "TTL": 300,
//       "data": "\"_k2n1y4vw3qtb4skdx9e7dxt97qrmmq9\""
//     }
//   ]
// }

//   url = `https://one.one.one.one/dns-query?name=${domain}&type=A`
// here's one with the multiple answers
// {
//   "Status": 0,
//   "TC": false,
//   "RD": true,
//   "RA": true,
//   "AD": false,
//   "CD": false,
//   "Question": [
//     {
//       "name": "example.com",
//       "type": 1
//     }
//   ],
//   "Answer": [
//     {
//       "name": "example.com",
//       "type": 1,
//       "TTL": 105,
//       "data": "172.66.147.243"
//     },
//     {
//       "name": "example.com",
//       "type": 1,
//       "TTL": 105,
//       "data": "104.20.23.154"
//     }
//   ]
// }

// the version from google it has a comment field and the AD field is true, but otherwise similar to the above.
// Fetchingrecords for example.com from   DNS over HTTPS... using url https://dns.google/resolve?name=example.com&type=TXT
//   Record json is: {
//   "Status": 0,
//   "TC": false,
//   "RD": true,
//   "RA": true,
//   "AD": true,
//   "CD": false,
//   "Question": [
//     {
//       "name": "example.com.",
//       "type": 16
//     }
//   ],
//   "Answer": [
//     {
//       "name": "example.com.",
//       "type": 16,
//       "TTL": 300,
//       "data": "v=spf1 -all"
//     },
//     {
//       "name": "example.com.",
//       "type": 16,
//       "TTL": 300,
//       "data": "_k2n1y4vw3qtb4skdx9e7dxt97qrmmq9"
//     }
//   ],
//   "Comment": "Response from 172.64.32.162."
// }

// google A record response
//     Fetchingrecords for example.com from   DNS over HTTPS... using url https://dns.google/resolve?name=example.com&type=A
//   Record A json is: {
//   "Status": 0,
//   "TC": false,
//   "RD": true,
//   "RA": true,
//   "AD": true,
//   "CD": false,
//   "Question": [
//     {
//       "name": "example.com.",
//       "type": 1
//     }
//   ],
//   "Answer": [
//     {
//       "name": "example.com.",
//       "type": 1,
//       "TTL": 300,
//       "data": "172.66.147.243"
//     },
//     {
//       "name": "example.com.",
//       "type": 1,
//       "TTL": 300,
//       "data": "104.20.23.154"
//     }
//   ],
//   "Comment": "Response from 173.245.58.162."
// }

// this is a successful fail to find.
// it has no answers.
// Fetchingrecords for exampleXX.com from   DNS over HTTPS... using url https://dns.google/resolve?name=exampleXX.com&type=A
//   Record A json is: {
//   "Status": 3,
//   "TC": false,
//   "RD": true,
//   "RA": true,
//   "AD": false,
//   "CD": false,
//   "Question": [
//     {
//       "name": "exampleXX.com.",
//       "type": 1
//     }
//   ],
//   "Authority": [
//     {
//       "name": "com.",
//       "type": 6,
//       "TTL": 900,
//       "data": "a.gtld-servers.net. nstld.verisign-grs.com. 1780675651 1800 900 604800 900"
//     }
//   ],
//   "Comment": "Response from 192.54.112.30."
// }

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
