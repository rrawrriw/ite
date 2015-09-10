package dictator

import (
	"io"
	"log"
)

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
