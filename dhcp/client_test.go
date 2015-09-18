package dhcp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func Test_MakeXidBytes_OK(t *testing.T) {
	b, err := MakeXidBytes(1)

	if err != nil {
		t.Fatal(err)
	}

	if len(b) != 4 {
		t.Fatal("Expect result to be 4 bytes, was", len(b))
	}

	expect := []byte{
		byte(1),
		byte(0),
		byte(0),
		byte(0),
	}
	if !bytes.Equal(b, expect) {
		t.Fatal("Expect", expect, "was", b)
	}
}

func Test_MakeXidBytes_Fail(t *testing.T) {
	_, err := MakeXidBytes(65536)

	if err == nil {
		t.Fatal("Expect error out of range")
	}
}

func Test_MakeMACAddrBytes_OK(t *testing.T) {
	mac, err := net.ParseMAC("34:23:87:01:c2:f9")
	if err != nil {
		t.Fatal(err)
	}

	b := MakeMACAddrBytes(mac)

	if len(b) != 16 {
		t.Fatal("Expect a 16 byte result was", len(b))
	}

	expect := []byte{
		byte(0x34),
		byte(0x23),
		byte(0x87),
		byte(0x01),
		byte(0xc2),
		byte(0xf9),
		byte(0), byte(0), byte(0), byte(0), byte(0),
		byte(0), byte(0), byte(0), byte(0), byte(0),
	}

	if !bytes.Equal(b, expect) {
		t.Fatal("Expect", expect, "was", b)
	}
}

func Test_MakeSecsBytes_OK(t *testing.T) {
	b, err := MakeSecsBytes(1)
	if err != nil {
		t.Fatal(err)
	}

	if len(b) != 2 {
		t.Fatal("Expect a 2 byte long result was", len(b))
	}

	expect := []byte{
		byte(1),
		byte(0),
	}

	if !bytes.Equal(expect, b) {
		t.Fatal("Expect", expect, "was", b)
	}
}

func Test_MakeIPBytes_OK(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")

	b := MakeIPBytes(ip)

	if len(b) != 4 {
		t.Fatal("Expect a 4 byte long result was", len(b))
	}

	expect := []byte{
		byte(192),
		byte(168),
		byte(1),
		byte(1),
	}
	if !bytes.Equal(expect, b) {
		t.Fatal("Expect", expect, "was", b)
	}
}

func Test_DHCPOptionsBytes_OK(t *testing.T) {
	opts := []DHCPOption{
		DHCPOption{1, []byte{byte(1)}, 1},
		DHCPOption{2, []byte{byte(2)}, 2},
	}

	b := MakeDHCPOptionsBytes(opts)

	// Magic Cookie + Options(code byte + len byte + len) + End
	if len(b) != 12 {
		t.Fatal("Expect a 12 byte long result was", len(b))
	}

	expect := []byte{
		byte(99), byte(130), byte(83), byte(99),
		byte(1), byte(1), byte(1),
		byte(2), byte(2), byte(2), byte(0),
		byte(255),
	}

	if !bytes.Equal(expect, b) {
		t.Fatal("Expect", expect, "was", b)
	}
}

func Test_MakeCStringBytes_OK(t *testing.T) {
	s := "bacon!"
	b, err := MakeCStringBytes(s, 10)

	if err != nil {
		t.Fatal(err)
	}

	if len(b) != 10 {
		t.Fatal("Expect a 10 byte long result was", len(b))
	}

	c := []byte(s)
	if !bytes.Equal(c, b[0:6]) {
		t.Fatal("Expect to appear", s, "was", string(b[0:6]))
	}
}

func Test_MakeCStringBytes_FailSlieceToSmall(t *testing.T) {
	s := "oh!"

	_, err := MakeCStringBytes(s, 1)

	if err == nil {
		t.Fatal("Expect Slice size is to small error")
	}
}

func Test_MakeCStringBytes_FailNotAllowedChar(t *testing.T) {
	s := "oä!"

	_, err := MakeCStringBytes(s, 24)

	if err == nil {
		t.Fatal("Expect Not allowed char error")
	}
}

