package dictator

import (
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

func Test_NodeBecomeDictator(t *testing.T) {

	testResultC := make(chan bool)
	timeout := time.After(2 * time.Second)
	payloadC := make(chan []byte)

	// Receive DictatorHearbeate
	go func() {
		select {
		case <-timeout:
			testResultC <- false
		case payload := <-payloadC:
			p := DictatorPayload{}
			err := bson.Unmarshal(payload, &p)
			if err != nil {
				t.Fatal(err.Error())
			}
			if p.Type != 1 {
				testResultC <- false
			}
			testResultC <- true
		}
	}()

	lAddr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 12345,
	}
	conn, err := net.ListenUDP("udp", &lAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer conn.Close()

	go func() {
		payload := make([]byte, 1024)
		_, _, err := conn.ReadFromUDP(payload)
		if err != nil {
			t.Fatal(err.Error())
		}
		payloadC <- payload
	}()

	wAddr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 12346,
	}

	Node(wAddr, lAddr, MakeTestLogger())

	testResult := <-testResultC
	if testResult != true {
		t.Fatal("Expect a heartbeat within a time unit")
	}

}

func Test_OvertrhowDictator(t *testing.T) {

	nodeID := "1"
	testResultC := make(chan bool)
	timeout := time.After(3 * time.Second)
	payloadC := make(chan []byte)
	udpOutC := make(chan UDPPacket)
	sendHeadbeat := time.Tick(150 * time.Millisecond)

	lAddr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 12345,
	}

	wAddr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 12346,
	}

	// Test node behavoir
	go func() {
		for {
			select {
			case <-timeout:
				testResultC <- true
			case payload := <-payloadC:
				p := DictatorPayload{}
				err := bson.Unmarshal(payload, &p)
				if err != nil {
					t.Fatal(err.Error())
				}
				if p.Type == 1 {
					testResultC <- false
				}
			case <-sendHeadbeat:
				heartbeat := DictatorPayload{
					Type:       1,
					DictatorID: nodeID,
					Blob:       0,
				}
				payload, err := bson.Marshal(heartbeat)
				if err != nil {
					t.Fatal(err.Error())
				}
				packet := UDPPacket{
					RemoteAddr: &wAddr,
					Payload:    payload,
				}
				udpOutC <- packet
			}
		}
	}()

	inConn, err := net.ListenUDP("udp", &lAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer inConn.Close()

	tAddr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 12347,
	}
	outConn, err := net.DialUDP("udp", &tAddr, &wAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer outConn.Close()

	// Manage incoming UDP packets
	go func() {
		for {
			payload := make([]byte, 1024)
			_, _, err := inConn.ReadFromUDP(payload)
			if err != nil {
				t.Fatal(err.Error())
			}
			payloadC <- payload
		}
	}()

	// Manage outgoing UDP packets
	go func() {
		for {
			select {
			case p := <-udpOutC:
				outConn.WriteToUDP(p.Payload, p.RemoteAddr)
			}
		}
	}()

	Node(wAddr, lAddr, MakeTestLogger())

	testResult := <-testResultC
	if testResult != true {
		t.Fatal("Expect no heartbeat within a time unit")
	}
}
