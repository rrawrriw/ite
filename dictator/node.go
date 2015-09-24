package dictator

import (
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"math/big"
	"time"

	"gopkg.in/mgo.v2/bson"
)

type (
	DictatorPayload struct {
		Type       int
		Blob       []byte
		DictatorID string
	}

	NodeContext struct {
		NodeID          string
		BecomeDictator  *time.Timer
		SuicideChan     chan struct{}
		AppContext      Context
		UDPIn           chan UDPPacket
		UDPOut          chan UDPPacket
		Mission         MissionSpecs
		IsDictatorAlive bool
	}
)

func NewNodeID() (string, error) {
	buf := make([]byte, 1000)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	k := sha1.Sum(buf)
	s := fmt.Sprintf("%x", k)
	return s, nil
}

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

	if 1 > pd.Type || pd.Type > 4 {
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
	if payload.Type == 2 {
		return true
	}

	return false
}

func IsCommandResponse(payload DictatorPayload) bool {
	if payload.Type == 3 {
		return true
	}

	return false
}

func IsHeartbeat(payload DictatorPayload) bool {
	if payload.Type == 1 {
		return true
	}

	return false
}

func StatusMsg(nodeID string, msg interface{}) string {
	return fmt.Sprintf("%v - %v", nodeID, msg)
}

func (nodeCtx NodeContext) HandlePacket(packet UDPPacket) error {
	l := nodeCtx.AppContext.Log
	if IsDictatorPayload(packet) {
		payload, err := ReadDictatorPayload(packet)
		if err != nil {
			return err
		}

		// Ignore myself except it is a CommandResposne
		if IsThatMe(nodeCtx.NodeID, payload) {
			if !IsCommandResponse(payload) {
				return nil
			}
		}

		// Make the command of the great dictator
		if IsCommand(payload) {
			// When a other dictator take over we have to die
			// and become a slave
			if nodeCtx.IsDictatorAlive {
				nodeCtx.SuicideChan <- struct{}{}
			}

			r := nodeCtx.Mission.CommandRouter
			blob := CommandBlob{}
			err := bson.Unmarshal(payload.Blob, &blob)
			if err != nil {
				return err
			}

			fun, ok := r.FindHandler(blob.Name)
			if !ok {
				errMsg := StatusMsg(nodeCtx.NodeID, "Cannot find CommandHandler")
				return errors.New(errMsg)
			}
			return fun(nodeCtx, payload)
		}

		// Just care about CommandResponse of my commands
		if IsCommandResponse(payload) {
			if IsThatMe(nodeCtx.NodeID, payload) {
				debugMsg := StatusMsg(nodeCtx.NodeID, "Receive command response")
				l.Debug.Println(debugMsg)
				nodeCtx.Mission.ResponseChan <- payload
				return nil
			}

			errMsg := StatusMsg(nodeCtx.NodeID, "Not my Command")
			return errors.New(errMsg)
		}

		// Reset heartbeat timeout
		if IsHeartbeat(payload) {
			// When a other dictator take over we have to die
			// and become a slave
			if nodeCtx.IsDictatorAlive {
				nodeCtx.SuicideChan <- struct{}{}
			}

			timeout, err := NewRandomTimeout(500, 1500)
			if err != nil {
				return err
			}
			nodeCtx.BecomeDictator.Reset(timeout)
			return nil
		}

	}

	return nil
}

func (nodeCtx NodeContext) LoopNode() {
	l := nodeCtx.AppContext.Log
	var err error
	var dictatorIsDead <-chan struct{}

	for {
		select {
		case <-nodeCtx.AppContext.DoneChan:
			debugMsg := StatusMsg(nodeCtx.NodeID, "Goodbye")
			l.Debug.Println(debugMsg)
			nodeCtx.SuicideChan <- struct{}{}
			return
		case packet := <-nodeCtx.UDPIn:
			//l.Debug.Println("Receive UDP packet", nodeCtx.NodeID)
			err = nodeCtx.HandlePacket(packet)
			if err != nil {
				errMsg := StatusMsg(nodeCtx.NodeID, err)
				l.Error.Println(errMsg)
			}
		case <-nodeCtx.BecomeDictator.C:
			nodeCtx.BecomeDictator.Stop()
			dictatorIsDead, err = nodeCtx.AwakeDictator()
			nodeCtx.IsDictatorAlive = true
			if err != nil {
				errMsg := StatusMsg(nodeCtx.NodeID, err)
				l.Error.Println(errMsg)
				return
			}
		case <-dictatorIsDead:
			nodeCtx.IsDictatorAlive = false

		}
	}
}

func (nodeCtx NodeContext) AwakeDictator() (<-chan struct{}, error) {
	l := nodeCtx.AppContext.Log
	debugMsg := StatusMsg(nodeCtx.NodeID, "Time to enslave some people")
	l.Debug.Println(debugMsg)

	nodeID := nodeCtx.NodeID

	dictatorIsDead := make(chan struct{})

	timeout, err := NewRandomTimeout(100, 150)
	if err != nil {
		return nil, err
	}
	dictatorHeartbeat := time.NewTicker(timeout)

	// Run mission impossible
	nodeCtx.Mission.Mission(nodeCtx)

	go func() {
		for {
			select {
			case <-nodeCtx.AppContext.DoneChan:
				errMsg := StatusMsg(nodeID, "The world shutdown")
				l.Debug.Println(errMsg)
				dictatorHeartbeat.Stop()
				return
			case <-nodeCtx.SuicideChan:
				errMsg := StatusMsg(nodeID, "Dictator must die")
				l.Debug.Println(errMsg)
				dictatorHeartbeat.Stop()
				dictatorIsDead <- struct{}{}
				return
			case <-dictatorHeartbeat.C:
				// It's time to say hello to the people
				// due to they don't forget us
				heartbeatPacket := DictatorPayload{
					Type:       1,
					DictatorID: nodeID,
					Blob:       []byte{},
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

	return dictatorIsDead, nil

}

func Node(ctx Context, udpIn, udpOut chan UDPPacket, missionSpecs MissionSpecs) error {

	nodeID, err := NewNodeID()
	if err != nil {
		return err
	}

	go func() {

		// First wait if there already a dictator
		timeout, err := NewRandomTimeout(500, 1500)
		if err != nil {
			ctx.Log.Error.Println(err.Error())
			return
		}

		nodeCtx := NodeContext{
			NodeID:          nodeID,
			SuicideChan:     make(chan struct{}),
			BecomeDictator:  time.NewTimer(timeout),
			UDPIn:           udpIn,
			UDPOut:          udpOut,
			AppContext:      ctx,
			Mission:         missionSpecs,
			IsDictatorAlive: false,
		}

		nodeCtx.LoopNode()

	}()

	return nil
}
