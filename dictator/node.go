package dictator

import (
	"crypto/rand"
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

	NodeContext struct {
		NodeID         string
		BecomeDictator *time.Timer
		SuicideChan    chan struct{}
		AppContext     Context
		UDPIn          chan UDPPacket
		UDPOut         chan UDPPacket
	}
)

func NewRandomTimeout(min, max int) (time.Duration, error) {
	minR := big.NewInt(int64(min))
	maxR := big.NewInt(int64(max - min))
	random, err := rand.Int(rand.Reader, maxR)
	if err != nil {
		return time.Duration(0), err
	}
	random = random.Add(random, minR)

	return time.Duration(random.Int64()) * time.Millisecond, nil

}

func IsThatMe(packet UDPPacket) bool {
	return false
}

func IsCommand(packet UDPPacket) bool {
	return false
}

func IsHeartbeat(packet UDPPacket) bool {
	return true
}

func (nodeCtx NodeContext) HandlePacket(packet UDPPacket) error {
	l := nodeCtx.AppContext.Log
	// Ignore myself
	if IsThatMe(packet) {
		return nil
	}
	// Make the command of the great dictator
	if IsCommand(packet) {
		return nil
	}
	// Reset heartbeat timeout
	if IsHeartbeat(packet) {
		nodeCtx.BecomeDictator.Stop()
		timeout, err := NewRandomTimeout(500, 1500)
		if err != nil {
			l.Error.Println(err.Error())
			return nil
		}
		nodeCtx.BecomeDictator = time.NewTimer(timeout)
	}
	nodeCtx.SuicideChan <- struct{}{}

	return nil
}

func (nodeCtx NodeContext) LoopNode() {
	l := nodeCtx.AppContext.Log
	for {
		select {
		case <-nodeCtx.AppContext.DoneChan:
			l.Debug.Println("Goodbye node", nodeCtx.NodeID)
			nodeCtx.SuicideChan <- struct{}{}
			return
		case packet := <-nodeCtx.UDPIn:
			l.Debug.Println("Receive UDP packet")
			nodeCtx.HandlePacket(packet)
		case <-nodeCtx.BecomeDictator.C:
			l.Debug.Println("Time to enslave some people", nodeCtx.NodeID)
			nodeCtx.BecomeDictator.Stop()
			err := nodeCtx.AwakeDictator()
			if err != nil {
				l.Error.Println(err.Error())
				return
			}

		}
	}
}

func (nodeCtx NodeContext) AwakeDictator() error {
	l := nodeCtx.AppContext.Log
	nodeID := nodeCtx.NodeID

	l.Debug.Println("Long live the dictator", nodeID)

	timeout, err := NewRandomTimeout(100, 150)
	if err != nil {
		l.Error.Println(err.Error())
		return err
	}
	dictatorHeartbeat := time.NewTicker(timeout)
	go func() {
		for {
			select {
			case <-nodeCtx.AppContext.DoneChan:
				l.Debug.Println("The world shutdown", nodeID)
				dictatorHeartbeat.Stop()
				return
			case <-nodeCtx.SuicideChan:
				l.Debug.Println("Dictator must die", nodeID)
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
					l.Error.Println(err.Error())
					continue
				}
				nodeCtx.UDPOut <- UDPPacket{
					Payload: p,
				}
			}
		}
	}()

	return nil

}

func Node(ctx Context, udpIn, udpOut chan UDPPacket) {

	// Wait for DictatorPacket
	go func() {

		// First wait if there already a dictator
		timeout, err := NewRandomTimeout(500, 1500)
		if err != nil {
			ctx.Log.Error.Println(err.Error())
			return
		}

		nodeCtx := NodeContext{
			NodeID:         "1234",
			SuicideChan:    make(chan struct{}),
			BecomeDictator: time.NewTimer(timeout),
			UDPIn:          udpIn,
			UDPOut:         udpOut,
			AppContext:     ctx,
		}

		nodeCtx.LoopNode()

	}()

}
