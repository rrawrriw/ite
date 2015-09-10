package dictator

import (
	"errors"
	"net"
	"testing"
	"time"
)

func Test_ContextCloseFuncWithConn(t *testing.T) {

	testResult := make(chan error)

	conns := make([]*net.UDPConn, 3)
	for x := 0; x < 3; x++ {
		a := net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 12345 + x,
		}
		c, err := net.ListenUDP("udp", &a)
		if err != nil {
			t.Fatal(err.Error())
		}
		conns[x] = c
	}

	ctx := NewContextWithConn(conns)

	// Test done channel
	testTimeout := time.After(200 * time.Millisecond)
	go func() {
		select {
		case <-testTimeout:
			errMsg := "Expect to close done channel"
			testResult <- errors.New(errMsg)
		case <-ctx.DoneChan:
			testResult <- nil
		}
	}()

	ctx.Done()

	// Test if a connection not closed
	for x := 0; x < 3; x++ {
		a := net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 12345 + x,
		}
		_, err := net.ListenUDP("udp", &a)
		if err != nil {
			t.Fatal(err.Error())
		}
	}

	err := <-testResult
	if err != nil {
		t.Fatal(err.Error())
	}
}
