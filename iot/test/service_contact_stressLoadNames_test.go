package iot_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/awootton/knotfreeiot/iot"
	"github.com/awootton/knotfreeiot/packets"
	"github.com/stretchr/testify/assert"
)

// success ratio is  1320  vs  0  =  1
// TestDialTCP_by_batches_bitches took  5.436035112s  for  1320  batches, average time  4.118208ms
// PASS

// to be fair, there's a cache.
// success ratio is  13200  vs  0  =  1
// TestDialTCP_by_batches_bitches took  5.7428781s  for  13200  batches, average time  435.066µs
// PASS
// so, ramp it up to 100 passes of 13200 batches each.

// success ratio is  132000  vs  0  =  1
// TestDialTCP_by_batches_bitches took  9.077715715s  for  132000  batches, average time  68.77µs
// PASS

// I just want to do this all day long while I touch myself.
// success ratio is  1320000  vs  0  =  1
// TestDialTCP_by_batches_bitches took  41.318963084s  for  1320000  batches, average time  31.302µs
// PASS

// one more thing, I'm turning off the checks. NOPE: failed. TODO: (atw) fix that. See sendChecker.go
// TODO: (atw) turn off the checks in sendChecker.go and make the tests pass.
// 6/27/26 still can't turn off turnOffTheseCheckers in sendChecker.go

func TestDialTCP_by_batches(t *testing.T) {

	ce := makeClusterWithServiceContact()
	_ = ce

	fmt.Println("TestDialTCP_by_batches makeClusterWithServiceContact done. We should have the guru dialed already.")

	sleepTime := 15 * time.Second
	time.Sleep(sleepTime) // except now it worked. wtf. Keep going. I can't live like this.

	// these were the names as presented to TwoWayLookupAndMerge (see the metaverse-proto-one project).
	// typically they are fetches as .vr from here and then .xyz from cloudflare
	// trying to just load them from here will probably crash something, or disconnect the guru or something.

	// fmt.Println("TestDialTCP", names)
	startTime := time.Now()
	okCount := 0
	failCount := 0
	// needs passes, lol.
	for pass := 0; pass < 1; pass++ {
		if pass%10 == 0 {
			fmt.Println("TestDialTCP_one_at_a_time pass", pass)
		}
		for _, nameList := range names {
			// fmt.Println("TestDialTCP", nameList)
			// through the front door.

			// with .vr and commas.
			// join them with commas and send them all at once.

			theNames := strings.Join(nameList, ".vr,")
			// fmt.Println("TestDialTCP_batches_bitches sending names:", theNames)

			url := "http://knotfree.com:8085/api1/dns-query?name=" + theNames + "&type=A&knotfree=1"

			// fmt.Println("TestDialTCP_batches_bitches url:", url)

			resp, err := http.Get(url)
			assert.NoError(t, err)
			if err != nil {
				t.Errorf("Error fetching URL: %v", err)
				failCount += len(nameList)
				continue
			}
			defer resp.Body.Close()

			// a buffer
			buf := make([]byte, 64*1024) // 64KB buffer, It's huge but I don't care. It's a test. I want to see the whale response.
			n, err := resp.Body.Read(buf)
			if err != nil && err.Error() != "EOF" { // is EOF normal?
				failCount += len(nameList)
				continue
			}
			// what about this? Technically w're getting an error: assert.NoError(t, err)
			result := string(buf[:n])
			// fmt.Printf("Response body: %s\n", result)
			assert.Equal(t, 200, resp.StatusCode)

			// fmt.Printf("Response body: %s\n", result)
			// eg.
			// why are we getting 2's?
			// [{"Status":2,"TC":false,"RD":true,"RA":false,"AD":false,"CD":true,"Question":[{"name":"testmain-0n0u0e16p-0.vr","type":1}],"Comment":"Server failure"},{"Status":2,"TC":false,"RD":true,"RA":false,"AD":false,"CD":true,"Question":[{"name":"
			{
				var responses []iot.DnsResponse
				err = json.Unmarshal([]byte(result), &responses)
				assert.NoError(t, err)
				if err != nil {
					t.Errorf("Error unmarshalling JSON: %v", err)
					failCount += len(nameList)
					continue
				}
				okCount += len(nameList)
				// this is the main one.
				assert.Equal(t, len(nameList), len(responses))
				for i, response := range responses {
					_ = i
					// let's print the answers
					answer := response.Answer[0].Data
					_ = answer
					if response.Status != 0 && response.Status != 3 {
						fmt.Println("TestDialTCP_batches_bitches expecting 0 or 3 for", nameList[i], "is", response.Status, "answer is", answer)
					}
					// let's not: fmt.Println("TestDialTCP_batches_bitches response for", nameList[i], "is", answer)
				}
			}
		}
	}
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	fmt.Println("success ratio is ", okCount, " vs ", failCount, " = ", float64(okCount)/float64(okCount+failCount))
	fmt.Println("TestDialTCP_by_batches_bitches took ", duration, " for ", okCount+failCount, " batches, average time ", duration/time.Duration(okCount+failCount))
}

// PASS with read-till-eof loop below.  TestDialTCP_by_batches PROD response body length: 1198 usually around 4000
//     ServiceContact timed out in GetPacketReplyLonger

// LookupDnsOverHttpKnotfree starting, looking up  16  domains with record type  1
// LookupDnsOverHttpKnotfree finished looking up  16  domains with record type  1
// LookupDnsOverHttpKnotfree starting, looking up  16  domains with record type  1

// ServiceContact timed out waiting for reply. Starting new ServiceContact. # 18
// LookupDnsOverHttpKnotfree to get from service contact ServiceContact timed out waiting for reply in GetPacketReplyLonger # 18 after 5s
// Error looking up DNS over HTTP for domain testmain-0n0u1w14p-6.vr : ServiceContact timed out waiting for reply in GetPacketReplyLonger # 18 after 5s
// LookupDnsOverHttpKnotfree finished looking up  16  domains with record type  1
// SendPacket, Receive-a-packet loop, timed out # 18
// it's like it's not running the same version as this.

// check if we are using https or not below.
// this is passing in the non-https version. Time to get rid of nginx.

func TestDialTCP_by_batches_PROD(t *testing.T) {

	// ce := makeClusterWithServiceContact()
	// _ = ce

	// sleepTime := 10 * time.Second
	// time.Sleep(sleepTime) // except now it worked. wtf. Keep going. I can't live like this.

	// these were the names as presented to TwoWayLookupAndMerge (see the metaverse-proto-one project).
	// typically they are fetches as .vr from here and then .xyz from cloudflare
	// trying to just load them from here will probably crash something, or disconnect the guru or something.

	// fmt.Println("TestDialTCP", names)
	startTime := time.Now()
	okCount := 0
	failCount := 0
	// needs passes, lol.
	for pass := 0; pass < 1; pass++ {
		if pass%10 == 0 {
			fmt.Println("TestDialTCP_by_batches PROD pass", pass)
		}
		for _, nameList := range names {
			// fmt.Println("TestDialTCP", nameList)
			// through the front door.

			// with .vr and commas.
			// join them with commas and send them all at once.

			theNames := strings.Join(nameList, ".vr,")
			// fmt.Println("TestDialTCP_by_batches PROD sending names:", theNames)

			url := "https://knotfree.net/api1/dns-query?name=" + theNames + "&type=A&knotfree=1"
			// url := "http://knotfree.io/api1/dns-query?name=" + theNames + "&type=A&knotfree=1"

			// fmt.Println("TestDialTCP_by_batches PROD url:", url)

			resp, err := http.Get(url)
			assert.NoError(t, err)
			if err != nil {
				t.Errorf("Error fetching URL: %v", err)
				failCount += len(nameList)
				continue
			}
			defer resp.Body.Close()

			// a buffer
			buf := make([]byte, 64*1024) // 64KB buffer, It's huge but I don't care. It's a test. I want to see the whale response.
			n, err := 0, error(nil)
			totalRead := 0
			for {
				nn, err := resp.Body.Read(buf[totalRead:])
				totalRead += nn
				if err != nil {
					if err.Error() == "EOF" {
						break
					} else {
						failCount += len(nameList)
						continue
					}
				}
			}
			fmt.Printf("TestDialTCP_by_batches PROD response body length: %d\n", totalRead)
			n = totalRead

			// what about this? Technically w're getting an error: assert.NoError(t, err)
			result := string(buf[:n])
			// fmt.Printf("Response body: %s\n", result)
			assert.Equal(t, 200, resp.StatusCode)

			// fmt.Printf("Response body: %s\n", result)
			// eg.
			// why are we getting 2's?
			// [{"Status":2,"TC":false,"RD":true,"RA":false,"AD":false,"CD":true,"Question":[{"name":"testmain-0n0u0e16p-0.vr","type":1}],"Comment":"Server failure"},{"Status":2,"TC":false,"RD":true,"RA":false,"AD":false,"CD":true,"Question":[{"name":"
			{
				var responses []iot.DnsResponse
				err = json.Unmarshal([]byte(result), &responses)
				assert.NoError(t, err)
				if err != nil {
					fmt.Println("TestDialTCP_by_batches PROD error unmarshalling JSON: ", result, err)
					t.Errorf("Error unmarshalling JSON: %v", err)
					failCount += len(nameList)
					continue
				}
				okCount += len(nameList)
				// this is the main one.
				if len(nameList) != len(responses) {
					fmt.Println("TestDialTCP_by_batches PROD mismatch: expected", len(nameList), "responses, got", len(responses))
				}
				assert.Equal(t, len(nameList), len(responses))
				for i, response := range responses {
					_ = i
					// let's print the answers
					if len(response.Answer) > 0 {
						answer := response.Answer[0].Data
						_ = answer
						if response.Status != 0 && response.Status != 3 {
							fmt.Println("TestDialTCP_by_batches PROD expecting 0 or 3 for", nameList[i], "is", response.Status, "answer is", answer)
						}
						// let's not: fmt.Println("TestDialTCP_by_batches PROD response for", nameList[i], "is", answer)
					}
				}
			}
		}
	}
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	fmt.Println("success ratio is ", okCount, " vs ", failCount, " = ", float64(okCount)/float64(okCount+failCount))
	fmt.Println("TestDialTCP_by_batches PROD took ", duration, " for ", okCount+failCount, " batches, average time ", duration/time.Duration(okCount+failCount))
}

