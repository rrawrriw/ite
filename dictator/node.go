package dictator

import (
	"crypto/rand"
	"fmt"
	"math/big"
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

func Node(ctx Context, udpIn, udpOut chan UDPPacket) {

	// Wait for DictatorPacket
	go func() {
		nodeID := "1234"

		// The Cannel to stop the dictator heartbeat goroutine
		killDictatorC := make(chan struct{})

		// First wait if there already a dictator
		timeout, err := NewTimeout(500, 1500)
		if err != nil {
			ctx.Log.Error.Println(err.Error())
			return
		}
		fmt.Println("Timeout:", timeout)
		deadDictator := time.NewTimer(timeout)

		for {
			select {
			case <-ctx.DoneChan:
				ctx.Log.Debug.Println("Goodbye node", nodeID)
				killDictatorC <- struct{}{}
				return
			case <-udpIn:
				ctx.Log.Debug.Println("Receive UDP packet")
				deadDictator.Stop()
				killDictatorC <- struct{}{}
				timeout, err := NewTimeout(500, 1500)
				if err != nil {
					ctx.Log.Error.Println(err.Error())
					return
				}
				deadDictator = time.NewTimer(timeout)
			case <-deadDictator.C:
				ctx.Log.Debug.Println("Time to enslave some people", nodeID)
				deadDictator.Stop()
				err := AwakeDictator(
					ctx,
					nodeID,
					udpOut,
					killDictatorC,
				)
				if err != nil {
					ctx.Log.Error.Println(err.Error())
					return
				}

			}
		}
	}()

}

func AwakeDictator(ctx Context, nodeID string, udpOut chan UDPPacket, killDictatorC chan struct{}) error {
	ctx.Log.Debug.Println("Long live the dictator", nodeID)
	timeout, err := NewTimeout(100, 150)
	if err != nil {
		ctx.Log.Error.Println(err.Error())
		return err
	}
	dictatorHeartbeat := time.NewTicker(timeout)
	go func() {
		for {
			select {
			case <-ctx.DoneChan:
				ctx.Log.Debug.Println("The world shutdown", nodeID)
				dictatorHeartbeat.Stop()
				return
			case <-killDictatorC:
				ctx.Log.Debug.Println("Dictator must die", nodeID)
				dictatorHeartbeat.Stop()
				return
			case <-dictatorHeartbeat.C:
				heartbeatPacket := DictatorPayload{
					Type:       1,
					DictatorID: nodeID,
					Blob:       0,
				}
				p, err := bson.Marshal(heartbeatPacket)
				if err != nil {
					ctx.Log.Error.Println(err.Error())
					continue
				}
				udpOut <- UDPPacket{
					Payload: p,
				}
			}
		}
	}()

	return nil

}
