package main

import (
	"net"
	"testing"

	"gopkg.in/mgo.v2/bson"
)

var TestIP = "192.168.0.1"
var TestMemcachedUrl = "127.0.0.1"

func Test_NewIP_WithoutEtcdNOMemecacheEntries_OK(t *testing.T) {

	appCtx := AppContext{
		LocalAddr:    net.ParseIP(TestIP),
		MemcachedUrl: TestMemcachedUrl,
	}

	inputC := make(chan UDPPacket, 1)
	outputC := make(chan []byte)
	errorC := make(chan error)

	req := IteRequest{
		Request: 1,
	}

	payload, err := bson.Marshal(req)

	if err != nil {
		t.Fatal(err.Error())
	}

	udpPacket := UDPPacket{
		Payload: payload,
	}

	inputC <- udpPacket

	go NewIP(appCtx, inputC, outputC, errorC)

	select {
	case result := <-outputC:
		res := IteResponse{}
		err := bson.Unmarshal(result, &res)
		if err != nil {
			t.Fatal(err.Error())
		}
		if (res.NewIP == "") &&
			(res.ConfirmID == "") &&
			(res.ServerIP == "") {
			t.Fatal("Error in Response", res)
		}
	case err := <-errorC:
		t.Fatal(err.Error())

	}
}

// Test, es ext. kein etcd-Cluster und die der Memcache-DB ist noch kein Eintrag vorhanden
func Test_NextIPMemcache_NoEntriesInDB_OK(t *testing.T) {
	appCtx := AppContext{
		LocalAddr:    net.ParseIP(TestIP),
		MemcachedUrl: TestMemcachedUrl,
	}

	ip, confirmID, err := NextIPMemcache(appCtx)

	if err != nil {
		t.Fatal(err.Error())
	}

	expectIP := "192.168.0.2"
	if ip != net.ParseIP(expectIP) {
		t.Fatal("Expect", expectIP, ", was", ip)
	}
	if confirmID == "" {
		t.Fatal("No confirmID")
	}
}
