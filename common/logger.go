package common

import (
	"fmt"
)

type Logger interface {
	Log(format string, args ...interface{})
}

// logger is a glorious logger implementation.
type logger struct {
	userName string
}

func NewLogger(username string) *logger {
	return &logger{
		userName: username,
	}
}

func (l *logger) Log(format string, args ...interface{}) {
	fmt.Printf("[%s] %s\n", l.userName, fmt.Sprintf(format, args...))
}