func Test_ReadUint64_OK(t *testing.T) {
	expect := uint64(11)
	zeroIP := net.ParseIP("0.0.0.0")
	clientMACAddr, err := net.ParseMAC("34:23:87:01:c2:f9")
	if err != nil {
		t.Fatal(err)
	}
	specs := DHCPSpecs{
		Op:     1,
		HType:  1,
		HLen:   6,
		Hops:   0,
		Xid:    expect,
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
		t.Fatal(err)
	}

	r, err := ReadUint64(p, 4, 8)
	if err != nil {
		t.Fatal(err)
	}

	if r != expect {
		t.Fatal("Expect", expect, "was", r)
	}

}

func Test_ReadUint64_OK2(t *testing.T) {
	expect := uint64(2)
	zeroIP := net.ParseIP("0.0.0.0")
	clientMACAddr, err := net.ParseMAC("34:23:87:01:c2:f9")
	if err != nil {
		t.Fatal(err)
	}
	specs := DHCPSpecs{
		Op:     1,
		HType:  expect,
		HLen:   6,
		Hops:   0,
		Xid:    111,
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
		t.Fatal(err)
	}

	r, err := ReadUint64(p, 1, 2)
	if err != nil {
		t.Fatal(err)
	}

	if r != expect {
		t.Fatal("Expect", expect, "was", r)
	}

}

func Test_ReadUint64_FailPayloadError(t *testing.T) {
	_, err := ReadUint64([]byte{}, 1, 100)

	if err == nil {
		t.Fatal("Expect Payload error")
	}
}

func Test_ReadIP_OK(t *testing.T) {
	expect := net.ParseIP("192.168.1.1")
	ciAddr := expect
	zeroIP := net.ParseIP("0.0.0.0")
	clientMACAddr, err := net.ParseMAC("34:23:87:01:c2:f9")
	if err != nil {
		t.Fatal(err)
	}
	specs := DHCPSpecs{
		Op:     1,
		HType:  1,
		HLen:   6,
		Hops:   0,
		Xid:    1,
		Secs:   1,
		Flags:  0,
		CiAddr: ciAddr,
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
		t.Fatal(err)
	}

	r, err := ReadIP(p, 12)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(r, expect) {
		t.Fatal("Expect", expect, "was", r)
	}

}

func Test_ReadIP_FailIndexOutOfRange(t *testing.T) {
	_, err := ReadIP([]byte{}, 4)

	if err == nil {
		t.Fatal("Expect Index out of range error")
	}
}

func Test_ReadMAC_OK(t *testing.T) {
	zeroIP := net.ParseIP("0.0.0.0")
	expect, err := net.ParseMAC("34:23:87:01:c2:f9")
	if err != nil {
		t.Fatal(err)
	}
	clientMACAddr := expect
	specs := DHCPSpecs{
		Op:     1,
		HType:  1,
		HLen:   6,
		Hops:   0,
		Xid:    1,
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
		t.Fatal(err)
	}

	r, err := ReadMAC(p, 28)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(r, expect) {
		t.Fatal("Expect", expect, "was", r)
	}

}

func Test_ReadMAC_FailIndexOutOfRange(t *testing.T) {
	_, err := ReadMAC([]byte{}, 4)

	if err == nil {
		t.Fatal("Expect Index out of range error")
	}
}

func Test_ReadFlags_OK(t *testing.T) {
	expect := uint64(7)
	zeroIP := net.ParseIP("0.0.0.0")
	clientMACAddr, err := net.ParseMAC("34:23:87:01:c2:f9")
	if err != nil {
		t.Fatal(err)
	}
	specs := DHCPSpecs{
		Op:     1,
		HType:  2,
		HLen:   3,
		Hops:   4,
		Xid:    5,
		Secs:   6,
		Flags:  expect,
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
		t.Fatal(err)
	}

	r, err := ReadFlags(p)
	if err != nil {
		t.Fatal(err)
	}

	if r != expect {
		t.Fatal("Expect", expect, "was", r)
	}

}