//
// success ratio is  1320  vs  0  =  1
// TestDialTCP_by_batches_bitches took  45.279134715s  for  1320  batches, average time  34.302374ms//
//
// It's slow but that's what we expected.
// TestDialTCP_one_at_a_time pass 1
// TestDialTCP_one_at_a_time pass 2
// TestDialTCP_one_at_a_time pass 3
// success ratio is  5280  vs  0  =  1
// TestDialTCP_by_batches_bitches took  55.92777849s  for  5280  batches, average time  10.592382ms
// PASS

func TestDialTCP_one_at_a_time(t *testing.T) {

	ce := makeClusterWithServiceContact()

	sleepTime := 10 * time.Second
	time.Sleep(sleepTime) // except now it worked. wtf. Keep going. I can't live like this.

	// these were the names as presented to TwoWayLookupAndMerge (see the metaverse-proto-one project).
	// typically they are fetches as .vr from here and then .xyz from cloudflare
	// trying to just load them from here will probably crash something, or disconnect the guru or something.

	// fmt.Println("TestDialTCP", names)
	startTime := time.Now()
	okCount := 0
	failCount := 0
	for pass := 0; pass < 4; pass++ {
		fmt.Println("TestDialTCP_one_at_a_time pass", pass)
		for _, nameList := range names {
			// fmt.Println("TestDialTCP", nameList)
			for _, name := range nameList {

				// fmt.Println("TestDialTCP", name)

				recordType := 1 // A record
				response, err := iot.LookupDnsOverHttpKnotfreeOnce(ce, name+".vr", recordType)
				assert.NoError(t, err)
				if err != nil {
					t.Errorf("Error looking up DNS over HTTP: %v", err)
					failCount++
					continue
				}

				// sleepTime := 10 * time.Second
				// _ = sleepTime
				//time.Sleep(sleepTime) // just to be silly. It's not supposed to matter AT ALL.

				// we don't know what they're all going to be.
				ok := false
				if response.Status == 0 || response.Status == 3 {
					ok = true
				}
				assert.Equal(t, true, ok)
				assert.Equal(t, recordType, response.Answer[0].Type)
				okCount++
				// let's just print these:
				// boring fmt.Println("address is ", response.Answer[0].Data)
			}
		}
	}
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	fmt.Println("success ratio is ", okCount, " vs ", failCount, " = ", float64(okCount)/float64(okCount+failCount))
	fmt.Println("TestDialTCP_by_batches_bitches took ", duration, " for ", okCount+failCount, " batches, average time ", duration/time.Duration(okCount+failCount))

}

func TestDialTCP_1000_FAILED_get_option_A(t *testing.T) {

	ce := makeClusterWithServiceContact()

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()

	// wait for 5 sec, just to be silly. It's not supposed to matter AT ALL.
	sleepTime := 5 * time.Second
	time.Sleep(sleepTime) // except now it worked. wtf. Keep going. I can't live like this.

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()

	passes := 100 // 1024 * 64
	startTime := time.Now()
	for i := 0; i < passes; i++ {

		// sc := ce.GetPacketService() // what name do we know that already exists?
		// itsSubscrition := sc.

		// Let's do "get help", you always love that one.
		name := "get-unix-time-that-does-not-exist"
		command := "get option A @ "
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))

		// send it, long timeout so we can debug.
		reply, err := ce.GetPacketService().GetPacketReplyLonger(&cmd, 999*time.Second)
		if err != nil {
			t.Fatal("GetPacketReply failed", err)
		}
		got := string(reply.(*packets.Send).Payload)
		iot.CheckSendPacket(reply.(*packets.Send))
		ok := strings.Contains(got, "status: topic not found errid=bvBbhJawYXIMWsxJOWHt")
		if !ok {
			t.Fatal("GetPacketReply did not return expected   message")
		}
		// bulky fmt.Println("TestDialTCP_one_at_a_time got  help reply", got)
	}
	endTime := time.Now()
	elapsed := endTime.Sub(startTime)
	fmt.Printf("TestDialTCP_1000_helps completed %d passes in %s, average time %s\n", passes, elapsed, elapsed/time.Duration(passes))
}

func TestDialTCP_1000_get_option_A(t *testing.T) {

	ce := makeClusterWithServiceContact()

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()

	// wait for 5 sec, just to be silly. It's not supposed to matter AT ALL.
	sleepTime := 5 * time.Second
	time.Sleep(sleepTime) // except now it worked. wtf. Keep going. I can't live like this.

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()

	passes := 100 // 1024 * 64
	startTime := time.Now()
	for i := 0; i < passes; i++ {

		// sc := ce.GetPacketService() // what name do we know that already exists?
		// itsSubscrition := sc.

		// Let's do "get help", you always love that one.
		name := "get-unix-time"
		command := "get option A @ "
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))

		// send it, long timeout so we can debug.
		reply, err := ce.GetPacketService().GetPacketReplyLonger(&cmd, 999*time.Second)
		if err != nil {
			t.Fatal("GetPacketReply failed", err)
		}
		got := string(reply.(*packets.Send).Payload)
		iot.CheckSendPacket(reply.(*packets.Send))
		ok := strings.Contains(got, "216.128.128.195")
		if !ok {
			t.Fatal("GetPacketReply did not return expected   message")
		}
		// bulky fmt.Println("TestDialTCP_one_at_a_time got   help reply", got)
	}
	endTime := time.Now()
	elapsed := endTime.Sub(startTime)
	fmt.Printf("TestDialTCP_1000_helps completed %d passes in %s, average time %s\n", passes, elapsed, elapsed/time.Duration(passes))
}

// see TestLookupDnsOverHttpNativeNotFound !!
// and TestQueryCallOverHttp

// _, err := LookupDnsOverHttpKnotfree(globalClusterExec, []string{name}, 1)

// THese seem gone now: fails with 1000 !!!! GetPacketReply failed ServiceContact timed out waiting for reply in GetPacketReplyLonger # 48 after 17s

// it's fast:
// TestDialTCP_1000_helps completed 2 passes in 1.118749ms, average time 559.374µs
// TestDialTCP_1000_helps completed 4 passes in 3.642724ms, average time 910.681µs
// TestDialTCP_1000_helps completed 8 passes in 6.259661ms, average time 782.457µs
// completed 32 passes in 10.499646ms, average time 328.113µs
// completed 64 passes in 23.110705ms, average time 361.104µs
// completed 128 passes in 38.18236ms, average time 298.299µs
// completed 128 passes in 40.143158ms, average time 313.618µs
// all the checkers made it slower.
// TestDialTCP_1000_helps completed 64 passes in 152.958792ms, average time 2.389981ms
// TestDialTCP_1000_helps completed 128 passes in 244.436234ms, average time 1.909658ms
// TestDialTCP_1000_helps completed 512 passes in 683.525299ms, average time 1.33501ms
// TestDialTCP_1000_helps completed 1024 passes in 1.201681323s, average time 1.173516ms
// TestDialTCP_1000_helps completed 4096 passes in 4.561602122s, average time 1.113672ms   wow
// and here we have it:
// TestDialTCP_1000_helps completed 65536 passes in 1m9.082767904s, average time 1.054119ms

// it just can't make up its mind.
//   NO, I'm going to call it ok.: 1024 is working, but I'm going for the max!!

// for 128 GetPacketReply failed ServiceContact timed out waiting for reply in GetPacketReplyLonger # 52 after 17s
// --- FAIL: TestDialTCP_1000_helps (33.30s)
//     stressLoadNames_test.go:71: GetPacketReply failed ServiceContact timed out waiting for reply in GetPacketReplyLonger # 23 after 17s

