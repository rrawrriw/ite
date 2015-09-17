package dhcp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"time"
	"unicode/utf8"
)

const MaxUDPPacketSize = 1024

type UDPPacket struct {
	RemoteAddr *net.UDPAddr
	Payload    []byte
	Size       int
}

type DHCPSpecs struct {
	Op      uint64 // Message type
	HType   uint64 // Hardware address type ex. Ethernet no. 1
	HLen    uint64 // Hardware address length ex. 6 for MAC address
	Hops    uint64 // Amount of hops
	Xid     uint64 // Connection ID
	Secs    uint64 // Secounds since the client started
	Flags   uint64
	CiAddr  net.IP           // Client IP
	YiAddr  net.IP           // Owen IP
	SiAddr  net.IP           // Server IP
	GiAddr  net.IP           // Realy-Agent IP
	CHAddr  net.HardwareAddr // Client MAC address
	SName   string           // Name of the dhcp-server
	File    string           // Filename
	Options []DHCPOption     // DHCP Parameter min size 312 Byte. The max size can be trade between server and client
}

type DHCPOption struct {
	Code  uint64
	Value []byte
	Len   uint64
}

type Context struct {
	Done     func()
	DoneChan <-chan struct{}
	Err      error
	Timeout  *time.Timer
	Log      Logger
}

type Logger struct {
	Error *log.Logger
	Debug *log.Logger
}

func NewLogger(errWriter, debugWriter io.Writer) Logger {

	l := Logger{
		Error: log.New(errWriter, "Error: ", log.LstdFlags|log.Lshortfile),
		Debug: log.New(debugWriter, "Debug: ", log.LstdFlags|log.Lshortfile),
	}

	return l
}

func NewContext(conns []*net.UDPConn, timeout *time.Timer) Context {
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
		Timeout:  timeout,
		Log:      NewLogger(os.Stderr, os.Stdout),
	}
}

func UDPOutbox(ctx Context, conn *net.UDPConn, buf int) (chan UDPPacket, error) {
	udpOut := make(chan UDPPacket, buf)
	go func() {
		for {
			select {
			case <-ctx.DoneChan:
				ctx.Log.Debug.Println("UDPOutbox shutdown")
				return
			case packet := <-udpOut:
				_, err := conn.Write(packet.Payload)
				if err != nil {
					ctx.Log.Error.Println(err.Error())
					continue
				}
			}
		}
	}()
	return udpOut, nil
}

func UDPInbox(ctx Context, conn *net.UDPConn, buf int) (chan UDPPacket, error) {
	udpIn := make(chan UDPPacket, buf)
	go func() {
		for {
			payload := make([]byte, MaxUDPPacketSize)

			size, rAddr, err := conn.ReadFromUDP(payload)
			if err != nil {
				// Check if the done channel closed then shutdown goroutine
				select {
				case <-ctx.DoneChan:
					ctx.Log.Debug.Println("UDPInbox shutdown")
					close(udpIn)
					return
				default:
					// Need default case otherwise the select statment would block
				}

				ctx.Log.Error.Println(err.Error())
				continue
			}
			ctx.Log.Debug.Println("Receive UDP Packet")
			udpIn <- UDPPacket{
				RemoteAddr: rAddr,
				Size:       size,
				Payload:    payload,
			}

		}

	}()
	return udpIn, nil
}

func MakeXidBytes(id uint64) ([]byte, error) {
	buf := make([]byte, 4)

	if float64(id) > (math.Pow(2, 4*4) - 1) {
		return []byte(""), errors.New("Xid is out of range")
	}

	binary.PutUvarint(buf, id)

	return buf, nil
}

func MakeSecsBytes(sec uint64) ([]byte, error) {
	buf := make([]byte, 2)

	if float64(sec) > (math.Pow(2, 4*2) - 1) {
		return []byte{}, errors.New("Secs is out of range")
	}

	binary.PutUvarint(buf, sec)

	return buf, nil
}

func MakeFlags(flags uint64) ([]byte, error) {
	buf := make([]byte, 2)

	if float64(flags) > (math.Pow(2, 4*2) - 1) {
		return []byte(""), errors.New("Flags is out of range")
	}

	binary.PutUvarint(buf, flags)

	return buf, nil
}

func MakeDHCPOptionsBytes(opts []DHCPOption) []byte {
	buf := bytes.NewBuffer([]byte{})

	// Add magic cookie OREO - IP 99.130.83.99 - hex 63.82.53.63
	buf.Write([]byte{byte(99), byte(130), byte(83), byte(99)})

	for _, v := range opts {
		buf.WriteByte(byte(v.Code))
		buf.WriteByte(byte(v.Len))
		tmp := make([]byte, v.Len)
		for i, b := range v.Value {
			tmp[i] = b
		}
		buf.Write(tmp)
	}

	buf.WriteByte(255)

	return buf.Bytes()
}

