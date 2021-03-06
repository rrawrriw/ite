package dictator

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"gopkg.in/mgo.v2/bson"
)

func MakeTestLogger() Logger {
	errorWriter := os.Stderr
	debugWriter := os.Stdout

	l := Logger{
		Error: log.New(errorWriter, "Error: ", log.LstdFlags|log.Lshortfile),
		Debug: log.New(debugWriter, "Debug: ", log.LstdFlags|log.Lshortfile),
	}

	return l
}

func existsID(ids []string, id string) bool {
	for _, i := range ids {
		if i == id {
			return true
		}
	}

	return false
}

func Test_NewNodeID_OK(t *testing.T) {
	max := 100
	ids := []string{}
	for x := 0; x < max; x++ {
		id, err := NewNodeID()
		if err != nil {
			t.Fatal(err)
		}
		if existsID(ids, id) {
			t.Fatal("Error double id found")
		}

		ids = append(ids, id)
	}
}

func Test_AwakeDictator(t *testing.T) {

	time.Sleep(1 * time.Second)

	nodeListenAddr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 12345,
	}

	testerListenAddr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 12345,
	}

	connIn, err := net.ListenUDP("udp", &nodeListenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	connOut, err := net.DialUDP("udp", nil, &testerListenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}

	conns := []*net.UDPConn{
		connIn,
		connOut,
	}
	ctx := NewContextWithConn(conns)
	defer ctx.Done()

	udpIn, err := UDPInbox(ctx, connIn)
	if err != nil {
		t.Fatal(err.Error())
	}

	udpOut, err := UDPOutbox(ctx, connOut)
	if err != nil {
		t.Fatal(err.Error())
	}

	testResultC := make(chan error)
	timeout := time.After(2 * time.Second)

	killDictatorC := make(chan struct{})

	mission := MissionSpecs{
		Mission: func(NodeContext) {
			go func() {
			}()
		},
	}

	dictatorID := "1234"
	nodeCtx := NodeContext{
		AppContext:  ctx,
		NodeID:      dictatorID,
		UDPOut:      udpOut,
		SuicideChan: killDictatorC,
		Mission:     mission,
	}
	nodeCtx.AwakeDictator()

	// Receive DictatorHearbeate
	go func() {
		select {
		case <-timeout:
			testResultC <- errors.New("Test runs out of time")
		case packet := <-udpIn:
			p := DictatorPayload{}
			err := bson.Unmarshal(packet.Payload, &p)
			if err != nil {
				testResultC <- err
			}
			if p.Type != 1 {
				testResultC <- errors.New("Wrong message type")
			}

			if p.DictatorID != dictatorID {
				errMsg := fmt.Sprintf("Expect %v was %v", dictatorID, p.DictatorID)
				testResultC <- errors.New(errMsg)

			}
			testResultC <- nil
		}
	}()

	err = <-testResultC
	if err != nil {
		t.Fatal(err.Error())
	}

	time.Sleep(1 * time.Second)

}

// Wird der Diktator aufgeweckt start seine Mission.
// Dieser Test prüft ob die übergeben Mission gestart wird
func Test_ExecDictatorCommand_OK(t *testing.T) {

	testResult := make(chan error)

	// DictatorMission
	mission := func(nCtx NodeContext) {
		// Muss goroutine starten damit weiterhin heartbeats
		// versendet werden. Ebenso sollte appContext.DoneChan
		// beachtet werden. Sowie SuicideChan.
		go func() {
			// Beende Dictator goroutine
			defer func() {
				nCtx.SuicideChan <- struct{}{}
			}()
			testResult <- nil
		}()
	}

	specs := MissionSpecs{
		Mission: mission,
	}

	suicide := make(chan struct{})
	nodeCtx := NodeContext{
		AppContext:  NewContext(),
		UDPOut:      make(chan UDPPacket),
		SuicideChan: suicide,
		Mission:     specs,
	}

	nodeCtx.AwakeDictator()

	err := <-testResult
	if err != nil {
		t.Fatal(err.Error())
	}

}

// Erhält ein Sklave ein Kommando von einem Diktator muss überprüft werden
// ob dieses bekannt ist und draufhin ausgeführt werden
func Test_ExecSlaveCommand_OK(t *testing.T) {

	testOK := false

	h := func(nCtx NodeContext, p DictatorPayload) error {
		testOK = true
		return nil
	}

	cmdRouter := CommandRouter{}
	cmdRouter.AddHandler("test", h)

	specs := MissionSpecs{
		CommandRouter: cmdRouter,
	}

	suicide := make(chan struct{})
	nodeCtx := NodeContext{
		NodeID:      "1",
		AppContext:  NewContext(),
		UDPOut:      make(chan UDPPacket),
		SuicideChan: suicide,
		Mission:     specs,
	}

	cmdBlob := CommandBlob{
		Name: "test",
	}

	blob, err := bson.Marshal(cmdBlob)
	if err != nil {
		t.Fatal(err.Error())
	}

	cmdPayload := DictatorPayload{
		Type: 2,
		Blob: blob,
	}

	payload, err := bson.Marshal(cmdPayload)
	if err != nil {
		t.Fatal(err.Error())
	}

	packet := UDPPacket{
		Payload: payload,
	}

	err = nodeCtx.HandlePacket(packet)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !testOK {
		t.Fatal("Expect to be true")
	}
}