func Test_ReadCString_OK(t *testing.T) {
	expect := "Name"
	zeroIP := net.ParseIP("0.0.0.0")
	clientMACAddr, err := net.ParseMAC("34:23:87:01:c2:f9")
	if err != nil {
		t.Fatal(err)
	}
	specs := DHCPSpecs{
		Op:     1,
		HType:  2,
		HLen:   3,
		Hops:   4,
		Xid:    5,
		Secs:   6,
		Flags:  7,
		CiAddr: zeroIP,
		YiAddr: zeroIP,
		SiAddr: zeroIP,
		GiAddr: zeroIP,
		CHAddr: clientMACAddr,
		SName:  expect,
		File:   "",
		Options: []DHCPOption{
			DHCPOption{53, []byte{1}, 1},
			DHCPOption{61, MakeMACAddrBytes(clientMACAddr), 16},
			DHCPOption{12, []byte("GO"), 2},
		},
	}

	p, err := MakeClientPayload(specs)
	if err != nil {
		t.Fatal(err)
	}

	r, err := ReadCString(p, 44, 64)
	if err != nil {
		t.Fatal(err)
	}

	if r != expect {
		t.Fatal("Expect", expect, "was", r)
	}

}

func Test_ReadCString_OK2(t *testing.T) {
	expect := "0123456789012345678901234567890123456789012345678901234567891234"
	r, err := ReadCString([]byte(expect), 0, 64)

	if err != nil {
		t.Fatal(err)
	}

	if r != expect {
		t.Fatal("Expect", expect, "was", r)
	}

}

func Test_ReadCString_FailPayloadError(t *testing.T) {
	expect := "0123456789012345678901234567890123456789012345678901234567891234"
	_, err := ReadCString([]byte(expect), 0, 65)

	if err == nil {
		t.Fatal("Expect Payload error")
	}

}

func Test_ReadDHCPOptions_FailMagicCookie(t *testing.T) {
	_, err := ReadDHCPOptions(make([]byte, 241))

	if err == nil {
		t.Fatal("Expect Cannot find maigc cookie error")
	}
}

func Test_ReadDHCPOptions_FailPayloadError(t *testing.T) {
	_, err := ReadDHCPOptions([]byte{})

	if err == nil {
		t.Fatal("Expect Payload error")
	}
}

func Test_ReadDHCPOptions_OK(t *testing.T) {
	zeroIP := net.ParseIP("0.0.0.0")
	clientMACAddr, err := net.ParseMAC("34:23:87:01:c2:f9")
	if err != nil {
		t.Fatal(err)
	}
	specs := DHCPSpecs{
		Op:     1,
		HType:  1,
		HLen:   6,
		Hops:   0,
		Xid:    1,
		Secs:   1,
		Flags:  1,
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
		t.Fatal(err)
	}

	r, err := ReadDHCPOptions(p)
	if err != nil {
		t.Fatal(err)
	}

	for i, o := range specs.Options {
		if r[i].Code != o.Code ||
			r[i].Len != o.Len {
			t.Fatal("Expect", specs, "was", r)
		}
	}

}

