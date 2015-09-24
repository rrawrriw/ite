package dictator

import (
	"testing"

	"gopkg.in/mgo.v2/bson"
)

func Test_CommandRouter(t *testing.T) {
	router := CommandRouter{}
	h := func(nCtx NodeContext, p DictatorPayload) error {
		return nil

	}
	router.AddHandler("test", h)

	fun, ok := router.FindHandler("test")

	if !ok {
		t.Fatal("Expect to find a handler")
	}

	result := fun(NodeContext{}, DictatorPayload{})

	if result != nil {
		t.Fatal(result.Error())
	}

}

func Test_NewCommandPacket_OK(t *testing.T) {
	cmdValue := 1
	cmdName := "test"
	dictatorID := "1"
	packet, err := NewCommandPacket(dictatorID, cmdName, cmdValue)

	if err != nil {
		t.Fatal(err)
	}

	dPayload := DictatorPayload{}
	err = bson.Unmarshal(packet.Payload, &dPayload)
	if err != nil {
		t.Fatal(err)
	}

	if !IsCommand(dPayload) {
		t.Fatal("Expect to be Command")
	}

	if dPayload.DictatorID != dictatorID {
		t.Fatal("Expect", dictatorID, "was", dPayload.DictatorID)
	}

	cmdBlob := CommandBlob{}
	err = bson.Unmarshal(dPayload.Blob, &cmdBlob)

	if cmdBlob.Name != cmdName {
		t.Fatal("Expect", cmdName, "was", cmdBlob.Name)
	}

	value := cmdBlob.Value.(int)
	if value != cmdValue {
		t.Fatal("Expect", cmdValue, "was", cmdBlob.Value)
	}
}

func Test_NewCommandResponsePacket_OK(t *testing.T) {
	respStatus := 1
	respResult := 2
	dictatorID := "1"
	nodeID := "2"
	packet, err := NewCommandResponsePacket(dictatorID, nodeID, respStatus, respResult)
	if err != nil {
		t.Fatal(err)
	}

	dPayload := DictatorPayload{}
	err = bson.Unmarshal(packet.Payload, &dPayload)
	if err != nil {
		t.Fatal(err)
	}

	if !IsCommandResponse(dPayload) {
		t.Fatal("Expect to be CommandResponse")
	}

	cResponseBlob := CommandResponseBlob{}
	err = bson.Unmarshal(dPayload.Blob, &cResponseBlob)
	if err != nil {
		t.Fatal(err)
	}

	if cResponseBlob.NodeID != nodeID {
		t.Fatal("Expect", nodeID, "was", cResponseBlob.NodeID)
	}

	if cResponseBlob.Status != respStatus {
		t.Fatal("Expect", respStatus, "was", cResponseBlob.Status)
	}

	result := cResponseBlob.Result.(int)
	if result != respResult {
		t.Fatal("Expect", respResult, "was", result)
	}

}
