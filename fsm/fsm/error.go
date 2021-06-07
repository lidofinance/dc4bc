package fsm

import (
	"encoding/json"
	"fmt"
)

// ErrorLevel level type
type Level uint32

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	default:
		return "undefined level"
	}
}

type FsmError struct {
	level   Level
	message string
}

func (e *FsmError) Error() string {
	return e.level.String() + ": " + e.message
}

func (e *FsmError) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Level   Level  `json:"level"`
		Message string `json:"message"`
	}{
		Level:   e.level,
		Message: e.message,
	})
}

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
)

func NewErr(level Level, message string) *FsmError {
	return &FsmError{
		level:   level,
		message: message,
	}
}

func NewErrf(level Level, format string, values ...interface{}) *FsmError {
	if len(values) == 0 {
		return &FsmError{
			level:   level,
			message: format,
		}
	} else {
		return &FsmError{
			level:   level,
			message: fmt.Sprintf(format, values...),
		}
	}
}
