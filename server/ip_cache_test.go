package main

func Test_NextIP_OK(t *testing.T) {
	resultC := make(chan IPItemResponse)

	spec := SubnetSpec{
		Sub:  16,
		From: net.ParseIP("192.168.1.254"),
		To:   net.ParseIP("192.168.2.10"),
	}

	c := NewIPCache(spec, []net.IP{net.ParseIP("192.168.1.254")}, 5)
	errC := c.NextIP(resultC)
	err <- errC
	if err != nil {
		t.Fatal(err.Error())
	}

	result := <-resultC

	if result.Err != nil {
		t.Fatal(result.Err.Error())
	}

	expectIP := "192.168.2.1"
	if result.IP != net.ParseIP(expectIP) {
		t.Fatal("Expect", expectIP, ", was", result.IP)
	}

	if result.ConfirmID == "" {
		t.Fatal("ConfirmID is empty!")
	}
}

func Test_CleanIPCache(t *testing.T) {
	ipCache := IPCache{
		IPItem{
			IP:     net.ParseIP("192.168.1.1"),
			Expire: time.Now().Add(-1),
		},
		IPItem{
			IP:     net.ParseIP("192.168.1.2"),
			Expire: time.Now().Add(-1),
		},
	}
	CleanIPCache(&ipCache)

	if len(ipCache) != 1 {
		t.Fatal("Error to many IPItems in cache", ipCache)
	}
}

func Test_InitIPCache(t *testing.T) {
	ipCache := IPCache{}
	ipItems := []IPItem{
		IPItem{
			IP:     net.ParseIP("192.168.1.1"),
			Expire: time.Now(),
		},
		IPItem{
			IP:     net.ParseIP("192.168.1.2"),
			Expire: time.Now(),
		},
	}
	InitIPCache(ipItems, &ipCache)

	if len(ipCache) != 2 {
		t.Fatal("Not enough IPItems in cache", ipCache)
	}
}

func Test_ReadIPByID_OK(t *testing.T) {
}
