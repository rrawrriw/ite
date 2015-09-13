package dictator

import (
	"crypto/rand"
	"errors"
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

	// Gibt vor welchen Befehl die Nodes ausfürhen sollen
	CommandBlob struct {
		Name string
	}

	// Gibt zurück ob Befehl erfolgreich ausgführt wurde
	CommandResponseBlob struct {
		NodeID string
		Status int
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

	AliveChan struct {
		Q <-chan struct{}
		A chan bool
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
				return errors.New("Cannot find CommandHandler")
			}
			return fun(nodeCtx)
		}

		// Just care about CommandResponse of my commands
		if IsCommandResponse(payload) {
			if IsThatMe(nodeCtx.NodeID, payload) {
				nodeCtx.Mission.ResponseChan <- payload
				return nil
			}

			return errors.New("Not my Command")
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
				l.Error.Println(err.Error())
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
			l.Debug.Println("Goodbye node", nodeCtx.NodeID)
			nodeCtx.SuicideChan <- struct{}{}
			return
		case packet := <-nodeCtx.UDPIn:
			l.Debug.Println("Receive UDP packet", nodeCtx.NodeID)
			nodeCtx.HandlePacket(packet)
		case <-nodeCtx.BecomeDictator.C:
			l.Debug.Println("Time to enslave some people", nodeCtx.NodeID)
			nodeCtx.BecomeDictator.Stop()
			dictatorIsDead, err = nodeCtx.AwakeDictator()
			nodeCtx.IsDictatorAlive = true
			if err != nil {
				l.Error.Println(err.Error())
				return
			}
		case <-dictatorIsDead:
			nodeCtx.IsDictatorAlive = false

		}
	}
}

func (nodeCtx NodeContext) AwakeDictator() (<-chan struct{}, error) {
	l := nodeCtx.AppContext.Log
	nodeID := nodeCtx.NodeID

	dictatorIsDead := make(chan struct{})

	l.Debug.Println("Long live the dictator", nodeID)

	timeout, err := NewRandomTimeout(100, 150)
	if err != nil {
		l.Error.Println(err.Error())
		return nil, err
	}
	dictatorHeartbeat := time.NewTicker(timeout)

	// Run mission impossible
	nodeCtx.Mission.Mission(nodeCtx)

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

func Node(ctx Context, udpIn, udpOut chan UDPPacket, missionSpecs MissionSpecs) {

	// Wait for DictatorPacket
	go func() {

		// First wait if there already a dictator
		timeout, err := NewRandomTimeout(500, 1500)
		if err != nil {
			ctx.Log.Error.Println(err.Error())
			return
		}

		nodeCtx := NodeContext{
			NodeID:          "1234",
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

}
