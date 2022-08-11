package gologs

import (
	"io"
	"sync/atomic"
)

// Logger is a near zero allocation logging mechanism.
type Logger struct {
	// NOTE: Logger is effectively a different type name for Event. Everything
	// is stored in Event to reduce amount of pointer dereferencing needed for
	// the various methods. The reason there is a Log type versus the Event
	// type is to control which methods are available for each. The caller
	// must call one of Log's methods to get an Event, and then call the
	// returned Event's methods to prepare the log Event for serialization.
	event Event
}

// New returns a new Logger that writes messages to the provided io.Writer.
//
// By default, a Logger has a log level of Warning, which is closer to the
// UNIX philosophy of avoiding unnecessary output.
func New(w io.Writer) *Logger {
	log := &Logger{
		event: Event{
			scratch: make([]byte, 1, 4096),
			branch:  make([]byte, 0, 4096),
			output:  &output{w: w},
			level:   uint32(Warning),
		},
	}
	log.event.scratch[0] = '{'
	return log
}

// SetWriter directs all future writes to w, potentially blocking until any in
// progress log event has been written.
func (log *Logger) SetWriter(w io.Writer) *Logger {
	log.event.output.SetWriter(w)
	return log
}

// SetLevel changes the Logger's level to the specified Level without
// blocking.
func (log *Logger) SetLevel(level Level) *Logger {
	log.event.setLevel(level)
	return log
}

// SetDebug changes the Logger's level to Debug, which allows all events to be
// logged. The change is made without blocking.
func (log *Logger) SetDebug() *Logger {
	log.event.setLevel(Debug)
	return log
}

// SetVerbose changes the Logger's level to Verbose, which causes all Debug
// events to be ignored, and all Verbose, Info, Warning, and Error events to
// be logged. The change is made without blocking.
func (log *Logger) SetVerbose() *Logger {
	log.event.setLevel(Verbose)
	return log
}

// SetInfo changes the Logger's level to Info, which causes all Debug and
// Verbose events to be ignored, and all Info, Warning, and Error events to be
// logged. The change is made without blocking. The change is made without
// blocking.
func (log *Logger) SetInfo() *Logger {
	log.event.setLevel(Info)
	return log
}

// SetWarning changes the Logger's level to Warning, which causes all Debug,
// Verbose, and Info events to be ignored, and all Warning, and Error events
// to be logged. The change is made without blocking.
func (log *Logger) SetWarning() *Logger {
	log.event.setLevel(Warning)
	return log
}

// SetError changes the Logger's level to Error, which causes all Debug,
// Verbose, Info, and Warning events to be ignored, and all Error events to be
// logged. The change is made without blocking.
func (log *Logger) SetError() *Logger {
	log.event.setLevel(Error)
	return log
}

// SetTimeFormatter updates the time formatting callback function that is
// invoked for every log message while it is being formatted, potentially
// blocking until any in progress log event has been written.
func (log *Logger) SetTimeFormatter(callback func([]byte) []byte) *Logger {
	log.event.setTimeFormatter(callback)
	return log
}

// SetTracing changes the Logger's tracing to the specified value without
// blocking.
func (log *Logger) SetTracing(enabled bool) *Logger {
	if enabled {
		atomic.StoreUint32((*uint32)(&log.event.isTracer), 1)
	} else {
		atomic.StoreUint32((*uint32)(&log.event.isTracer), 0)
	}
	return log
}

// Debug returns an Event to be formatted and sent to the Logger's underlying
// io.Writer when the Logger's level is Debug. If the Logger's level is above
// Debug, this method returns without blocking.
func (log *Logger) Debug() *Event {
	return log.event.debug()
}

// Verbose returns an Event to be formatted and sent to the Logger's
// underlying io.Writer when the Logger's level is Debug or Verbose. If the
// Logger's level is above Verbose, this method returns without blocking.
func (log *Logger) Verbose() *Event {
	return log.event.verbose()
}

// Info returns an Event to be formatted and sent to the Logger's underlying
// io.Writer when the Logger's level is Debug, Verbose, or Info. If the
// Logger's level is above Info, this method returns without blocking.
func (log *Logger) Info() *Event {
	return log.event.info()
}

// Warning returns an Event to be formatted and sent to the Logger's
// underlying io.Writer when the Logger's level is Debug, Verbose, Info, or
// Warning. If the Logger's level is above Warning, this method returns
// without blocking.
func (log *Logger) Warning() *Event {
	return log.event.warning()
}

// Error returns an Event to be formatted and sent to the Logger's underlying
// io.Writer.
func (log *Logger) Error() *Event {
	return log.event.error()
}

// NewWriter creates an io.Writer that conveys all writes it receives to the
// underlying io.Writer as individual log events.
func (log *Logger) NewWriter() *Writer {
	return log.event.newWriter()
}

// With returns an Intermediate Logger instance that inherits from log, but
// can be modified to add one or more additional properties for every outgoing
// log event. Callers never need to create an Intermediate Logger
// specifically, but rather receive one as a result of invoking this method.
//
// log = log.With().String("s", "value").Bool("b", true).Logger()
func (log *Logger) With() *Intermediate {
	return log.event.newIntermediate()
}
