package gologs

import (
	"errors"
	"fmt"
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
	atomic.StoreUint32((*uint32)(&log.event.level), uint32(level))
	return log
}

// SetDebug changes the Logger's level to Debug, which allows all events to be
// logged. The change is made without blocking.
func (log *Logger) SetDebug() *Logger {
	atomic.StoreUint32((*uint32)(&log.event.level), uint32(Debug))
	return log
}

// SetVerbose changes the Logger's level to Verbose, which causes all Debug
// events to be ignored, and all Verbose, Info, Warning, and Error events to
// be logged. The change is made without blocking.
func (log *Logger) SetVerbose() *Logger {
	atomic.StoreUint32((*uint32)(&log.event.level), uint32(Verbose))
	return log
}

// SetInfo changes the Logger's level to Info, which causes all Debug and
// Verbose events to be ignored, and all Info, Warning, and Error events to be
// logged. The change is made without blocking. The change is made without
// blocking.
func (log *Logger) SetInfo() *Logger {
	atomic.StoreUint32((*uint32)(&log.event.level), uint32(Info))
	return log
}

// SetWarning changes the Logger's level to Warning, which causes all Debug,
// Verbose, and Info events to be ignored, and all Warning, and Error events
// to be logged. The change is made without blocking.
func (log *Logger) SetWarning() *Logger {
	atomic.StoreUint32((*uint32)(&log.event.level), uint32(Warning))
	return log
}

// SetError changes the Logger's level to Error, which causes all Debug,
// Verbose, Info, and Warning events to be ignored, and all Error events to be
// logged. The change is made without blocking.
func (log *Logger) SetError() *Logger {
	atomic.StoreUint32((*uint32)(&log.event.level), uint32(Error))
	return log
}

// SetTimeFormatter updates the time formatting callback function that is
// invoked for every log message while it is being formatted, potentially
// blocking until any in progress log event has been written.
func (log *Logger) SetTimeFormatter(callback func([]byte) []byte) *Logger {
	log.event.mutex.Lock()
	log.event.timeFormatter = callback
	log.event.mutex.Unlock()
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

// formatTimePanics attempts to format the time using the stored time
// formatting callback function. When the function does not panic, it returns
// false. When the function does panic, it returns true so the Logger method
// can stop processing the provided event.
func (log *Logger) formatTimePanics() (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			var err error
			switch t := r.(type) {
			case error:
				err = t
			case string:
				err = errors.New(t)
			default:
				err = fmt.Errorf("%v", t)
			}
			log.event.scratch = log.event.scratch[:1] // erase all but prefix '{'
			log.event.Err(err).Msg("panic when time formatter invoked")
			panicked = true
		}
	}()
	log.event.scratch = log.event.timeFormatter(log.event.scratch)
	return
}

// Debug returns an Event to be formatted and sent to the Logger's underlying
// io.Writer when the Logger's level is Debug. If the Logger's level is above
// Debug, this method returns without blocking.
func (log *Logger) Debug() *Event {
	if Level(atomic.LoadUint32((*uint32)(&log.event.level))) > Debug &&
		atomic.LoadUint32((*uint32)(&log.event.isTracer)) == 0 {
		return nil
	}
	log.event.mutex.Lock() // unlocked inside Event.Msg()
	if log.event.timeFormatter != nil && log.formatTimePanics() {
		return nil
	}
	log.event.scratch = append(log.event.scratch, []byte("\"level\":\"debug\",")...)
	if log.event.branch != nil {
		log.event.scratch = append(log.event.scratch, log.event.branch...)
	}
	return &log.event
}

// Verbose returns an Event to be formatted and sent to the Logger's
// underlying io.Writer when the Logger's level is Debug or Verbose. If the
// Logger's level is above Verbose, this method returns without blocking.
func (log *Logger) Verbose() *Event {
	if Level(atomic.LoadUint32((*uint32)(&log.event.level))) > Verbose &&
		atomic.LoadUint32((*uint32)(&log.event.isTracer)) == 0 {
		return nil
	}
	log.event.mutex.Lock() // unlocked inside Event.Msg()
	if log.event.timeFormatter != nil && log.formatTimePanics() {
		return nil
	}
	log.event.scratch = append(log.event.scratch, []byte("\"level\":\"verbose\",")...)
	if log.event.branch != nil {
		log.event.scratch = append(log.event.scratch, log.event.branch...)
	}
	return &log.event
}

// Info returns an Event to be formatted and sent to the Logger's underlying
// io.Writer when the Logger's level is Debug, Verbose, or Info. If the
// Logger's level is above Info, this method returns without blocking.
func (log *Logger) Info() *Event {
	if Level(atomic.LoadUint32((*uint32)(&log.event.level))) > Info &&
		atomic.LoadUint32((*uint32)(&log.event.isTracer)) == 0 {
		return nil
	}
	log.event.mutex.Lock() // unlocked inside Event.Msg()
	if log.event.timeFormatter != nil && log.formatTimePanics() {
		return nil
	}
	log.event.scratch = append(log.event.scratch, []byte("\"level\":\"info\",")...)
	if log.event.branch != nil {
		log.event.scratch = append(log.event.scratch, log.event.branch...)
	}
	return &log.event
}

// Warning returns an Event to be formatted and sent to the Logger's
// underlying io.Writer when the Logger's level is Debug, Verbose, Info, or
// Warning. If the Logger's level is above Warning, this method returns
// without blocking.
func (log *Logger) Warning() *Event {
	if Level(atomic.LoadUint32((*uint32)(&log.event.level))) > Warning &&
		atomic.LoadUint32((*uint32)(&log.event.isTracer)) == 0 {
		return nil
	}
	log.event.mutex.Lock() // unlocked inside Event.Msg()
	if log.event.timeFormatter != nil && log.formatTimePanics() {
		return nil
	}
	log.event.scratch = append(log.event.scratch, []byte("\"level\":\"warning\",")...)
	if log.event.branch != nil {
		log.event.scratch = append(log.event.scratch, log.event.branch...)
	}
	return &log.event
}

// Error returns an Event to be formatted and sent to the Logger's underlying
// io.Writer.
func (log *Logger) Error() *Event {
	log.event.mutex.Lock() // unlocked inside Event.Msg()
	if log.event.timeFormatter != nil && log.formatTimePanics() {
		return nil
	}
	log.event.scratch = append(log.event.scratch, []byte("\"level\":\"error\",")...)
	if log.event.branch != nil {
		log.event.scratch = append(log.event.scratch, log.event.branch...)
	}
	return &log.event
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
