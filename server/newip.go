package main

import (
	"errors"
	"net"

	"github.com/bradfitz/gomemcache/memcache"
	"gopkg.in/mgo.v2/bson"
)

var UnknowRequestError = errors.New("Unknow Request")

type AppContext struct {
	LocalAddr    net.IP
	MemcachedUrl string
}

type UDPPacket struct {
	Payload    []byte
	RemoteAddr net.UDPAddr
}

type IteRequest struct {
	Request int
}

type IteResponse struct {
	NewIP     string
	ServerIP  string
	ConfirmID string
}

func NewIP(appCtx AppContext, queue chan UDPPacket, resultC chan []byte, errorC chan error) {
	udpPacket := <-queue
	request := IteRequest{}
	err := bson.Unmarshal(udpPacket.Payload, &request)
	if err != nil {
		errorC <- err
		return
	}

	if request.Request != 1 {
		errorC <- UnknowRequestError
		return
	}

	mainIface, err := net.InterfaceByName("")
	if err != nil {
		errorC <- err
		return
	}

	//ip, confirmID, err := NextIPEtcd()
	//ip, confirmID, err := NextIPMemcache(ctx)

	resultC <- []byte("")

	//Wenn etcd cluster verfügbar
	//Nicht erzeug aus eigener IP neue IP und speicher in memcached
	//Erzeuge Response
}

func NextIPMemcache(appCtx AppContext) (net.IP, string, error) {

}
