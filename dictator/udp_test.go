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

	_, err = connOut.Write([]byte("1"))
	if err != nil {
		t.Fatal(err.Error())
	}

	err = <-testResult
	if err != nil {
		t.Fatal(err.Error())
	}

}
