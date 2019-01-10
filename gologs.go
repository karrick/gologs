package gologs

import (
	"fmt"
	"io"
	"time"
)

// Level type defines one of several possible log levels.
type Level uint32

const (
	Debug   Level = iota // Debug is a low level log level commonly used while debugging software source code.
	Verbose              // Verbose is a low level log level commonly used for detailed software status and runtime information.
	Info                 // Info is a mid level log level commonly used for observing routine software status.
	Warning              // Warning is a high level log level commonly used when only desire to observe unusual but recoverable software failures.
	Error                // Error is the highest log level commonly used when only desire to observe software failures.
)

func (l Level) String() string {
	switch l {
	case Debug:
		return "debug"
	case Verbose:
		return "verbose"
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	}
	panic(fmt.Sprintf("invalid log level: %v", uint32(l)))
}

// Logger is anything that provides basic logging functionality.
type Logger interface {
	Debug(string, ...interface{})   // Debug is used to optionally emit source code level debugging information.
	Verbose(string, ...interface{}) // Verbose is used to optionally emit detailed software status and runtime information.
	Info(string, ...interface{})    // Info used to emit routine software status and runtime information.
	Warning(string, ...interface{}) // Warning is used to emit unusual but recoverable software failures.
	Error(string, ...interface{})   // Error is used to emit non-recoverable software failures.

	SetLevel(Level) // Level is general level setter
	SetQuiet()      // Quiet sets log level to Warning, useful for setting level based on CLI -q flag.
	SetVerbose()    // Verbose sets log level to Verbose, useful for setting level based on CLI -v flag.
}

// NewDefaultLogger returns a Logger that prints to w logs formatted with the
// DefaultLogFormat.
func NewDefaultLogger(w io.Writer) Logger {
	return NewFormattedLogger(w, DefaultLogFormat)
}

type event struct {
	message string
	when    time.Time
	level   Level
}

func newEvent(l Level, format string, a ...interface{}) event {
	return event{
		message: fmt.Sprintf(format, a...),
		when:    time.Now(), // consider lazy filling this
		level:   l,
	}
}
