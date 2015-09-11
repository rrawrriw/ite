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

func ReadDictatorPayload(packet UDPPacket) (DictatorPayload, error) {
	payload := DictatorPayload{}
	err := bson.Unmarshal(packet.Payload, &payload)
	if err != nil {
		return DictatorPayload{}, err
	}

	return payload, nil
}

func IsDictatorPayload(packet UDPPacket) bool {
	pd, err := ReadDictatorPayload(packet)
	if err != nil {
		return false
	}

	if pd.DictatorID == "" && pd.Type == 0 {
		return false
	}

	return true
}

func IsThatMe(id string, payload DictatorPayload) bool {
	if id == payload.DictatorID {
		return true
	}

	return false
}

func IsCommand(payload DictatorPayload) bool {
	return false
}

func IsHeartbeat(payload DictatorPayload) bool {
	if payload.Type == 1 {
		return true
	}

	return false
}

func (nodeCtx NodeContext) HandlePacket(packet UDPPacket) error {
	l := nodeCtx.AppContext.Log
	if IsDictatorPayload(packet) {
		payload, err := ReadDictatorPayload(packet)
		if err != nil {
			return err
		}

		// Ignore myself
		if IsThatMe(nodeCtx.NodeID, payload) {
			return nil
		}
		// Make the command of the great dictator
		if IsCommand(payload) {
			return nil
		}
		// Reset heartbeat timeout
		if IsHeartbeat(payload) {
			timeout, err := NewRandomTimeout(500, 1500)
			if err != nil {
				l.Error.Println(err.Error())
				return nil
			}
			nodeCtx.BecomeDictator.Reset(timeout)
		}

		nodeCtx.SuicideChan <- struct{}{}
	}

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
			l.Debug.Println("Receive UDP packet", nodeCtx.NodeID)
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
