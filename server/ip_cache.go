package main

import (
	"net"
	"time"
)

type IPItemResponse struct {
	IP        net.IP
	ConfirmID string
	Err       error
}

type IPItem struct {
	IP     net.IP
	Expire time.Time
}

// Netmask 255.255.255.0 es wird nichts akzeptiert wie 255.0.255.255.255
type SubnetSpec struct {
	Sub  int
	From net.IP
	To   net.IP
}

type IPCacheRequest struct {
	Request    int
	ResultChan chan IPItemResponse
	ErrorChan  chan error
}

type IPCacheC chan IPCacheRequest

type IPCache map[string]IPItem

// Verwaltet IPv4 Adressen
// ttl in Sekunden
func NewIPCache(subnet SubnetSpec, init []IPItem, ttl int) (IPCacheC, error) {
	ipCacheC := make(chan IPCacheRequest)
	ipCache := IPCache{}
	err := InitIPCache(init, &ipCache)
	if err != nil {
		return nil, err
	}
	go func() {
		for req := range ipCacheC {
			err := CleanIPCache(&ipCache)
			if err != nil {
				req.ErrorChan <- err
				continue
			}
			switch req.Request {
			// Next IP
			case 1:
			// Read IP by confirm id
			case 2:

			}
		}

	}()

	return make(IPCacheC), nil

}

func InitIPCache(initIP []IPItem, cache *IPCache) error {
	return nil
}

// Entfernet alle Abgelaufen Einträge
func CleanIPCache(cache *IPCache) error {
	return nil
}

func (c IPCacheC) NextIP(resultC chan IPItemResponse) <-chan error {
	errorC := make(chan error)
	c <- IPCacheRequest{
		Request:    1,
		ResultChan: resultC,
		ErrorChan:  errorC,
	}

	return errorC
}