// -- FAIL: TestDialTCP_1000_helps (18.38s)
// stressLoadNames_test.go:63: GetPacketReply failed ServiceContact timed out waiting for reply in GetPacketReplyLonger # 4 after 17s
// jeez. I didn't even try to load the names yet.

func TestDialTCP_1000_helps(t *testing.T) {

	ce := makeClusterWithServiceContact()

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()

	// wait for 5 sec, just to be silly. It's not supposed to matter AT ALL.
	sleepTime := 5 * time.Second
	time.Sleep(sleepTime) // except now it worked. wtf. Keep going. I can't live like this.

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println()

	passes := 100 // 1024 * 64
	startTime := time.Now()
	for i := 0; i < passes; i++ {
		// Let's do "get help", you always love that one.
		name := "no-name-needed-for-help"
		command := "help"
		cmd := packets.Lookup{}
		cmd.Address.FromString(name)
		cmd.SetOption("cmd", []byte(command))

		// send it
		reply, err := ce.GetPacketService().GetPacketReply(&cmd)
		if err != nil {
			t.Fatal("GetPacketReply failed", err)
		}
		got := string(reply.(*packets.Send).Payload)
		iot.CheckSendPacket(reply.(*packets.Send))
		ok := strings.Contains(got, "[help] lists all commands. 🔓 means no encryption required")
		if !ok {
			t.Fatal("GetPacketReply did not return expected help message")
		}
		// bulky fmt.Println("TestDialTCP_one_at_a_time got help reply", got)
	}
	endTime := time.Now()
	elapsed := endTime.Sub(startTime)
	fmt.Printf("TestDialTCP_1000_helps completed %d passes in %s, average time %s\n", passes, elapsed, elapsed/time.Duration(passes))
}

// TestDialTCP_1000_helps_Prod calls prod with the simplest of the lookup commands.
// it does require for the service-contact to be running and for the guru to be reachable etc.
// It's really only 100, not 1000 lol.
// Here's the one offs:
// real easy: curl "https://knotfree.net/api1/getPublicKey"
// slightly more: curl "https://knotfree.net/api1/nameService?name=dummmyName&cmd=help"

// This is how far away Texas is from Reno:
// TestDialTCP_1000_helps_Prod completed 100 passes in 5.630100194s, average time 56.301001ms

func TestDialTCP_1000_helps_Prod(t *testing.T) {

	// ce := makeClusterWithServiceContact()

	fmt.Println()
	fmt.Println()
	fmt.Println()

	// curl "https://knotfree.net/api1/nameService?name=somename&cmd=help"
	failCount := 0

	passes := 100 // 1024
	startTime := time.Now()
	for i := 0; i < passes; i++ {

		url := "https://knotfree.net/api1/nameService?name=dummmyName&cmd=help"
		// url := "http://knotfree.io/api1/nameService?name=dummmyName&cmd=help"

		// fmt.Println("TestDialTCP_by_batches PROD url:", url)

		resp, err := http.Get(url)
		assert.NoError(t, err)
		if err != nil {
			t.Errorf("Error fetching URL: %v", err)
			failCount++
			continue
		}
		defer resp.Body.Close()

		// a buffer
		buf := make([]byte, 1024) // it's 660
		n, err := resp.Body.Read(buf)
		if err != nil && err.Error() != "EOF" { // is EOF normal?
			failCount++
			continue
		}
		//	fmt.Printf("body length: %d\n", n)
		got := string(buf[:n])
		ok := strings.Contains(got, "[help] lists all commands. 🔓 means no encryption required")
		if !ok {
			t.Fatal("GetPacketReply did not return expected help message")
		}
		// bulky fmt.Println("TestDialTCP_1000_helps_Prod got help reply", got)
	}
	endTime := time.Now()
	elapsed := endTime.Sub(startTime)
	fmt.Printf("TestDialTCP_1000_helps_Prod completed %d passes in %s, average time %s\n", passes, elapsed, elapsed/time.Duration(passes))
}

