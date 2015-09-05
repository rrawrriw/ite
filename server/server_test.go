package main

import (
	"errors"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

const (
	IPServerIP   = "255.255.255.255"
	IPServerPort = 12345
	IPClientIP   = "255.255.255.255"
	IPClientPort = 12346
)

/* Tester Funktion
 * Sind alle Ausgegeben IP Adressen unterschiedlich
 * und werden alle Request beantwortet.
 */
func Tester(numberOfClients int, boom <-chan time.Time, testerResult chan error, testerIPs chan string) {
	ips := make(map[string]int)
	for {
		select {
		case ip := <-testerIPs:
			log.Println("Tester receive", ip)
			_, ok := ips[ip]
			if ok {
				testerResult <- errors.New("Got same IP")
			}

			ips[ip] = 1
			log.Println("Collected IPs", strconv.Itoa(len(ips)))

			if len(ips) == numberOfClients {
				testerResult <- nil
			}
		case <-boom:
			testerResult <- errors.New("You are to slow!")
		}
	}

	testerResult <- nil
}

func AskForIPs(numberOfClients int) {
	log.Println("Ask for IPs")
	for x := 0; x < numberOfClients; x++ {

		go func(n int) {
			rAddr := net.UDPAddr{
				IP:   net.ParseIP(IPServerIP),
				Port: IPServerPort,
			}

			conn, err := net.DialUDP("udp", nil, &rAddr)
			if err != nil {
				log.Fatal(err.Error())
			}
			defer conn.Close()
			log.Println("Ask For IP", n)
			_, err = conn.Write([]byte("1"))
			if err != nil {
				log.Fatal(err.Error())
			}
		}(x)

	}
}

func IPReceivers(numberOfClients int, tester chan string) {
	log.Println("Start IPReceivers")
	addr := net.UDPAddr{
		IP:   net.ParseIP(IPClientIP),
		Port: IPClientPort,
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatal(err.Error())
	}
	for x := 0; x < numberOfClients; x++ {
		response := make([]byte, 2048)
		_, rAddr, err := conn.ReadFromUDP(response)
		if err != nil {
			log.Fatal(err.Error())
		}
		go func(n int, rAddr *net.UDPAddr, response []byte) {
			log.Println("Handle IP response", n)
			if err != nil {
				log.Fatal(err.Error())
			}
			ipResponse := IPResponse{}
			err := bson.Unmarshal(response, &ipResponse)
			if err != nil {
				log.Println("ERROR BSON:", err.Error())
				return
			}
			log.Println(
				"IP Response value",
				rAddr,
				ipResponse.NewIP,
			)
			tester <- ipResponse.NewIP
		}(x, rAddr, response)
	}
}

func Test_GetIPFromIPServer(t *testing.T) {

	numberOfClients := 10

	cmd := exec.Command(
		"./ite-server",
		"-ip", IPServerIP,
		"-port", strconv.Itoa(IPServerPort),
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatal(err.Error())
	}
	defer cmd.Wait()
	defer cmd.Process.Kill()

	log.Println(cmd.Process.Pid)
	time.Sleep(500 * time.Millisecond)

	boom := time.After(3 * time.Second)
	testerResult := make(chan error)
	testerIPs := make(chan string, numberOfClients)

	go Tester(numberOfClients, boom, testerResult, testerIPs)

	go IPReceivers(numberOfClients, testerIPs)

	go AskForIPs(numberOfClients)

	err := <-testerResult
	if err != nil {
		t.Fatal(err.Error())
	}

}
