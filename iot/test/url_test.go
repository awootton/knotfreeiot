package iot_test

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getVal(t *testing.T, url string) string {
	resp, err := http.Get(url)
	assert.Nil(t, err)
	if err != nil {
		return ""
	}
	assert.Equal(t, resp.StatusCode, 200)
	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	if err != nil {
		return ""
	}
	return string(resBody)
}

func TestUrl(t *testing.T) {

	ce := makeClusterWithServiceContact()

	// note: the .com and .test tlds are in /etc/hosts
	{ // a regular api call
		val := getVal(t, "http://knotlocal.com:8085/api1/getPublicKey")
		fmt.Println("getPublicKey", val)
		sss := base64.RawURLEncoding.EncodeToString(ce.PublicKeyTemp[:])
		assert.Equal(t, val, sss) //"-muxcABH_pTsuNqT3yaYfQj-3krwM6XmEu47vTZLSHM")
	}
	startAServer("get-unix-time", "")     // start a thing server
	startAServer("get-unix-time_iot", "") // start a thing server
	{                                     // a device call
		val := getVal(t, "http://get-unix-time.knotlocal.com:8085/get/pubk")
		fmt.Println("pubk", val)
		assert.Equal(t, val, "bht-Ka3j7GKuMFOablMlQnABnBvBeugvSf4CdFV3LXs")
	}
	{ // a device call
		val := getVal(t, "http://get-unix-time_iot.knotlocal.com:8085/get/pubk")
		fmt.Println("pubk", val)
		assert.Equal(t, val, "bht-Ka3j7GKuMFOablMlQnABnBvBeugvSf4CdFV3LXs")
	}

	{ // a device call with iot name like get-unix-time.iot
		val := getVal(t, "http://get-unix-time.test:8085/get/pubk") // note: the .com and .test tlds are in /etc/hosts
		fmt.Println("pubk", val)
		assert.Equal(t, val, "bht-Ka3j7GKuMFOablMlQnABnBvBeugvSf4CdFV3LXs")
	}

	{ // a device call with iot name like get-unix-time.iot
		val := getVal(t, "http://get.option.a.get-unix-time.test:8085") // note: the .com and .test tlds are in /etc/hosts
		fmt.Println("get.option.a", val)
		assert.Equal(t, val, "216.128.128.195")
	}

}
