package dictator

import (
	"crypto/rand"
	"log"
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

	UDPPacket struct {
		RemoteAddr *net.UDPAddr
		Payload    []byte
		Size       int
	}

	Logger struct {
		Error *log.Logger
		Debug *log.Logger
	}
)

func NewHeartbeatTimeout(min, max int) (time.Duration, error) {
	minR := big.NewInt(int64(min))
	maxR := big.NewInt(int64(max - min))
	random, err := rand.Int(rand.Reader, maxR)
	if err != nil {
		return time.Duration(0), err
	}
	random = random.Add(random, minR)

	return time.Duration(random.Int64()) * time.Millisecond, nil

}

func Node(readAddr, writeAddr net.UDPAddr, l Logger) {

	conn, err := net.ListenUDP("udp", &readAddr)
	if err != nil {
		l.Error.Fatalln(err.Error())
	}

	udpPacketC := make(chan UDPPacket)
	// Wait for udp packages
	go func() {
		for {
			p := make([]byte, 1024)
			s, rAddr, err := conn.ReadFromUDP(p)
			if err != nil {
				l.Error.Println(err.Error())
				continue
			}
			udpPacketC <- UDPPacket{
				RemoteAddr: rAddr,
				Payload:    p,
				Size:       s,
			}

		}
	}()

	// Wait for DictatorPacket
	go func() {
		dictatorID := "1234"
		timeout, err := NewHeartbeatTimeout(500, 1500)
		if err != nil {
			l.Error.Println(err.Error())
			return
		}
		deadDictator := time.NewTimer(timeout)
		for {
			select {
			case <-udpPacketC:
				l.Debug.Println("Receive UDP packet")
				deadDictator.Stop()
				timeout, err := NewHeartbeatTimeout(500, 1500)
				if err != nil {
					l.Error.Println(err.Error())
					return
				}
				deadDictator = time.NewTimer(timeout)
			case <-deadDictator.C:
				l.Debug.Println("Try to become a dictator")
				heartbeat := DictatorPayload{
					Type:       1,
					DictatorID: dictatorID,
					Blob:       0,
				}
				p, err := bson.Marshal(heartbeat)
				if err != nil {
					l.Error.Println(err.Error())
					continue
				}
				_, err = conn.WriteToUDP(p, &writeAddr)
				if err != nil {
					l.Error.Println(err.Error())
					continue
				}
			}
		}
	}()

}