// Teste Reaktion auf CommandResponse Diktator seitig.
func Test_ExecCommandOfResponse_OK(t *testing.T) {
	testResult := make(chan error)
	responseChan := make(chan DictatorPayload)

	m := func(c NodeContext) {
		go func() {
			select {
			case <-responseChan:
				testResult <- nil
			}
		}()
	}

	specs := MissionSpecs{
		Mission:      m,
		ResponseChan: responseChan,
	}

	suicide := make(chan struct{})

	nodeCtx := NodeContext{
		NodeID:      "1",
		AppContext:  NewContext(),
		UDPOut:      make(chan UDPPacket),
		SuicideChan: suicide,
		Mission:     specs,
	}

	commandResponseBlob := CommandResponseBlob{
		NodeID: "1234",
		Status: 1,
	}
	blob, err := bson.Marshal(commandResponseBlob)
	if err != nil {
		t.Fatal(err.Error())
	}

	commandResponse := DictatorPayload{
		Type:       3,
		DictatorID: "1",
		Blob:       blob,
	}
	payload, err := bson.Marshal(commandResponse)
	if err != nil {
		t.Fatal(err.Error())
	}
	packet := UDPPacket{
		Payload: payload,
	}

	m(nodeCtx)

	err = nodeCtx.HandlePacket(packet)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = <-testResult
	if err != nil {
		t.Fatal(err.Error())
	}
}

// Dieser Test prüft ob eine gestartet Node welche nach einem Timeout
// zu einem Diktator wurde diesen Status wieder abgibt und ebenso aufhört
// Heartbeats zu senden nachdem eine andere Node einen Heartbeat gesendet
// hat. Es gilt zu beachten da alles Nebenläufig statt findet so kann es
// vorkommen das der Tester weiter Heartbeats empfängt nachdem dieser
// den gefakten Heartbeat gesendet hat. Dieses verhalten muss beim
// Testen beachtet werden. Es gibt dazu aber einen weiteren Hinweis
// im Quellcode.
func Test_NodeBecomeSlaveAfterReceivedDictatorHeartbeat(t *testing.T) {

	time.Sleep(1 * time.Second)

	nodeID := "123"
	testResultC := make(chan error)
	nodeListenAddr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 12345,
	}
	nodeListenConn, err := net.ListenUDP("udp", &nodeListenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}

	nodeSendAddr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 12346,
	}
	nodeSendConn, err := net.DialUDP("udp", nil, &nodeSendAddr)
	if err != nil {
		t.Fatal(err.Error())
	}

	testerListenAddr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 12346,
	}
	testerListenConn, err := net.ListenUDP("udp", &testerListenAddr)
	if err != nil {
		t.Fatal(err.Error())
	}

	testerSendAddr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 12345,
	}
	testerSendConn, err := net.DialUDP("udp", nil, &testerSendAddr)
	if err != nil {
		t.Fatal(err.Error())
	}

	conns := []*net.UDPConn{
		nodeListenConn,
		nodeSendConn,
		testerListenConn,
		testerSendConn,
	}
	ctx := NewContextWithConn(conns)
	defer ctx.Done()

	nodeUDPIn, err := UDPInbox(ctx, nodeListenConn)
	if err != nil {
		t.Fatal(err.Error())
	}

	nodeUDPOut, err := UDPOutbox(ctx, nodeSendConn)
	if err != nil {
		t.Fatal(err.Error())
	}

	testerUDPIn, err := UDPInbox(ctx, testerListenConn)
	if err != nil {
		t.Fatal(err.Error())
	}

	testerUDPOut, err := UDPOutbox(ctx, testerSendConn)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Test timeout
	// Läuft die Zeit für den Test ab und es wurde kein Fehler ausgelöst
	// wurde der Test bestanden.
	time.AfterFunc(
		5*time.Second,
		func() {
			testResultC <- nil
		},
	)

	killDictator := make(chan struct{})

	missionSpecs := MissionSpecs{
		Mission: func(NodeContext) {
			go func() {
			}()
		},
	}

	// Der erster Test-Schritt, es wird gewartet bis die Node Heartbeats
	// sendet. Daraufhin werden vom Tester ebenfalls Heartbeats gesendet
	// Daraufhin dürfen von der Node keine weitern Heartbeats kommen.
	// Zu beachten gilt was in der Einführung über nebenläufigkeit gesagt
	// wurde
	go func() {
		for {
			select {
			case packet := <-testerUDPIn:
				p := DictatorPayload{}
				err := bson.Unmarshal(packet.Payload, &p)
				if err != nil {
					testResultC <- err
				}
				if p.Type != 1 {
					testResultC <- errors.New("Wrong message type")
				}

				// Starte mit senden von Heartbeats
				nodeCtx := NodeContext{
					AppContext:  ctx,
					NodeID:      nodeID,
					UDPOut:      testerUDPOut,
					Mission:     missionSpecs,
					SuicideChan: killDictator,
				}
				nodeCtx.AwakeDictator()
				// Starte Schritt 2 test Tests
				// warte ob mehr als 3 weiter Heartbeats
				// von der Node gesendet werden.
				go func() {
					for x := 1; ; x++ {
						select {
						case <-testerUDPIn:
							if x > 3 {
								testResultC <- errors.New("Expect to receive no further heartbeats")
							}

						}
					}
				}()

				// Beende den ersten Schritt des Tests
				// damit dieser keine weiteren UDP Pakete
				// empfängt da ansont goroutines des 2 Schritts
				// mehrfach ausgeführt werden.
				return
			}
		}
	}()

	Node(ctx, nodeUDPIn, nodeUDPOut, missionSpecs)

	err = <-testResultC
	if err != nil {
		t.Fatal(err.Error())
	}

	time.Sleep(1 * time.Second)

}

