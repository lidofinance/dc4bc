package client

import (
	"fmt"
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
	fmt.Printf("[%s] %s\n", l.userName, fmt.Sprintf(format, args...))
}
