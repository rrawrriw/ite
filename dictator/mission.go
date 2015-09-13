package dictator

type MissionSpecs struct {
	// Funktion sollte eine goroutine starten damit der UDP Listener
	// weiterhin erreichbach ist.
	// Die Funktion sollte sich beenden wenn der AppContext Done Channel
	// geschlossen wird oder wenn der Suicide Channel angesprochen wird.
	Mission       func(NodeContext)
	CommandRouter CommandRouter
	// An diesen Channel werden alle CommandResponse Paket weitergeleitet
	// Wird im allgemeien in der Mission Funktionen verwendet.
	ResponseChan chan DictatorPayload
}

type CommandHandler func(NodeContext) error

type CommandRouter map[string]CommandHandler

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