func Test_ReadDictatorPayload_OK(t *testing.T) {
	id := "1"

	payload := DictatorPayload{
		DictatorID: id,
	}

	dp, err := bson.Marshal(payload)
	if err != nil {
		t.Fatal(err.Error())
	}

	packet := UDPPacket{
		Payload: dp,
	}

	p, err := ReadDictatorPayload(packet)
	if err != nil {
		t.Fatal(err.Error())
	}

	if p.DictatorID != id {
		t.Fatal("Expect", id, "was", p.DictatorID)
	}
}

func Test_ReadDictatorPayload_Fail(t *testing.T) {
	packet := UDPPacket{
		Payload: []byte(""),
	}

	_, err := ReadDictatorPayload(packet)
	if err == nil {
		t.Fatal("Expect func to return a error")
	}
}

func Test_IsDictatorPayload_OK(t *testing.T) {
	payload, err := bson.Marshal(
		DictatorPayload{
			Type: 1,
		},
	)
	if err != nil {
		t.Fatal(err.Error())
	}
	packet := UDPPacket{
		Payload: payload,
	}
	if !IsDictatorPayload(packet) {
		t.Fatal("Expect to be true")
	}
}

func Test_IsDictatorPayload_Fail(t *testing.T) {
	packet := UDPPacket{}
	if IsDictatorPayload(packet) {
		t.Fatal("Expect to be false")
	}
}

func Test_IsDictatorPayload_Fail2(t *testing.T) {
	payload, err := bson.Marshal(
		DictatorPayload{
			Type: 5,
		},
	)
	if err != nil {
		t.Fatal(err.Error())
	}
	packet := UDPPacket{
		Payload: payload,
	}
	if IsDictatorPayload(packet) {
		t.Fatal("Expect to be false")
	}
}

func Test_IsDictatorPayload_Fail3(t *testing.T) {
	payload, err := bson.Marshal(
		DictatorPayload{
			Type: 0,
		},
	)
	if err != nil {
		t.Fatal(err.Error())
	}
	packet := UDPPacket{
		Payload: payload,
	}
	if IsDictatorPayload(packet) {
		t.Fatal("Expect to be false")
	}
}

func Test_IsThatMe_OK(t *testing.T) {
	id := "1"
	payload := DictatorPayload{
		DictatorID: id,
	}
	if !IsThatMe(id, payload) {
		t.Fatal("Expect to be me")
	}
}

func Test_IsThatMe_Fail(t *testing.T) {
	id := "1"
	payload := DictatorPayload{
		DictatorID: "2",
	}
	if IsThatMe(id, payload) {
		t.Fatal("Expect not myself")
	}
}

func Test_IsThatMe_Fail2(t *testing.T) {
	id := "1"
	payload := DictatorPayload{}
	if IsThatMe(id, payload) {
		t.Fatal("Expect not myself")
	}
}

func Test_IsHeartbeat_OK(t *testing.T) {
	payload := DictatorPayload{
		Type: 1,
	}

	if !IsHeartbeat(payload) {
		t.Fatal("Expect to be heartbeat")
	}
}

func Test_IsHeartbeat_Fail(t *testing.T) {
	payload := DictatorPayload{
		Type: 2,
	}

	if IsHeartbeat(payload) {
		t.Fatal("Expect not to be a heartbeat")
	}
}

func Test_IsCommand_OK(t *testing.T) {
	payload := DictatorPayload{
		Type: 2,
	}

	if !IsCommand(payload) {
		t.Fatal("Expect to be a command")
	}
}

func Test_IsCommand_Fail(t *testing.T) {
	payload := DictatorPayload{
		Type: 3,
	}

	if IsCommand(payload) {
		t.Fatal("Expect not to be a command")
	}
}

func Test_IsCommandResponse_OK(t *testing.T) {
	payload := DictatorPayload{
		Type: 3,
	}

	if !IsCommandResponse(payload) {
		t.Fatal("Expect to be a command response")
	}
}

func Test_IsCommandResponse_Fail(t *testing.T) {
	payload := DictatorPayload{
		Type: 4,
	}

	if IsCommandResponse(payload) {
		t.Fatal("Expect not to be a command response")
	}
}
