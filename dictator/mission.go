package dictator

import "gopkg.in/mgo.v2/bson"

type (
	MissionSpecs struct {
		// Funktion sollte eine goroutine starten damit der UDP Listener
		// weiterhin erreichbach ist.
		// Die Funktion sollte sich beenden wenn der AppContext Done Channel
		// geschlossen wird oder wenn der Suicide Channel angesprochen wird.
		Mission       Mission
		CommandRouter CommandRouter
		// An diesen Channel werden alle CommandResponse Paket weitergeleitet
		// Wird im allgemeien in der Mission Funktionen verwendet.
		ResponseChan ResponseChan
	}

	Mission func(NodeContext)

	CommandHandler func(NodeContext, DictatorPayload) error

	CommandRouter map[string]CommandHandler

	ResponseChan chan DictatorPayload

	// Gibt vor welchen Befehl die Nodes ausfürhen sollen
	CommandBlob struct {
		Name  string
		Value interface{}
	}

	// Gibt zurück ob Befehl erfolgreich ausgführt wurde
	CommandResponseBlob struct {
		NodeID string
		Status int
		Result interface{}
	}
)

func (r CommandRouter) AddHandler(name string, fun CommandHandler) {
	r[name] = fun
}

func (r CommandRouter) FindHandler(name string) (CommandHandler, bool) {
	fun, ok := r[name]
	if !ok {
		return *new(CommandHandler), ok
	}

	return fun, true
}

func NewCommandPacket(dictatorID string, cmdName string, cmdValue interface{}) (UDPPacket, error) {
	commandBlob := CommandBlob{
		Name:  cmdName,
		Value: cmdValue,
	}

	dictatorPayloadBlob, err := bson.Marshal(commandBlob)
	if err != nil {
		return UDPPacket{}, nil
	}

	dictatorPayload := DictatorPayload{
		Type:       2,
		DictatorID: dictatorID,
		Blob:       dictatorPayloadBlob,
	}

	udpPayload, err := bson.Marshal(dictatorPayload)
	if err != nil {
		return UDPPacket{}, nil
	}

	packet := UDPPacket{
		Payload: udpPayload,
	}

	return packet, nil

}

func NewCommandResponsePacket(dictatorID, nodeID string, respStatus int, respResult interface{}) (UDPPacket, error) {
	commandResponseBlob := CommandResponseBlob{
		Status: respStatus,
		Result: respResult,
		NodeID: nodeID,
	}
	commandBlob, err := bson.Marshal(commandResponseBlob)
	if err != nil {
		return UDPPacket{}, nil
	}

	dictatorPayload := DictatorPayload{
		Type:       3,
		Blob:       commandBlob,
		DictatorID: dictatorID,
	}
	udpPayload, err := bson.Marshal(dictatorPayload)
	if err != nil {
		return UDPPacket{}, nil
	}

	udpPacket := UDPPacket{
		Payload: udpPayload,
	}

	return udpPacket, nil
}
