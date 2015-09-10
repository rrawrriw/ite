package dictator

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func Test_RunUDPInbox(t *testing.T) {

	time.Sleep(500 * time.Millisecond)

	testResult := make(chan error)

	lAddr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 12345,
	}
	connIn, err := net.ListenUDP("udp", &lAddr)
	if err != nil {
		t.Fatal(err.Error())
	}

	conns := []*net.UDPConn{
		connIn,
	}
	ctx := NewContextWithConn(conns)
	defer ctx.Done()
	udpIn, err := UDPInbox(ctx, connIn)

	//Test if udp listner is running and forwarding
	testTimeout := time.After(200 * time.Millisecond)
	go func() {
		select {
		case result := <-udpIn:
			expect := make([]byte, MaxUDPPacketSize)
			expect[0] = '1'
			if !bytes.Equal(expect, result.Payload) {
				errMsg := fmt.Sprintf("Expect %v was %v", string(expect), string(result.Payload))
				testResult <- errors.New(errMsg)
			}
			testResult <- nil
		case <-testTimeout:
			errMsg := "Test runs out of time"
			testResult <- errors.New(errMsg)
		}

		testResult <- nil
	}()

	//Test if server startetd
	connOut, err := net.DialUDP("udp", nil, &lAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	_, err = connOut.Write([]byte("1"))
	if err != nil {
		t.Fatal(err.Error())
	}
	defer connOut.Close()

	err = <-testResult
	if err != nil {
		t.Fatal(err.Error())
	}

}

func Test_RunUDPOutbox(t *testing.T) {

	time.Sleep(500 * time.Millisecond)

	testResult := make(chan error)

	lAddr := net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 12345,
	}

	conn, err := net.DialUDP("udp", nil, &lAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	conns := []*net.UDPConn{
		conn,
	}
	ctx := NewContextWithConn(conns)
	defer ctx.Done()

	udpOut, err := UDPOutbox(ctx, conn)

	connIn, err := net.ListenUDP("udp", &lAddr)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer connIn.Close()

	// Test if udp out sends udp packet
	go func() {
		payload := make([]byte, MaxUDPPacketSize)
		_, _, err := connIn.ReadFromUDP(payload)
		if err != nil {
			testResult <- err
		}
		expect := make([]byte, MaxUDPPacketSize)
		expect[0] = '1'
		if !bytes.Equal(expect, payload) {
			errMsg := fmt.Sprintf("Expect %v was %v", string(expect), string(payload))
			testResult <- errors.New(errMsg)
		}

		testResult <- nil
	}()

	udpOut <- UDPPacket{
		Payload: []byte("1"),
	}

	err = <-testResult
	if err != nil {
		t.Fatal(err.Error())
	}

	time.Sleep(500 * time.Millisecond)

}
