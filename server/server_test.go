package main

import (
	"errors"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

const (
	IPServerIP   = "127.0.0.1"
	IPServerPort = 12345
	IPClientIP   = "127.0.0.1"
	IPClientPort = 12346
)

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
		go func(n int) {
			log.Println("Start IP reciever", n)
			packet := []byte{}
			_, _, err := conn.ReadFromUDP(packet)
			if err != nil {
				close(tester)
			}
			log.Println(string(packet))
		}(x)
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

	time.Sleep(500 * time.Millisecond)

	boom := time.After(3 * time.Second)
	testerResult := make(chan error, numberOfClients)
	testerIPs := make(chan string)
	/* Tester Funktion
	 * Sind alle Ausgegeben IP Adressen unterschiedlich
	 * und werden alle Request beantwortet.
	 */
	go func() {
		ips := make(map[string]int)
		select {
		case ip := <-testerIPs:
			_, ok := ips[ip]
			if ok {
				testerResult <- errors.New("too mutch")
			}
			ips[ip] = 1
		case <-boom:
			testerResult <- errors.New("test")
		}

		testerResult <- nil
	}()

	go IPReceivers(numberOfClients, testerIPs)

	go AskForIPs(numberOfClients)

	err := <-testerResult
	if err != nil {
		t.Fatal(err.Error())
	}

}
