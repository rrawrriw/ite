package main

import (
	"fmt"
	"net"
	"time"

	"github.com/rrawrriw/ite/dhcp"
	"github.com/rrawrriw/ite/dictator"
)

func NewCluster() (dictator.Mission, dictator.ResponseChan) {
	response := make(dictator.ResponseChan)

	dhcpInAddr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 68,
	}

	dhcpOutAddr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 67,
	}

	mission := func(nCtx dictator.NodeContext) {
		log := nCtx.AppContext.Log
		go func() {
			log.Debug.Println(nCtx.NodeID, "- Start to build new cluster")

			ips, timeout, err := dhcp.RequestIPAddr(dhcpInAddr, dhcpOutAddr, 10*time.Second, "eth0")
			if err != nil {
				log.Error.Println(nCtx.NodeID, "-", err)
				return
			}

			// Wait for DHCP Server Response
			for {
				select {

				case <-timeout:
					log.Debug.Println(nCtx.NodeID, "- DHCP IP request run out of time")
				case ip := <-ips:
					log.Debug.Println(nCtx.NodeID, "- got", ip)
				case <-nCtx.AppContext.DoneChan:
					return
				case <-nCtx.SuicideChan:
					return
				}
			}

			// Send AssignIP commands until a slave response
			packet, err := dictator.NewCommandPacket(nCtx.NodeID, "AssignIP", nil)
			if err != nil {
				log.Error.Println(err)
				nCtx.AppContext.Done()
			}
			go func() {
				select {
				case <-nCtx.AppContext.DoneChan:
					return
				case <-nCtx.SuicideChan:
					return
				default:
					nCtx.UDPOut <- packet
					time.Sleep(1 * time.Second)
				}

			}()

			// Wait for a slave to response the AssingIP command
			for {
				select {
				case <-response:
					log.Debug.Println(nCtx.NodeID, "- Command successfully done")
					log.Debug.Println(nCtx.NodeID, "- Send reboot command")
					packet, err := dictator.NewCommandPacket(nCtx.NodeID, "Reboot", nil)
					if err != nil {
						log.Error.Println(err)
						nCtx.AppContext.Done()
					}
					nCtx.UDPOut <- packet
					time.Sleep(1 * time.Second)
					nCtx.AppContext.Done()
				case <-nCtx.AppContext.DoneChan:
					return
				case <-nCtx.SuicideChan:
					return
				}
			}
		}()
	}

	return mission, response
}

func AssignIPHandler(nCtx dictator.NodeContext, payload dictator.DictatorPayload) error {
	log := nCtx.AppContext.Log
	log.Debug.Println(nCtx.NodeID, "- Receive command AssignIP from", payload.DictatorID)

	response, err := dictator.NewCommandResponsePacket(payload.DictatorID, nCtx.NodeID, 1, nil)
	if err != nil {
		return err
	}

	nCtx.UDPOut <- response

	return nil
}

func RebootHandler(nCtx dictator.NodeContext, payload dictator.DictatorPayload) error {
	log := nCtx.AppContext.Log
	log.Debug.Println(nCtx.NodeID, "- Receive command Reboot from", payload.DictatorID)
	nCtx.AppContext.Done()

	return nil
}

func main() {
	inAddr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 43001,
	}
	connIn, err := net.ListenUDP("udp", &inAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	outAddr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 43001,
	}
	connOut, err := net.DialUDP("udp", nil, &outAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	conns := []*net.UDPConn{
		connIn,
		connOut,
	}
	ctx := dictator.NewContextWithConn(conns)

	udpIn, err := dictator.UDPInbox(ctx, connIn)
	if err != nil {
		fmt.Println(err)
		return
	}
	udpOut, err := dictator.UDPOutbox(ctx, connOut)
	if err != nil {
		fmt.Println(err)
		return
	}

	cmdRouter := dictator.CommandRouter{}
	cmdRouter.AddHandler("AssignIP", AssignIPHandler)
	cmdRouter.AddHandler("Reboot", RebootHandler)

	m, response := NewCluster()

	mission := dictator.MissionSpecs{
		Mission:       m,
		CommandRouter: cmdRouter,
		ResponseChan:  response,
	}

	dictator.Node(ctx, udpIn, udpOut, mission)

	<-ctx.DoneChan
}
