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