func MakeCStringBytes(s string, b int) ([]byte, error) {
	buf := make([]byte, b)

	if len(s) > b {
		errMsg := fmt.Sprintf("Slice size %v is to small", b)
		err := errors.New(errMsg)
		return []byte{}, err
	}

	for i, r := range s {
		if len(string(r)) > 1 {
			errMsg := fmt.Sprintf("Char %v not allowed", r)
			err := errors.New(errMsg)
			return []byte{}, err
		}
		tmp := make([]byte, 1)
		utf8.EncodeRune(tmp, r)
		buf[i] = tmp[0]
	}

	return buf, nil
}

func MakeIPBytes(ip net.IP) []byte {
	tmp := make([]byte, 4)
	tmp[0] = ip[12]
	tmp[1] = ip[13]
	tmp[2] = ip[14]
	tmp[3] = ip[15]
	return tmp
}

func MakeMACAddrBytes(mac net.HardwareAddr) []byte {
	tmp := make([]byte, 16)
	for x := 0; x < 6; x++ {
		tmp[x] = mac[x]
	}

	return tmp
}

func MakeClientPayload(specs DHCPSpecs) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})

	buf.WriteByte(byte(specs.Op))
	buf.WriteByte(byte(specs.HType))
	buf.WriteByte(byte(specs.HLen))
	buf.WriteByte(byte(specs.Hops))

	tmp, err := MakeXidBytes(uint64(specs.Xid))
	if err != nil {
		return []byte(""), err
	}
	buf.Write(tmp)

	tmp, err = MakeSecsBytes(uint64(specs.Secs))
	if err != nil {
		return []byte(""), err
	}
	buf.Write(tmp)

	// Flags
	tmp, err = MakeFlags(specs.Flags)
	if err != nil {
		return []byte(""), err
	}
	buf.Write(tmp)

	buf.Write(MakeIPBytes(specs.CiAddr))
	buf.Write(MakeIPBytes(specs.YiAddr))
	buf.Write(MakeIPBytes(specs.SiAddr))
	buf.Write(MakeIPBytes(specs.GiAddr))

	buf.Write(MakeMACAddrBytes(specs.CHAddr))

	tmp, err = MakeCStringBytes(specs.SName, 64)
	if err != nil {
		return []byte{}, err
	}
	buf.Write(tmp)

	tmp, err = MakeCStringBytes(specs.File, 128)
	if err != nil {
		return []byte{}, err
	}
	buf.Write(tmp)

	tmp = MakeDHCPOptionsBytes(specs.Options)
	buf.Write(tmp)

	return buf.Bytes(), nil

}

func NewDHCPDiscover() ([]byte, error) {
	zeroIP := net.ParseIP("0.0.0.0")
	iFace, err := net.InterfaceByName("wlp1s0")
	if err != nil {
		return []byte{}, err
	}
	clientMACAddr := iFace.HardwareAddr
	specs := DHCPSpecs{
		Op:     1,
		HType:  1,
		HLen:   6,
		Hops:   0,
		Xid:    11, // random
		Secs:   1,
		Flags:  0,
		CiAddr: zeroIP,
		YiAddr: zeroIP,
		SiAddr: zeroIP,
		GiAddr: zeroIP,
		CHAddr: clientMACAddr,
		SName:  "",
		File:   "",
		Options: []DHCPOption{
			DHCPOption{53, []byte{1}, 1},
			DHCPOption{61, MakeMACAddrBytes(clientMACAddr), 16},
			DHCPOption{12, []byte("GO"), 2},
		},
	}

	p, err := MakeClientPayload(specs)
	if err != nil {
		return []byte{}, err
	}
	fmt.Println(p)

	return p, nil

}

func ReadUint64(payload []byte, s, e int) (uint64, error) {
	if len(payload) < e {
		return uint64(0), errors.New("Payload error")
	}
	xid, err := binary.ReadUvarint(bytes.NewBuffer(payload[s:e]))
	if err != nil {
		return uint64(0), err
	}

	return xid, nil
}

func ReadIP(payload []byte, s int) (net.IP, error) {
	if len(payload) < s+4 {
		return net.IP{}, errors.New("Index out of range")
	}

	a := payload[s]
	b := payload[s+1]
	c := payload[s+2]
	d := payload[s+3]

	ip := net.IPv4(a, b, c, d)

	return ip, nil

}

func ReadMAC(payload []byte, s int) (net.HardwareAddr, error) {
	if len(payload) <= s+16 {
		return net.HardwareAddr{}, errors.New("Index out of range")
	}

	a := payload[s]
	b := payload[s+1]
	c := payload[s+2]
	d := payload[s+3]
	e := payload[s+4]
	f := payload[s+5]

	mac := net.HardwareAddr{a, b, c, d, e, f}

	return mac, nil

}

func ReadFlags(payload []byte) (uint64, error) {
	if len(payload) < 12 {
		errors.New("Payload error")
	}

	i, err := binary.ReadUvarint(bytes.NewBuffer(payload[10:12]))
	if err != nil {
		return 0, err
	}
	if err != nil {
		return 0, err
	}

	return i, nil

}

func ReadCString(payload []byte, s, offset int) string {
	b := payload[s : s+offset]
	return string(bytes.TrimSuffix(b, []byte{byte(0)}))
}

