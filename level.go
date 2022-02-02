package gologs

import "fmt"

// Level type defines one of several possible log levels.
type Level uint32

const (
	// Debug is for events that might help a person understand the cause of a
	// bug in a program.
	Debug Level = iota

	// Verbose is for events that might help a person understand the state of a
	// program.
	Verbose

	// Info is for events that annotate high level status of a program.
	Info

	// Warning is for events that indicate a possible problem with the
	// program. Warning events should be investigated and corrected soon.
	Warning

	// Error is for events that indicate a definite problem that might prevent
	// normal program execution. Error events should be corrected immediately.
	Error
)

func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Verbose:
		return "VERBOSE"
	case Info:
		return "INFO"
	case Warning:
		return "WARNING"
	case Error:
		return "ERROR"
	}
	// NOT REACHED
	panic(fmt.Sprintf("invalid log level: %d", uint32(l)))
}
