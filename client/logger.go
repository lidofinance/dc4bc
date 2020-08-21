package client

import (
	"fmt"
	"log"
)

type logger struct {
	userName string
}

func newLogger(username string) *logger {
	return &logger{
		userName: username,
	}
}

func (l *logger) Log(format string, args ...interface{}) {
	log.Printf("[%s] %s\n", l.userName, fmt.Sprintf(format, args...))
}