var names = [][]string{
	{"testmain-0n0u0e16p-0", "testmain-1s0u0e16p-1", "testmain-0n1d0e16p-2", "testmain-1s1d0e16p-3", "testmain-0n0u1w16p-4", "testmain-1s0u1w16p-5", "testmain-0n1d1w16p-6", "testmain-1s1d1w16p-7"},
	{"testmain-0n0u0e15p-0", "testmain-0n0u0e15p-1", "testmain-0n0u0e15p-2", "testmain-0n0u0e15p-3", "testmain-0n0u0e15p-4", "testmain-0n0u0e15p-5", "testmain-0n0u0e15p-6", "testmain-0n0u0e15p-7", "testmain-0n0u0e14p", "testmain-1n0u0e14p", "testmain-0n1u0e14p", "testmain-1n1u0e14p", "testmain-0n0u1e14p", "testmain-1n0u1e14p", "testmain-0n1u1e14p", "testmain-1n1u1e14p"},
	{"testmain-0n0u1w15p-0", "testmain-0n0u1w15p-1", "testmain-0n0u1w15p-2", "testmain-0n0u1w15p-3", "testmain-0n0u1w15p-4", "testmain-0n0u1w15p-5", "testmain-0n0u1w15p-6", "testmain-0n0u1w15p-7", "testmain-0n0u2w14p", "testmain-1n0u2w14p", "testmain-0n1u2w14p", "testmain-1n1u2w14p", "testmain-0n0u1w14p", "testmain-1n0u1w14p", "testmain-0n1u1w14p", "testmain-1n1u1w14p"},
	{"testmain-0n0u0e14p-0", "testmain-0n0u0e14p-1", "testmain-0n0u0e14p-2", "testmain-0n0u0e14p-3", "testmain-0n0u0e14p-4", "testmain-0n0u0e14p-5", "testmain-0n0u0e14p-6", "testmain-0n0u0e14p-7", "testmain-0n0u0e13p", "testmain-1n0u0e13p", "testmain-0n1u0e13p", "testmain-1n1u0e13p", "testmain-0n0u1e13p", "testmain-1n0u1e13p", "testmain-0n1u1e13p", "testmain-1n1u1e13p"},
	{"testmain-0n0u0e13p-0", "testmain-0n0u0e13p-1", "testmain-0n0u0e13p-2", "testmain-0n0u0e13p-3", "testmain-0n0u0e13p-4", "testmain-0n0u0e13p-5", "testmain-0n0u0e13p-6", "testmain-0n0u0e13p-7", "testmain-0n0u0e12p", "testmain-1n0u0e12p", "testmain-0n1u0e12p", "testmain-1n1u0e12p", "testmain-0n0u1e12p", "testmain-1n0u1e12p", "testmain-0n1u1e12p", "testmain-1n1u1e12p"},
	{"testmain-0n0u0e12p-0", "testmain-0n0u0e12p-1", "testmain-0n0u0e12p-2", "testmain-0n0u0e12p-3", "testmain-0n0u0e12p-4", "testmain-0n0u0e12p-5", "testmain-0n0u0e12p-6", "testmain-0n0u0e12p-7", "testmain-0n0u0e11p", "testmain-1n0u0e11p", "testmain-0n1u0e11p", "testmain-1n1u0e11p", "testmain-0n0u1e11p", "testmain-1n0u1e11p", "testmain-0n1u1e11p", "testmain-1n1u1e11p"},
	{"testmain-0n0u0e11p-0", "testmain-0n0u0e11p-1", "testmain-0n0u0e11p-2", "testmain-0n0u0e11p-3", "testmain-0n0u0e11p-4", "testmain-0n0u0e11p-5", "testmain-0n0u0e11p-6", "testmain-0n0u0e11p-7", "testmain-0n0u0e10p", "testmain-1n0u0e10p", "testmain-0n1u0e10p", "testmain-1n1u0e10p", "testmain-0n0u1e10p", "testmain-1n0u1e10p", "testmain-0n1u1e10p", "testmain-1n1u1e10p"},
	{"testmain-0n0u0e10p-0", "testmain-0n0u0e10p-1", "testmain-0n0u0e10p-2", "testmain-0n0u0e10p-3", "testmain-0n0u0e10p-4", "testmain-0n0u0e10p-5", "testmain-0n0u0e10p-6", "testmain-0n0u0e10p-7", "testmain-0n0u0e9p", "testmain-1n0u0e9p", "testmain-0n1u0e9p", "testmain-1n1u0e9p", "testmain-0n0u1e9p", "testmain-1n0u1e9p", "testmain-0n1u1e9p", "testmain-1n1u1e9p"},
	{"testmain-0n0u0e9p-0", "testmain-0n0u0e9p-1", "testmain-0n0u0e9p-2", "testmain-0n0u0e9p-3", "testmain-0n0u0e9p-4", "testmain-0n0u0e9p-5", "testmain-0n0u0e9p-6", "testmain-0n0u0e9p-7", "testmain-0n0u0e8p", "testmain-1n0u0e8p", "testmain-0n1u0e8p", "testmain-1n1u0e8p", "testmain-0n0u1e8p", "testmain-1n0u1e8p", "testmain-0n1u1e8p", "testmain-1n1u1e8p"},
	{"testmain-0n0u0e8p-0", "testmain-0n0u0e8p-1", "testmain-0n0u0e8p-2", "testmain-0n0u0e8p-3", "testmain-0n0u0e8p-4", "testmain-0n0u0e8p-5", "testmain-0n0u0e8p-6", "testmain-0n0u0e8p-7", "testmain-0n0u0e7p", "testmain-1n0u0e7p", "testmain-0n1u0e7p", "testmain-1n1u0e7p", "testmain-0n0u1e7p", "testmain-1n0u1e7p", "testmain-0n1u1e7p", "testmain-1n1u1e7p"},
	{"testmain-0n0u0e7p-0", "testmain-0n0u0e7p-1", "testmain-0n0u0e7p-2", "testmain-0n0u0e7p-3", "testmain-0n0u0e7p-4", "testmain-0n0u0e7p-5", "testmain-0n0u0e7p-6", "testmain-0n0u0e7p-7", "testmain-0n0u0e6p", "testmain-1n0u0e6p", "testmain-0n1u0e6p", "testmain-1n1u0e6p", "testmain-0n0u1e6p", "testmain-1n0u1e6p", "testmain-0n1u1e6p", "testmain-1n1u1e6p"},
	{"testmain-0n0u0e6p-0", "testmain-0n0u0e6p-1", "testmain-0n0u0e6p-2", "testmain-0n0u0e6p-3", "testmain-0n0u0e6p-4", "testmain-0n0u0e6p-5", "testmain-0n0u0e6p-6", "testmain-0n0u0e6p-7", "testmain-0n0u0e5p", "testmain-1n0u0e5p", "testmain-0n1u0e5p", "testmain-1n1u0e5p", "testmain-0n0u1e5p", "testmain-1n0u1e5p", "testmain-0n1u1e5p", "testmain-1n1u1e5p"},
	{"testmain-0n0u1w14p-0", "testmain-0n0u1w14p-1", "testmain-0n0u1w14p-2", "testmain-0n0u1w14p-3", "testmain-0n0u1w14p-4", "testmain-0n0u1w14p-5", "testmain-0n0u1w14p-6", "testmain-0n0u1w14p-7", "testmain-0n0u2w13p", "testmain-1n0u2w13p", "testmain-0n1u2w13p", "testmain-1n1u2w13p", "testmain-0n0u1w13p", "testmain-1n0u1w13p", "testmain-0n1u1w13p", "testmain-1n1u1w13p"},
	{"testmain-0n0u1w13p-0", "testmain-0n0u1w13p-1", "testmain-0n0u1w13p-2", "testmain-0n0u1w13p-3", "testmain-0n0u1w13p-4", "testmain-0n0u1w13p-5", "testmain-0n0u1w13p-6", "testmain-0n0u1w13p-7", "testmain-0n0u2w12p", "testmain-1n0u2w12p", "testmain-0n1u2w12p", "testmain-1n1u2w12p", "testmain-0n0u1w12p", "testmain-1n0u1w12p", "testmain-0n1u1w12p", "testmain-1n1u1w12p"},
	{"testmain-0n0u1w12p-0", "testmain-0n0u1w12p-1", "testmain-0n0u1w12p-2", "testmain-0n0u1w12p-3", "testmain-0n0u1w12p-4", "testmain-0n0u1w12p-5", "testmain-0n0u1w12p-6", "testmain-0n0u1w12p-7", "testmain-0n0u2w11p", "testmain-1n0u2w11p", "testmain-0n1u2w11p", "testmain-1n1u2w11p", "testmain-0n0u1w11p", "testmain-1n0u1w11p", "testmain-0n1u1w11p", "testmain-1n1u1w11p"},
	{"testmain-0n0u1w11p-0", "testmain-0n0u1w11p-1", "testmain-0n0u1w11p-2", "testmain-0n0u1w11p-3", "testmain-0n0u1w11p-4", "testmain-0n0u1w11p-5", "testmain-0n0u1w11p-6", "testmain-0n0u1w11p-7", "testmain-0n0u2w10p", "testmain-1n0u2w10p", "testmain-0n1u2w10p", "testmain-1n1u2w10p", "testmain-0n0u1w10p", "testmain-1n0u1w10p", "testmain-0n1u1w10p", "testmain-1n1u1w10p"},
	{"testmain-0n0u1w10p-0", "testmain-0n0u1w10p-1", "testmain-0n0u1w10p-2", "testmain-0n0u1w10p-3", "testmain-0n0u1w10p-4", "testmain-0n0u1w10p-5", "testmain-0n0u1w10p-6", "testmain-0n0u1w10p-7", "testmain-0n0u2w9p", "testmain-1n0u2w9p", "testmain-0n1u2w9p", "testmain-1n1u2w9p", "testmain-0n0u1w9p", "testmain-1n0u1w9p", "testmain-0n1u1w9p", "testmain-1n1u1w9p"},
	{"testmain-0n0u2w9p-0", "testmain-0n0u2w9p-1", "testmain-0n0u2w9p-2", "testmain-0n0u2w9p-3", "testmain-0n0u2w9p-4", "testmain-0n0u2w9p-5", "testmain-0n0u2w9p-6", "testmain-0n0u2w9p-7", "testmain-0n0u4w8p", "testmain-1n0u4w8p", "testmain-0n1u4w8p", "testmain-1n1u4w8p", "testmain-0n0u3w8p", "testmain-1n0u3w8p", "testmain-0n1u3w8p", "testmain-1n1u3w8p"},
	{"testmain-0n0u4w8p-0", "testmain-0n0u4w8p-1", "testmain-0n0u4w8p-2", "testmain-0n0u4w8p-3", "testmain-0n0u4w8p-4", "testmain-0n0u4w8p-5", "testmain-0n0u4w8p-6", "testmain-0n0u4w8p-7", "testmain-0n0u8w7p", "testmain-1n0u8w7p", "testmain-0n1u8w7p", "testmain-1n1u8w7p", "testmain-0n0u7w7p", "testmain-1n0u7w7p", "testmain-0n1u7w7p", "testmain-1n1u7w7p"},
	{"testmain-0n0u8w7p-0", "testmain-0n0u8w7p-1", "testmain-0n0u8w7p-2", "testmain-0n0u8w7p-3", "testmain-0n0u8w7p-4", "testmain-0n0u8w7p-5", "testmain-0n0u8w7p-6", "testmain-0n0u8w7p-7", "testmain-0n0u16w6p", "testmain-1n0u16w6p", "testmain-0n1u16w6p", "testmain-1n1u16w6p", "testmain-0n0u15w6p", "testmain-1n0u15w6p", "testmain-0n1u15w6p", "testmain-1n1u15w6p"},
	{"testmain-0n0u16w6p-0", "testmain-0n0u16w6p-1", "testmain-0n0u16w6p-2", "testmain-0n0u16w6p-3", "testmain-0n0u16w6p-4", "testmain-0n0u16w6p-5", "testmain-0n0u16w6p-6", "testmain-0n0u16w6p-7", "testmain-0n0u32w5p", "testmain-1n0u32w5p", "testmain-0n1u32w5p", "testmain-1n1u32w5p", "testmain-0n0u31w5p", "testmain-1n0u31w5p", "testmain-0n1u31w5p", "testmain-1n1u31w5p"},
	{"testmain-0n0u32w5p-0", "testmain-0n0u32w5p-1", "testmain-0n0u32w5p-2", "testmain-0n0u32w5p-3", "testmain-0n0u32w5p-4", "testmain-0n0u32w5p-5", "testmain-0n0u32w5p-6", "testmain-0n0u32w5p-7", "testmain-0n0u64w4p", "testmain-1n0u64w4p", "testmain-0n1u64w4p", "testmain-1n1u64w4p", "testmain-0n0u63w4p", "testmain-1n0u63w4p", "testmain-0n1u63w4p", "testmain-1n1u63w4p"},
	{"testmain-0n0u31w5p-0", "testmain-0n0u31w5p-1", "testmain-0n0u31w5p-2", "testmain-0n0u31w5p-3", "testmain-0n0u31w5p-4", "testmain-0n0u31w5p-5", "testmain-0n0u31w5p-6", "testmain-0n0u31w5p-7", "testmain-0n0u62w4p", "testmain-1n0u62w4p", "testmain-0n1u62w4p", "testmain-1n1u62w4p", "testmain-0n0u61w4p", "testmain-1n0u61w4p", "testmain-0n1u61w4p", "testmain-1n1u61w4p"},
	{"testmain-0n0u15w6p-0", "testmain-0n0u15w6p-1", "testmain-0n0u15w6p-2", "testmain-0n0u15w6p-3", "testmain-0n0u15w6p-4", "testmain-0n0u15w6p-5", "testmain-0n0u15w6p-6", "testmain-0n0u15w6p-7", "testmain-0n0u30w5p", "testmain-1n0u30w5p", "testmain-0n1u30w5p", "testmain-1n1u30w5p", "testmain-0n0u29w5p", "testmain-1n0u29w5p", "testmain-0n1u29w5p", "testmain-1n1u29w5p"},
	{"testmain-0n0u30w5p-0", "testmain-0n0u30w5p-1", "testmain-0n0u30w5p-2", "testmain-0n0u30w5p-3", "testmain-0n0u30w5p-4", "testmain-0n0u30w5p-5", "testmain-0n0u30w5p-6", "testmain-0n0u30w5p-7", "testmain-0n0u60w4p", "testmain-1n0u60w4p", "testmain-0n1u60w4p", "testmain-1n1u60w4p", "testmain-0n0u59w4p", "testmain-1n0u59w4p", "testmain-0n1u59w4p", "testmain-1n1u59w4p"},
	{"testmain-0n0u29w5p-0", "testmain-0n0u29w5p-1", "testmain-0n0u29w5p-2", "testmain-0n0u29w5p-3", "testmain-0n0u29w5p-4", "testmain-0n0u29w5p-5", "testmain-0n0u29w5p-6", "testmain-0n0u29w5p-7", "testmain-0n0u58w4p", "testmain-1n0u58w4p", "testmain-0n1u58w4p", "testmain-1n1u58w4p", "testmain-0n0u57w4p", "testmain-1n0u57w4p", "testmain-0n1u57w4p", "testmain-1n1u57w4p"},
	{"testmain-0n0u7w7p-0", "testmain-0n0u7w7p-1", "testmain-0n0u7w7p-2", "testmain-0n0u7w7p-3", "testmain-0n0u7w7p-4", "testmain-0n0u7w7p-5", "testmain-0n0u7w7p-6", "testmain-0n0u7w7p-7", "testmain-0n0u14w6p", "testmain-1n0u14w6p", "testmain-0n1u14w6p", "testmain-1n1u14w6p", "testmain-0n0u13w6p", "testmain-1n0u13w6p", "testmain-0n1u13w6p", "testmain-1n1u13w6p"},
	{"testmain-0n0u14w6p-0", "testmain-0n0u14w6p-1", "testmain-0n0u14w6p-2", "testmain-0n0u14w6p-3", "testmain-0n0u14w6p-4", "testmain-0n0u14w6p-5", "testmain-0n0u14w6p-6", "testmain-0n0u14w6p-7", "testmain-0n0u28w5p", "testmain-1n0u28w5p", "testmain-0n1u28w5p", "testmain-1n1u28w5p", "testmain-0n0u27w5p", "testmain-1n0u27w5p", "testmain-0n1u27w5p", "testmain-1n1u27w5p"},
	{"testmain-0n0u28w5p-0", "testmain-0n0u28w5p-1", "testmain-0n0u28w5p-2", "testmain-0n0u28w5p-3", "testmain-0n0u28w5p-4", "testmain-0n0u28w5p-5", "testmain-0n0u28w5p-6", "testmain-0n0u28w5p-7", "testmain-0n0u56w4p", "testmain-1n0u56w4p", "testmain-0n1u56w4p", "testmain-1n1u56w4p", "testmain-0n0u55w4p", "testmain-1n0u55w4p", "testmain-0n1u55w4p", "testmain-1n1u55w4p"},
	{"testmain-0n0u27w5p-0", "testmain-0n0u27w5p-1", "testmain-0n0u27w5p-2", "testmain-0n0u27w5p-3", "testmain-0n0u27w5p-4", "testmain-0n0u27w5p-5", "testmain-0n0u27w5p-6", "testmain-0n0u27w5p-7", "testmain-0n0u54w4p", "testmain-1n0u54w4p", "testmain-0n1u54w4p", "testmain-1n1u54w4p", "testmain-0n0u53w4p", "testmain-1n0u53w4p", "testmain-0n1u53w4p", "testmain-1n1u53w4p"},
	{"testmain-0n0u13w6p-0", "testmain-0n0u13w6p-1", "testmain-0n0u13w6p-2", "testmain-0n0u13w6p-3", "testmain-0n0u13w6p-4", "testmain-0n0u13w6p-5", "testmain-0n0u13w6p-6", "testmain-0n0u13w6p-7", "testmain-0n0u26w5p", "testmain-1n0u26w5p", "testmain-0n1u26w5p", "testmain-1n1u26w5p", "testmain-0n0u25w5p", "testmain-1n0u25w5p", "testmain-0n1u25w5p", "testmain-1n1u25w5p"},
	{"testmain-0n0u26w5p-0", "testmain-0n0u26w5p-1", "testmain-0n0u26w5p-2", "testmain-0n0u26w5p-3", "testmain-0n0u26w5p-4", "testmain-0n0u26w5p-5", "testmain-0n0u26w5p-6", "testmain-0n0u26w5p-7", "testmain-0n0u52w4p", "testmain-1n0u52w4p", "testmain-0n1u52w4p", "testmain-1n1u52w4p", "testmain-0n0u51w4p", "testmain-1n0u51w4p", "testmain-0n1u51w4p", "testmain-1n1u51w4p"},
	{"testmain-0n0u25w5p-0", "testmain-0n0u25w5p-1", "testmain-0n0u25w5p-2", "testmain-0n0u25w5p-3", "testmain-0n0u25w5p-4", "testmain-0n0u25w5p-5", "testmain-0n0u25w5p-6", "testmain-0n0u25w5p-7", "testmain-0n0u50w4p", "testmain-1n0u50w4p", "testmain-0n1u50w4p", "testmain-1n1u50w4p", "testmain-0n0u49w4p", "testmain-1n0u49w4p", "testmain-0n1u49w4p", "testmain-1n1u49w4p"},
	{"testmain-0n0u3w8p-0", "testmain-0n0u3w8p-1", "testmain-0n0u3w8p-2", "testmain-0n0u3w8p-3", "testmain-0n0u3w8p-4", "testmain-0n0u3w8p-5", "testmain-0n0u3w8p-6", "testmain-0n0u3w8p-7", "testmain-0n0u6w7p", "testmain-1n0u6w7p", "testmain-0n1u6w7p", "testmain-1n1u6w7p", "testmain-0n0u5w7p", "testmain-1n0u5w7p", "testmain-0n1u5w7p", "testmain-1n1u5w7p"},
	{"testmain-0n0u6w7p-0", "testmain-0n0u6w7p-1", "testmain-0n0u6w7p-2", "testmain-0n0u6w7p-3", "testmain-0n0u6w7p-4", "testmain-0n0u6w7p-5", "testmain-0n0u6w7p-6", "testmain-0n0u6w7p-7", "testmain-0n0u12w6p", "testmain-1n0u12w6p", "testmain-0n1u12w6p", "testmain-1n1u12w6p", "testmain-0n0u11w6p", "testmain-1n0u11w6p", "testmain-0n1u11w6p", "testmain-1n1u11w6p"},
	{"testmain-0n0u12w6p-0", "testmain-0n0u12w6p-1", "testmain-0n0u12w6p-2", "testmain-0n0u12w6p-3", "testmain-0n0u12w6p-4", "testmain-0n0u12w6p-5", "testmain-0n0u12w6p-6", "testmain-0n0u12w6p-7", "testmain-0n0u24w5p", "testmain-1n0u24w5p", "testmain-0n1u24w5p", "testmain-1n1u24w5p", "testmain-0n0u23w5p", "testmain-1n0u23w5p", "testmain-0n1u23w5p", "testmain-1n1u23w5p"},
	{"testmain-0n0u24w5p-0", "testmain-0n0u24w5p-1", "testmain-0n0u24w5p-2", "testmain-0n0u24w5p-3", "testmain-0n0u24w5p-4", "testmain-0n0u24w5p-5", "testmain-0n0u24w5p-6", "testmain-0n0u24w5p-7", "testmain-0n0u48w4p", "testmain-1n0u48w4p", "testmain-0n1u48w4p", "testmain-1n1u48w4p", "testmain-0n0u47w4p", "testmain-1n0u47w4p", "testmain-0n1u47w4p", "testmain-1n1u47w4p"},
	{"testmain-0n0u23w5p-0", "testmain-0n0u23w5p-1", "testmain-0n0u23w5p-2", "testmain-0n0u23w5p-3", "testmain-0n0u23w5p-4", "testmain-0n0u23w5p-5", "testmain-0n0u23w5p-6", "testmain-0n0u23w5p-7", "testmain-0n0u46w4p", "testmain-1n0u46w4p", "testmain-0n1u46w4p", "testmain-1n1u46w4p", "testmain-0n0u45w4p", "testmain-1n0u45w4p", "testmain-0n1u45w4p", "testmain-1n1u45w4p"},
	{"testmain-0n0u11w6p-0", "testmain-0n0u11w6p-1", "testmain-0n0u11w6p-2", "testmain-0n0u11w6p-3", "testmain-0n0u11w6p-4", "testmain-0n0u11w6p-5", "testmain-0n0u11w6p-6", "testmain-0n0u11w6p-7", "testmain-0n0u22w5p", "testmain-1n0u22w5p", "testmain-0n1u22w5p", "testmain-1n1u22w5p", "testmain-0n0u21w5p", "testmain-1n0u21w5p", "testmain-0n1u21w5p", "testmain-1n1u21w5p"},
	{"testmain-0n0u22w5p-0", "testmain-0n0u22w5p-1", "testmain-0n0u22w5p-2", "testmain-0n0u22w5p-3", "testmain-0n0u22w5p-4", "testmain-0n0u22w5p-5", "testmain-0n0u22w5p-6", "testmain-0n0u22w5p-7", "testmain-0n0u44w4p", "testmain-1n0u44w4p", "testmain-0n1u44w4p", "testmain-1n1u44w4p", "testmain-0n0u43w4p", "testmain-1n0u43w4p", "testmain-0n1u43w4p", "testmain-1n1u43w4p"},
	{"testmain-0n0u21w5p-0", "testmain-0n0u21w5p-1", "testmain-0n0u21w5p-2", "testmain-0n0u21w5p-3", "testmain-0n0u21w5p-4", "testmain-0n0u21w5p-5", "testmain-0n0u21w5p-6", "testmain-0n0u21w5p-7", "testmain-0n0u42w4p", "testmain-1n0u42w4p", "testmain-0n1u42w4p", "testmain-1n1u42w4p", "testmain-0n0u41w4p", "testmain-1n0u41w4p", "testmain-0n1u41w4p", "testmain-1n1u41w4p"},
	{"testmain-0n0u5w7p-0", "testmain-0n0u5w7p-1", "testmain-0n0u5w7p-2", "testmain-0n0u5w7p-3", "testmain-0n0u5w7p-4", "testmain-0n0u5w7p-5", "testmain-0n0u5w7p-6", "testmain-0n0u5w7p-7", "testmain-0n0u10w6p", "testmain-1n0u10w6p", "testmain-0n1u10w6p", "testmain-1n1u10w6p", "testmain-0n0u9w6p", "testmain-1n0u9w6p", "testmain-0n1u9w6p", "testmain-1n1u9w6p"},
	{"testmain-0n0u10w6p-0", "testmain-0n0u10w6p-1", "testmain-0n0u10w6p-2", "testmain-0n0u10w6p-3", "testmain-0n0u10w6p-4", "testmain-0n0u10w6p-5", "testmain-0n0u10w6p-6", "testmain-0n0u10w6p-7", "testmain-0n0u20w5p", "testmain-1n0u20w5p", "testmain-0n1u20w5p", "testmain-1n1u20w5p", "testmain-0n0u19w5p", "testmain-1n0u19w5p", "testmain-0n1u19w5p", "testmain-1n1u19w5p"},
	{"testmain-0n0u20w5p-0", "testmain-0n0u20w5p-1", "testmain-0n0u20w5p-2", "testmain-0n0u20w5p-3", "testmain-0n0u20w5p-4", "testmain-0n0u20w5p-5", "testmain-0n0u20w5p-6", "testmain-0n0u20w5p-7", "testmain-0n0u40w4p", "testmain-1n0u40w4p", "testmain-0n1u40w4p", "testmain-1n1u40w4p", "testmain-0n0u39w4p", "testmain-1n0u39w4p", "testmain-0n1u39w4p", "testmain-1n1u39w4p"},
	{"testmain-0n0u19w5p-0", "testmain-0n0u19w5p-1", "testmain-0n0u19w5p-2", "testmain-0n0u19w5p-3", "testmain-0n0u19w5p-4", "testmain-0n0u19w5p-5", "testmain-0n0u19w5p-6", "testmain-0n0u19w5p-7", "testmain-0n0u38w4p", "testmain-1n0u38w4p", "testmain-0n1u38w4p", "testmain-1n1u38w4p", "testmain-0n0u37w4p", "testmain-1n0u37w4p", "testmain-0n1u37w4p", "testmain-1n1u37w4p"},
	{"testmain-0n0u9w6p-0", "testmain-0n0u9w6p-1", "testmain-0n0u9w6p-2", "testmain-0n0u9w6p-3", "testmain-0n0u9w6p-4", "testmain-0n0u9w6p-5", "testmain-0n0u9w6p-6", "testmain-0n0u9w6p-7", "testmain-0n0u18w5p", "testmain-1n0u18w5p", "testmain-0n1u18w5p", "testmain-1n1u18w5p", "testmain-0n0u17w5p", "testmain-1n0u17w5p", "testmain-0n1u17w5p", "testmain-1n1u17w5p"},
	{"testmain-0n0u18w5p-0", "testmain-0n0u18w5p-1", "testmain-0n0u18w5p-2", "testmain-0n0u18w5p-3", "testmain-0n0u18w5p-4", "testmain-0n0u18w5p-5", "testmain-0n0u18w5p-6", "testmain-0n0u18w5p-7", "testmain-0n0u36w4p", "testmain-1n0u36w4p", "testmain-0n1u36w4p", "testmain-1n1u36w4p", "testmain-0n0u35w4p", "testmain-1n0u35w4p", "testmain-0n1u35w4p", "testmain-1n1u35w4p"},
	{"testmain-0n0u17w5p-0", "testmain-0n0u17w5p-1", "testmain-0n0u17w5p-2", "testmain-0n0u17w5p-3", "testmain-0n0u17w5p-4", "testmain-0n0u17w5p-5", "testmain-0n0u17w5p-6", "testmain-0n0u17w5p-7", "testmain-0n0u34w4p", "testmain-1n0u34w4p", "testmain-0n1u34w4p", "testmain-1n1u34w4p", "testmain-0n0u33w4p", "testmain-1n0u33w4p", "testmain-0n1u33w4p", "testmain-1n1u33w4p"},
	{"testmain-0n0u1w9p-0", "testmain-0n0u1w9p-1", "testmain-0n0u1w9p-2", "testmain-0n0u1w9p-3", "testmain-0n0u1w9p-4", "testmain-0n0u1w9p-5", "testmain-0n0u1w9p-6", "testmain-0n0u1w9p-7", "testmain-0n0u2w8p", "testmain-1n0u2w8p", "testmain-0n1u2w8p", "testmain-1n1u2w8p", "testmain-0n0u1w8p", "testmain-1n0u1w8p", "testmain-0n1u1w8p", "testmain-1n1u1w8p"},
	{"testmain-0n0u2w8p-0", "testmain-0n0u2w8p-1", "testmain-0n0u2w8p-2", "testmain-0n0u2w8p-3", "testmain-0n0u2w8p-4", "testmain-0n0u2w8p-5", "testmain-0n0u2w8p-6", "testmain-0n0u2w8p-7", "testmain-0n0u4w7p", "testmain-1n0u4w7p", "testmain-0n1u4w7p", "testmain-1n1u4w7p", "testmain-0n0u3w7p", "testmain-1n0u3w7p", "testmain-0n1u3w7p", "testmain-1n1u3w7p"},
	{"testmain-0n0u4w7p-0", "testmain-0n0u4w7p-1", "testmain-0n0u4w7p-2", "testmain-0n0u4w7p-3", "testmain-0n0u4w7p-4", "testmain-0n0u4w7p-5", "testmain-0n0u4w7p-6", "testmain-0n0u4w7p-7", "testmain-0n0u8w6p", "testmain-1n0u8w6p", "testmain-0n1u8w6p", "testmain-1n1u8w6p", "testmain-0n0u7w6p", "testmain-1n0u7w6p", "testmain-0n1u7w6p", "testmain-1n1u7w6p"},
	{"testmain-0n0u8w6p-0", "testmain-0n0u8w6p-1", "testmain-0n0u8w6p-2", "testmain-0n0u8w6p-3", "testmain-0n0u8w6p-4", "testmain-0n0u8w6p-5", "testmain-0n0u8w6p-6", "testmain-0n0u8w6p-7", "testmain-0n0u16w5p", "testmain-1n0u16w5p", "testmain-0n1u16w5p", "testmain-1n1u16w5p", "testmain-0n0u15w5p", "testmain-1n0u15w5p", "testmain-0n1u15w5p", "testmain-1n1u15w5p"},
	{"testmain-0n0u16w5p-0", "testmain-0n0u16w5p-1", "testmain-0n0u16w5p-2", "testmain-0n0u16w5p-3", "testmain-0n0u16w5p-4", "testmain-0n0u16w5p-5", "testmain-0n0u16w5p-6", "testmain-0n0u16w5p-7", "testmain-0n0u32w4p", "testmain-1n0u32w4p", "testmain-0n1u32w4p", "testmain-1n1u32w4p", "testmain-0n0u31w4p", "testmain-1n0u31w4p", "testmain-0n1u31w4p", "testmain-1n1u31w4p"},
	{"testmain-0n0u15w5p-0", "testmain-0n0u15w5p-1", "testmain-0n0u15w5p-2", "testmain-0n0u15w5p-3", "testmain-0n0u15w5p-4", "testmain-0n0u15w5p-5", "testmain-0n0u15w5p-6", "testmain-0n0u15w5p-7", "testmain-0n0u30w4p", "testmain-1n0u30w4p", "testmain-0n1u30w4p", "testmain-1n1u30w4p", "testmain-0n0u29w4p", "testmain-1n0u29w4p", "testmain-0n1u29w4p", "testmain-1n1u29w4p"},
	{"testmain-0n0u7w6p-0", "testmain-0n0u7w6p-1", "testmain-0n0u7w6p-2", "testmain-0n0u7w6p-3", "testmain-0n0u7w6p-4", "testmain-0n0u7w6p-5", "testmain-0n0u7w6p-6", "testmain-0n0u7w6p-7", "testmain-0n0u14w5p", "testmain-1n0u14w5p", "testmain-0n1u14w5p", "testmain-1n1u14w5p", "testmain-0n0u13w5p", "testmain-1n0u13w5p", "testmain-0n1u13w5p", "testmain-1n1u13w5p"},
	{"testmain-0n0u14w5p-0", "testmain-0n0u14w5p-1", "testmain-0n0u14w5p-2", "testmain-0n0u14w5p-3", "testmain-0n0u14w5p-4", "testmain-0n0u14w5p-5", "testmain-0n0u14w5p-6", "testmain-0n0u14w5p-7", "testmain-0n0u28w4p", "testmain-1n0u28w4p", "testmain-0n1u28w4p", "testmain-1n1u28w4p", "testmain-0n0u27w4p", "testmain-1n0u27w4p", "testmain-0n1u27w4p", "testmain-1n1u27w4p"},
	{"testmain-0n0u13w5p-0", "testmain-0n0u13w5p-1", "testmain-0n0u13w5p-2", "testmain-0n0u13w5p-3", "testmain-0n0u13w5p-4", "testmain-0n0u13w5p-5", "testmain-0n0u13w5p-6", "testmain-0n0u13w5p-7", "testmain-0n0u26w4p", "testmain-1n0u26w4p", "testmain-0n1u26w4p", "testmain-1n1u26w4p", "testmain-0n0u25w4p", "testmain-1n0u25w4p", "testmain-0n1u25w4p", "testmain-1n1u25w4p"},
	{"testmain-0n0u3w7p-0", "testmain-0n0u3w7p-1", "testmain-0n0u3w7p-2", "testmain-0n0u3w7p-3", "testmain-0n0u3w7p-4", "testmain-0n0u3w7p-5", "testmain-0n0u3w7p-6", "testmain-0n0u3w7p-7", "testmain-0n0u6w6p", "testmain-1n0u6w6p", "testmain-0n1u6w6p", "testmain-1n1u6w6p", "testmain-0n0u5w6p", "testmain-1n0u5w6p", "testmain-0n1u5w6p", "testmain-1n1u5w6p"},
	{"testmain-0n0u6w6p-0", "testmain-0n0u6w6p-1", "testmain-0n0u6w6p-2", "testmain-0n0u6w6p-3", "testmain-0n0u6w6p-4", "testmain-0n0u6w6p-5", "testmain-0n0u6w6p-6", "testmain-0n0u6w6p-7", "testmain-0n0u12w5p", "testmain-1n0u12w5p", "testmain-0n1u12w5p", "testmain-1n1u12w5p", "testmain-0n0u11w5p", "testmain-1n0u11w5p", "testmain-0n1u11w5p", "testmain-1n1u11w5p"},
	{"testmain-0n0u12w5p-0", "testmain-0n0u12w5p-1", "testmain-0n0u12w5p-2", "testmain-0n0u12w5p-3", "testmain-0n0u12w5p-4", "testmain-0n0u12w5p-5", "testmain-0n0u12w5p-6", "testmain-0n0u12w5p-7", "testmain-0n0u24w4p", "testmain-1n0u24w4p", "testmain-0n1u24w4p", "testmain-1n1u24w4p", "testmain-0n0u23w4p", "testmain-1n0u23w4p", "testmain-0n1u23w4p", "testmain-1n1u23w4p"},
	{"testmain-0n0u11w5p-0", "testmain-0n0u11w5p-1", "testmain-0n0u11w5p-2", "testmain-0n0u11w5p-3", "testmain-0n0u11w5p-4", "testmain-0n0u11w5p-5", "testmain-0n0u11w5p-6", "testmain-0n0u11w5p-7", "testmain-0n0u22w4p", "testmain-1n0u22w4p", "testmain-0n1u22w4p", "testmain-1n1u22w4p", "testmain-0n0u21w4p", "testmain-1n0u21w4p", "testmain-0n1u21w4p", "testmain-1n1u21w4p"},
	{"testmain-0n0u5w6p-0", "testmain-0n0u5w6p-1", "testmain-0n0u5w6p-2", "testmain-0n0u5w6p-3", "testmain-0n0u5w6p-4", "testmain-0n0u5w6p-5", "testmain-0n0u5w6p-6", "testmain-0n0u5w6p-7", "testmain-0n0u10w5p", "testmain-1n0u10w5p", "testmain-0n1u10w5p", "testmain-1n1u10w5p", "testmain-0n0u9w5p", "testmain-1n0u9w5p", "testmain-0n1u9w5p", "testmain-1n1u9w5p"},
	{"testmain-0n0u10w5p-0", "testmain-0n0u10w5p-1", "testmain-0n0u10w5p-2", "testmain-0n0u10w5p-3", "testmain-0n0u10w5p-4", "testmain-0n0u10w5p-5", "testmain-0n0u10w5p-6", "testmain-0n0u10w5p-7", "testmain-0n0u20w4p", "testmain-1n0u20w4p", "testmain-0n1u20w4p", "testmain-1n1u20w4p", "testmain-0n0u19w4p", "testmain-1n0u19w4p", "testmain-0n1u19w4p", "testmain-1n1u19w4p"},
	{"testmain-0n0u9w5p-0", "testmain-0n0u9w5p-1", "testmain-0n0u9w5p-2", "testmain-0n0u9w5p-3", "testmain-0n0u9w5p-4", "testmain-0n0u9w5p-5", "testmain-0n0u9w5p-6", "testmain-0n0u9w5p-7", "testmain-0n0u18w4p", "testmain-1n0u18w4p", "testmain-0n1u18w4p", "testmain-1n1u18w4p", "testmain-0n0u17w4p", "testmain-1n0u17w4p", "testmain-0n1u17w4p", "testmain-1n1u17w4p"},
	{"testmain-0n0u1w8p-0", "testmain-0n0u1w8p-1", "testmain-0n0u1w8p-2", "testmain-0n0u1w8p-3", "testmain-0n0u1w8p-4", "testmain-0n0u1w8p-5", "testmain-0n0u1w8p-6", "testmain-0n0u1w8p-7", "testmain-0n0u2w7p", "testmain-1n0u2w7p", "testmain-0n1u2w7p", "testmain-1n1u2w7p", "testmain-0n0u1w7p", "testmain-1n0u1w7p", "testmain-0n1u1w7p", "testmain-1n1u1w7p"},
	{"testmain-0n0u2w7p-0", "testmain-0n0u2w7p-1", "testmain-0n0u2w7p-2", "testmain-0n0u2w7p-3", "testmain-0n0u2w7p-4", "testmain-0n0u2w7p-5", "testmain-0n0u2w7p-6", "testmain-0n0u2w7p-7", "testmain-0n0u4w6p", "testmain-1n0u4w6p", "testmain-0n1u4w6p", "testmain-1n1u4w6p", "testmain-0n0u3w6p", "testmain-1n0u3w6p", "testmain-0n1u3w6p", "testmain-1n1u3w6p"},
	{"testmain-0n0u4w6p-0", "testmain-0n0u4w6p-1", "testmain-0n0u4w6p-2", "testmain-0n0u4w6p-3", "testmain-0n0u4w6p-4", "testmain-0n0u4w6p-5", "testmain-0n0u4w6p-6", "testmain-0n0u4w6p-7", "testmain-0n0u8w5p", "testmain-1n0u8w5p", "testmain-0n1u8w5p", "testmain-1n1u8w5p", "testmain-0n0u7w5p", "testmain-1n0u7w5p", "testmain-0n1u7w5p", "testmain-1n1u7w5p"},
	{"testmain-0n0u8w5p-0", "testmain-0n0u8w5p-1", "testmain-0n0u8w5p-2", "testmain-0n0u8w5p-3", "testmain-0n0u8w5p-4", "testmain-0n0u8w5p-5", "testmain-0n0u8w5p-6", "testmain-0n0u8w5p-7", "testmain-0n0u16w4p", "testmain-1n0u16w4p", "testmain-0n1u16w4p", "testmain-1n1u16w4p", "testmain-0n0u15w4p", "testmain-1n0u15w4p", "testmain-0n1u15w4p", "testmain-1n1u15w4p"},
	{"testmain-0n0u7w5p-0", "testmain-0n0u7w5p-1", "testmain-0n0u7w5p-2", "testmain-0n0u7w5p-3", "testmain-0n0u7w5p-4", "testmain-0n0u7w5p-5", "testmain-0n0u7w5p-6", "testmain-0n0u7w5p-7", "testmain-0n0u14w4p", "testmain-1n0u14w4p", "testmain-0n1u14w4p", "testmain-1n1u14w4p", "testmain-0n0u13w4p", "testmain-1n0u13w4p", "testmain-0n1u13w4p", "testmain-1n1u13w4p"},
	{"testmain-0n0u3w6p-0", "testmain-0n0u3w6p-1", "testmain-0n0u3w6p-2", "testmain-0n0u3w6p-3", "testmain-0n0u3w6p-4", "testmain-0n0u3w6p-5", "testmain-0n0u3w6p-6", "testmain-0n0u3w6p-7", "testmain-0n0u6w5p", "testmain-1n0u6w5p", "testmain-0n1u6w5p", "testmain-1n1u6w5p", "testmain-0n0u5w5p", "testmain-1n0u5w5p", "testmain-0n1u5w5p", "testmain-1n1u5w5p"},
	{"testmain-0n0u6w5p-0", "testmain-0n0u6w5p-1", "testmain-0n0u6w5p-2", "testmain-0n0u6w5p-3", "testmain-0n0u6w5p-4", "testmain-0n0u6w5p-5", "testmain-0n0u6w5p-6", "testmain-0n0u6w5p-7", "testmain-0n0u12w4p", "testmain-1n0u12w4p", "testmain-0n1u12w4p", "testmain-1n1u12w4p", "testmain-0n0u11w4p", "testmain-1n0u11w4p", "testmain-0n1u11w4p", "testmain-1n1u11w4p"},
	{"testmain-0n0u5w5p-0", "testmain-0n0u5w5p-1", "testmain-0n0u5w5p-2", "testmain-0n0u5w5p-3", "testmain-0n0u5w5p-4", "testmain-0n0u5w5p-5", "testmain-0n0u5w5p-6", "testmain-0n0u5w5p-7", "testmain-0n0u10w4p", "testmain-1n0u10w4p", "testmain-0n1u10w4p", "testmain-1n1u10w4p", "testmain-0n0u9w4p", "testmain-1n0u9w4p", "testmain-0n1u9w4p", "testmain-1n1u9w4p"},
	{"testmain-0n0u1w7p-0", "testmain-0n0u1w7p-1", "testmain-0n0u1w7p-2", "testmain-0n0u1w7p-3", "testmain-0n0u1w7p-4", "testmain-0n0u1w7p-5", "testmain-0n0u1w7p-6", "testmain-0n0u1w7p-7", "testmain-0n0u2w6p", "testmain-1n0u2w6p", "testmain-0n1u2w6p", "testmain-1n1u2w6p", "testmain-0n0u1w6p", "testmain-1n0u1w6p", "testmain-0n1u1w6p", "testmain-1n1u1w6p"},
	{"testmain-0n0u2w6p-0", "testmain-0n0u2w6p-1", "testmain-0n0u2w6p-2", "testmain-0n0u2w6p-3", "testmain-0n0u2w6p-4", "testmain-0n0u2w6p-5", "testmain-0n0u2w6p-6", "testmain-0n0u2w6p-7", "testmain-0n0u4w5p", "testmain-1n0u4w5p", "testmain-0n1u4w5p", "testmain-1n1u4w5p", "testmain-0n0u3w5p", "testmain-1n0u3w5p", "testmain-0n1u3w5p", "testmain-1n1u3w5p"},
	{"testmain-0n0u4w5p-0", "testmain-0n0u4w5p-1", "testmain-0n0u4w5p-2", "testmain-0n0u4w5p-3", "testmain-0n0u4w5p-4", "testmain-0n0u4w5p-5", "testmain-0n0u4w5p-6", "testmain-0n0u4w5p-7", "testmain-0n0u8w4p", "testmain-1n0u8w4p", "testmain-0n1u8w4p", "testmain-1n1u8w4p", "testmain-0n0u7w4p", "testmain-1n0u7w4p", "testmain-0n1u7w4p", "testmain-1n1u7w4p"},
	{"testmain-0n0u3w5p-0", "testmain-0n0u3w5p-1", "testmain-0n0u3w5p-2", "testmain-0n0u3w5p-3", "testmain-0n0u3w5p-4", "testmain-0n0u3w5p-5", "testmain-0n0u3w5p-6", "testmain-0n0u3w5p-7", "testmain-0n0u6w4p", "testmain-1n0u6w4p", "testmain-0n1u6w4p", "testmain-1n1u6w4p", "testmain-0n0u5w4p", "testmain-1n0u5w4p", "testmain-0n1u5w4p", "testmain-1n1u5w4p"},
	{"testmain-0n0u1w6p-0", "testmain-0n0u1w6p-1", "testmain-0n0u1w6p-2", "testmain-0n0u1w6p-3", "testmain-0n0u1w6p-4", "testmain-0n0u1w6p-5", "testmain-0n0u1w6p-6", "testmain-0n0u1w6p-7", "testmain-0n0u2w5p", "testmain-1n0u2w5p", "testmain-0n1u2w5p", "testmain-1n1u2w5p", "testmain-0n0u1w5p", "testmain-1n0u1w5p", "testmain-0n1u1w5p", "testmain-1n1u1w5p"},
	{"testmain-0n0u2w5p-0", "testmain-0n0u2w5p-1", "testmain-0n0u2w5p-2", "testmain-0n0u2w5p-3", "testmain-0n0u2w5p-4", "testmain-0n0u2w5p-5", "testmain-0n0u2w5p-6", "testmain-0n0u2w5p-7", "testmain-0n0u4w4p", "testmain-1n0u4w4p", "testmain-0n1u4w4p", "testmain-1n1u4w4p", "testmain-0n0u3w4p", "testmain-1n0u3w4p", "testmain-0n1u3w4p", "testmain-1n1u3w4p"},
	{"testmain-0n0u1w5p-0", "testmain-0n0u1w5p-1", "testmain-0n0u1w5p-2", "testmain-0n0u1w5p-3", "testmain-0n0u1w5p-4", "testmain-0n0u1w5p-5", "testmain-0n0u1w5p-6", "testmain-0n0u1w5p-7", "testmain-0n0u2w4p", "testmain-1n0u2w4p", "testmain-0n1u2w4p", "testmain-1n1u2w4p", "testmain-0n0u1w4p", "testmain-1n0u1w4p", "testmain-0n1u1w4p", "testmain-1n1u1w4p"},
	{"testmain-0n0u2w4p-0", "testmain-0n0u2w4p-1", "testmain-0n0u2w4p-2", "testmain-0n0u2w4p-3", "testmain-0n0u2w4p-4", "testmain-0n0u2w4p-5", "testmain-0n0u2w4p-6", "testmain-0n0u2w4p-7", "testmain-0n0u4w3p", "testmain-1n0u4w3p", "testmain-0n1u4w3p", "testmain-1n1u4w3p", "testmain-0n0u3w3p", "testmain-1n0u3w3p", "testmain-0n1u3w3p", "testmain-1n1u3w3p"},
	{"testmain-1n0u3w3p-0", "testmain-1n0u3w3p-1", "testmain-1n0u3w3p-2", "testmain-1n0u3w3p-3", "testmain-1n0u3w3p-4", "testmain-1n0u3w3p-5", "testmain-1n0u3w3p-6", "testmain-1n0u3w3p-7", "testmain-2n0u6w2p", "testmain-3n0u6w2p", "testmain-2n1u6w2p", "testmain-3n1u6w2p", "testmain-2n0u5w2p", "testmain-3n0u5w2p", "testmain-2n1u5w2p", "testmain-3n1u5w2p"},
	{"testmain-0n0u1w4p-0", "testmain-0n0u1w4p-1", "testmain-0n0u1w4p-2", "testmain-0n0u1w4p-3", "testmain-0n0u1w4p-4", "testmain-0n0u1w4p-5", "testmain-0n0u1w4p-6", "testmain-0n0u1w4p-7", "testmain-0n0u2w3p", "testmain-1n0u2w3p", "testmain-0n1u2w3p", "testmain-1n1u2w3p", "testmain-0n0u1w3p", "testmain-1n0u1w3p", "testmain-0n1u1w3p", "testmain-1n1u1w3p"},
	{"testmain-1n0u2w3p-0", "testmain-1n0u2w3p-1", "testmain-1n0u2w3p-2", "testmain-1n0u2w3p-3", "testmain-1n0u2w3p-4", "testmain-1n0u2w3p-5", "testmain-1n0u2w3p-6", "testmain-1n0u2w3p-7", "testmain-2n0u4w2p", "testmain-3n0u4w2p", "testmain-2n1u4w2p", "testmain-3n1u4w2p", "testmain-2n0u3w2p", "testmain-3n0u3w2p", "testmain-2n1u3w2p", "testmain-3n1u3w2p"},
}

// Copyright 2026 Alan Tracey Wootton
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
