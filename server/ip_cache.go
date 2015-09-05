package main

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
	ResultChan chan IPItem
}

type IPCacheC chan IPItem

type IPCache map[string]IpItem

// Verwaltet IPv4 Adressen
// 192.168.2.5/24 dann werden die folgenden IP Adressen verwaltet
// 192.168.2.1 - 192.168.2.255 es ist egal ob eine 5 als letzte
// ttl in Sekunden
func NewIPCache(subnet SubnetSpec, init []net.IP, ttl int) {
	ipCacheC := make(chan IPCacheRequest)
	ipCache := map[string]string{}
	err := initIPCache(init, &ipCache)
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

}

// Entfernet alle Abgelaufen Einträge
func CleanIPCache(cache *IPCache) {
}

func (c IPCacheC) NextIP(resultC chan IPItem) <-chan error {
	errorC := make(chan error)
	c <- IPCacheRequest{
		Request:    1,
		ResultChan: resultC,
		ErrorChan:  errorC,
	}

	return errorC
}
