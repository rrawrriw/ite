package dictator

import "net"

const MaxUDPPacketSize = 1024

type UDPPacket struct {
	RemoteAddr *net.UDPAddr
	Payload    []byte
	Size       int
}

func UDPInbox(ctx Context, conn *net.UDPConn) (chan UDPPacket, error) {
	udpIn := make(chan UDPPacket)
	go func() {
		for {
			payload := make([]byte, MaxUDPPacketSize)

			size, rAddr, err := conn.ReadFromUDP(payload)
			if err != nil {
				// Check if the done channel closed then shutdown goroutine
				select {
				case <-ctx.DoneChan:
					close(udpIn)
					return
				default:
					// Need default case otherwise the select statment would block
				}

				ctx.Log.Error.Println(err.Error())
				continue
			}

			udpIn <- UDPPacket{
				RemoteAddr: rAddr,
				Size:       size,
				Payload:    payload,
			}

		}

	}()
	return udpIn, nil
}