func Test_ReadDHCPSpecs_OK(t *testing.T) {
	clientMACAddr, err := net.ParseMAC("34:23:87:01:c2:f9")
	if err != nil {
		t.Fatal(err)
	}

	ciAddr := net.ParseIP("192.168.1.1")
	yiAddr := net.ParseIP("192.168.1.2")
	siAddr := net.ParseIP("192.168.1.3")
	giAddr := net.ParseIP("192.168.1.4")

	specs := DHCPSpecs{
		Op:     1,
		HType:  2,
		HLen:   6,
		Hops:   7,
		Xid:    11,
		Secs:   9,
		Flags:  3,
		CiAddr: ciAddr,
		YiAddr: yiAddr,
		SiAddr: siAddr,
		GiAddr: giAddr,
		CHAddr: clientMACAddr,
		SName:  "Name",
		File:   "File",
		Options: []DHCPOption{
			DHCPOption{53, []byte{1}, 1},
			DHCPOption{61, MakeMACAddrBytes(clientMACAddr), 16},
			DHCPOption{12, []byte("GO"), 2},
		},
	}

	p, err := MakeClientPayload(specs)
	if err != nil {
		t.Fatal(err)
	}

	r, err := ReadDHCPSpecs(p)
	if err != nil {
		t.Fatal(err)
	}

	if specs.Op != r.Op ||
		specs.HType != r.HType ||
		specs.HLen != r.HLen ||
		specs.Xid != r.Xid ||
		specs.Secs != r.Secs ||
		specs.Flags != r.Flags ||
		!bytes.Equal(specs.CiAddr, r.CiAddr) ||
		!bytes.Equal(specs.YiAddr, r.YiAddr) ||
		!bytes.Equal(specs.SiAddr, r.SiAddr) ||
		!bytes.Equal(specs.GiAddr, r.GiAddr) ||
		!bytes.Equal(specs.CHAddr, r.CHAddr) ||
		specs.SName != r.SName ||
		specs.File != r.File {

		t.Fatal("Expect", specs, "was", r)
	}

	for i, o := range specs.Options {
		if r.Options[i].Code != o.Code ||
			r.Options[i].Len != o.Len {
			t.Fatal("Expect", specs, "was", r)
		}
	}

}

func existsIP(l []net.IP, ip net.IP) bool {
	for _, e := range l {
		if bytes.Equal(e, ip) {
			return true
		}
	}

	return false
}

func removeIP(l []net.IP, ip net.IP) []net.IP {
	i := -1
	for x, e := range l {
		if bytes.Equal(e, ip) {
			i = x
		}
	}

	if i < 0 {
		return l
	}

	return append(l[:i], l[i+1:]...)
}

func Test_ResponseHandlerDiscover_OK(t *testing.T) {
	conns := []*net.UDPConn{}
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	ctx := NewContext(conns, timer)

	in := make(chan UDPPacket)

	nodeID := uint64(11)
	out, timeout := ResponseHandlerDiscover(ctx, in, nodeID)

	amountIPs := 5
	ips := make([]net.IP, amountIPs)
	for x := 0; x < amountIPs; x++ {
		ipS := fmt.Sprintf("192.168.1.%v", x)
		ips[x] = net.ParseIP(ipS)
	}

	// Tester goroutine
	result := make(chan error)
	go func() {
		for {
			select {
			case <-timeout:
				if len(ips) != 0 {
					result <- errors.New("IPs are misssing")
				}
			case ip := <-out:
				if existsIP(ips, ip) {
					ips = removeIP(ips, ip)
					if len(ips) == 0 {
						result <- nil
					}
				}
			}
		}
	}()

	// Create UDPPackets
	udpPackets := make([]UDPPacket, amountIPs)
	for i, ip := range ips {
		zeroIP := net.ParseIP("0.0.0.0")
		macAddr, err := net.ParseMAC("34:23:87:01:c2:f9")
		if err != nil {
			t.Fatal(err)
		}

		specs := DHCPSpecs{
			Op:     1,
			HType:  2,
			HLen:   6,
			Hops:   7,
			Xid:    nodeID,
			Secs:   9,
			Flags:  3,
			CiAddr: zeroIP,
			YiAddr: ip,
			SiAddr: zeroIP,
			GiAddr: zeroIP,
			CHAddr: macAddr,
			SName:  "Name",
			File:   "File",
			Options: []DHCPOption{
				DHCPOption{53, []byte{1}, 1},
				DHCPOption{61, MakeMACAddrBytes(macAddr), 16},
				DHCPOption{12, []byte("GO"), 2},
			},
		}

		p, err := MakeClientPayload(specs)
		if err != nil {
			t.Fatal(err)
		}

		udpPackets[i] = UDPPacket{
			Payload: p,
		}

	}

	for _, p := range udpPackets {
		in <- p
	}

	err := <-result
	if err != nil {
		t.Fatal(err)
	}
}
