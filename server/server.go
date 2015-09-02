package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

type IPServer struct {
	Addr string
}

func main() {
	serverIP := flag.String("ip", "127.0.0.1", "Server IP")
	serverPort := flag.Int("port", 0, "Server Port")
	flag.Parse()

	if (*serverIP != "") && (*serverPort != 0) {
		s := IPServer{
			Addr: fmt.Sprintf(
				"%v:%v",
				*serverIP,
				*serverPort,
			),
		}
		log.Fatal(s.Run())
	}
}

func (s IPServer) Run() error {
	log.Println("Run server - ", s.Addr)
	addr, err := net.ResolveUDPAddr("udp", s.Addr)
	if err != nil {
		log.Fatal(err.Error())
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err.Error())
	}

	for {
		packet := make([]byte, 2)
		read, rAddr, err := conn.ReadFromUDP(packet)
		if err != nil {
			log.Fatal(err.Error())
		}
		log.Println(read, rAddr, string(packet))
		response := []byte("2")

		_, err = conn.WriteToUDP(
			response,
			&net.UDPAddr{
				IP:   net.ParseIP("255.255.255.255"),
				Port: 12346,
			},
		)
		if err != nil {
			log.Fatal(err.Error())
		}
		log.Println("Response with IP")
	}

	return nil
}
