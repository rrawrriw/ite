package dictator

import (
	"crypto/rand"
	"math/big"
	"net"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type (
	DictatorPayload struct {
		Type       int
		Blob       interface{}
		DictatorID string
	}
)

func NewTimeout(min, max int) (time.Duration, error) {
	minR := big.NewInt(int64(min))
	maxR := big.NewInt(int64(max - min))
	random, err := rand.Int(rand.Reader, maxR)
	if err != nil {
		return time.Duration(0), err
	}
	random = random.Add(random, minR)

	return time.Duration(random.Int64()) * time.Millisecond, nil

}

func Node(doneC <-chan struct{}, readAddr, writeAddr net.UDPAddr, l Logger) {

	connIn, err := net.ListenUDP("udp", &readAddr)
	if err != nil {
		l.Error.Println(err.Error())
		return
	}

	connOut, err := net.DialUDP("udp", nil, &writeAddr)
	if err != nil {
		l.Error.Println(err.Error())
		return
	}

	udpPacketInC := make(chan UDPPacket)
	// Wait for UDP packages
	go func() {
		l.Debug.Println("Start UDP packet handler")
		defer connIn.Close()
		for {
			p := make([]byte, 1024)
			s, rAddr, err := connIn.ReadFromUDP(p)
			if err != nil {
				l.Error.Println(err.Error())
				continue
			}
			udpPacketInC <- UDPPacket{
				RemoteAddr: rAddr,
				Payload:    p,
				Size:       s,
			}

		}
	}()

	udpPacketOutC := make(chan UDPPacket)
	// Repeate UDP packages
	go func() {
		l.Debug.Println("Start UDP packets sender")
		defer connOut.Close()
		for udpPacket := range udpPacketOutC {
			b, err := bson.Marshal(udpPacket)
			if err != nil {
				l.Error.Println(err.Error())
				continue
			}

			_, err = connOut.WriteToUDP(b, &writeAddr)
			if err != nil {
				l.Error.Println(err.Error())
				continue
			}
		}
	}()

	// Wait for DictatorPacket
	go func() {
		nodeID := "1234"

		// The Cannel to stop the dictator heartbeat goroutine
		killDictatorC := make(chan bool)

		// First wait if there already a dictator
		timeout, err := NewTimeout(500, 1500)
		if err != nil {
			l.Error.Println(err.Error())
			return
		}
		deadDictator := time.NewTimer(timeout)

		for {
			select {
			case <-doneC:
				l.Debug.Println("Goodbye node")
				killDictatorC <- true
				return
			case <-udpPacketInC:
				l.Debug.Println("Receive UDP packet")
				deadDictator.Stop()
				timeout, err := NewTimeout(500, 1500)
				if err != nil {
					l.Error.Println(err.Error())
					return
				}
				deadDictator = time.NewTimer(timeout)
			case <-deadDictator.C:
				l.Debug.Println("Time to enslave some people")
				deadDictator.Stop()
				err := DictatorHeartbeat(
					nodeID,
					killDictatorC,
					udpPacketOutC,
					l,
				)
				if err != nil {
					l.Error.Println(err.Error())
					return
				}

			}
		}
	}()

}

func DictatorHeartbeat(nodeID string, doneC chan bool, outputC chan UDPPacket, l Logger) error {
	timeout, err := NewTimeout(100, 150)
	if err != nil {
		l.Error.Println(err.Error())
		return err
	}
	greatDictator := time.NewTicker(timeout)
	go func() {
		for {
			select {
			case <-doneC:
				l.Debug.Println("piiiiiiiip")
				greatDictator.Stop()
				return
			case <-greatDictator.C:
				heartbeat := DictatorPayload{
					Type:       1,
					DictatorID: nodeID,
					Blob:       0,
				}
				p, err := bson.Marshal(heartbeat)
				if err != nil {
					l.Error.Println(err.Error())
					continue
				}
				outputC <- UDPPacket{
					Payload: p,
				}
			}
		}
	}()

	return nil

}