func ReadDHCPOptions(payload []byte) ([]DHCPOption, error) {
	magicCookieStart := 236
	magicCookieEnd := 240

	if len(payload) < magicCookieEnd {
		return []DHCPOption{}, errors.New("Payload error")
	}

	magicCookie := []byte{byte(99), byte(130), byte(83), byte(99)}

	if !bytes.Equal(payload[magicCookieStart:magicCookieEnd], magicCookie) {
		return []DHCPOption{}, errors.New("Cannot find magic cookie")
	}

	opts := []DHCPOption{}
	pR := bytes.NewReader(payload[magicCookieEnd:])
	// End Loop when only 255 end DHCP option is left
	for pR.Len() != 1 {

		buf := make([]byte, 1)
		if _, err := pR.Read(buf); err != nil {
			return []DHCPOption{}, err
		}

		code, err := binary.ReadUvarint(bytes.NewReader(buf))
		if err != nil {
			return []DHCPOption{}, err
		}

		buf = make([]byte, 1)
		if _, err := pR.Read(buf); err != nil {
			return []DHCPOption{}, err
		}
		size, err := binary.ReadUvarint(bytes.NewReader(buf))
		if err != nil {
			return []DHCPOption{}, err
		}

		buf = make([]byte, size)
		if _, err := pR.Read(buf); err != nil {
			return []DHCPOption{}, err
		}

		opts = append(opts, DHCPOption{code, buf, size})

	}

	return opts, nil
}

func ReadDHCPSpecs(payload []byte) (DHCPSpecs, error) {
	specs := DHCPSpecs{}

	op, err := ReadUint64(payload, 0, 1)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.Op = op

	hType, err := ReadUint64(payload, 1, 2)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.HType = hType

	hLen, err := ReadUint64(payload, 2, 3)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.HLen = hLen

	hOps, err := ReadUint64(payload, 3, 4)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.Hops = hOps

	xID, err := ReadUint64(payload, 4, 8)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.Xid = xID

	secs, err := ReadUint64(payload, 8, 9)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.Secs = secs

	flags, err := ReadFlags(payload)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.Flags = flags

	ip, err := ReadIP(payload, 12)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.CiAddr = ip

	ip, err = ReadIP(payload, 16)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.YiAddr = ip

	ip, err = ReadIP(payload, 20)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.SiAddr = ip

	ip, err = ReadIP(payload, 24)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.GiAddr = ip

	cHAddr, err := ReadMAC(payload, 28)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.CHAddr = cHAddr

	opts, err := ReadDHCPOptions(payload)
	if err != nil {
		return DHCPSpecs{}, err
	}
	specs.Options = opts

	return specs, nil
}

func ResponseHandlerDiscover(ctx Context, in chan UDPPacket) (<-chan net.IP, <-chan struct{}) {
	ipOut := make(chan net.IP)
	timeout := make(chan struct{})
	go func() {
		for {
			select {
			case <-ctx.DoneChan:
				ctx.Log.Debug.Println("ResponseHandlerDiscover Done")
				return
			case <-ctx.Timeout.C:
				ctx.Log.Debug.Println("ResponseHandlerDiscover Timeout")
				ctx.Done()
				timeout <- struct{}{}
				return
			case packet := <-in:
				ip, _ := ReadIP(packet.Payload, 12)
				ipOut <- ip

			}
		}
	}()

	return ipOut, timeout

}

func RequestIPAddr(timeout time.Duration) (<-chan net.IP, <-chan struct{}, error) {
	inAddr := net.UDPAddr{
		//IP:   net.ParseIP("255.255.255.255"),
		IP:   net.ParseIP("192.168.0.14"),
		Port: 68,
	}
	connIn, err := net.ListenUDP("udp", &inAddr)
	if err != nil {
		return nil, nil, err
	}
	conns := []*net.UDPConn{
		connIn,
	}
	ctx := NewContext(conns, time.NewTimer(timeout))
	udpIn, err := UDPInbox(ctx, connIn, 10)
	if err != nil {
		return nil, nil, err
	}

	newIP, timeoutC := ResponseHandlerDiscover(ctx, udpIn)

	remoteAddr := net.UDPAddr{
		IP:   net.ParseIP("255.255.255.255"),
		Port: 67,
	}
	conn, err := net.DialUDP("udp", nil, &remoteAddr)
	if err != nil {
		ctx.Done()
		return nil, nil, err
	}

	p, err := NewDHCPDiscover()
	if err != nil {
		return nil, nil, err
	}

	s, err := conn.Write(p)
	if err != nil {
		ctx.Done()
		return nil, nil, err
	}
	fmt.Println("Send", s, "bytes")

	return newIP, timeoutC, nil
}

func main() {

	timeout := 8 * time.Second
	ip, timeoutC, err := RequestIPAddr(timeout)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for {
		select {
		case r := <-ip:
			fmt.Println(r)
			break
		case <-timeoutC:
			ip, timeoutC, err = RequestIPAddr(timeout)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			continue

		}
	}

}
