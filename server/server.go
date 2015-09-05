package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"

	"gopkg.in/mgo.v2/bson"
)

type (
	IPServer struct {
		Addr string
	}
	IPResponse struct {
		NewIP string
	}
)

func main() {
	serverIP := flag.String("ip", "255.255.255.255", "Server IP")
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

	for x := 1; ; x++ {
		request := make([]byte, 2)
		_, rAddr, err := conn.ReadFromUDP(request)
		if err != nil {
			log.Fatal(err.Error())
		}
		log.Println(
			"Receive IP Request From",
			rAddr,
		)

		tmpIP := "192.168.1." + strconv.Itoa(x)
		ipResponse := IPResponse{
			NewIP: tmpIP,
		}
		response, err := bson.Marshal(ipResponse)
		if err != nil {
			log.Println("ERROR:", err.Error())
			continue
		}

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
		log.Println("Response with IP", ipResponse.NewIP)
	}

	return nil
}
