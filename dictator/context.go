package dictator

import (
	"net"
	"os"
)

type Context struct {
	Done     func()
	DoneChan <-chan struct{}
	Err      error
	Log      Logger
}

func NewContextWithConn(conns []*net.UDPConn) Context {
	doneC := make(chan struct{})

	doneF := func() {
		close(doneC)
		for _, c := range conns {
			err := c.Close()
			if err != nil {
				panic(err.Error())
			}
		}
	}

	return Context{
		Done:     doneF,
		DoneChan: doneC,
		Err:      nil,
		Log:      NewLogger(os.Stderr, os.Stdout),
	}
}

func NewContext() Context {
	doneC := make(chan struct{})

	doneF := func() {
		close(doneC)
	}

	return Context{
		Done:     doneF,
		DoneChan: doneC,
		Err:      nil,
		Log:      NewLogger(os.Stderr, os.Stdout),
	}
}
