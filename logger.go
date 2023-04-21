package gologs

import (
	"io"
	"sync"
	"sync/atomic"
)

// Logger is a near zero allocation logging mechanism. Each log event is
// written using a single invocation of the Write method for the underlying
// io.Writer.
type Logger struct {
	event   Event
	branch  []byte       // branch holds potentially empty prefix of each log event
	mutex   sync.RWMutex // mutex for copying branch
	level   uint32
	tracing bool
}

// New returns a new Logger that writes log events to w.
//
// By default, a Logger has a log level of Warning, which is closer to the
// UNIX philosophy of avoiding unnecessary output.
//
//	log := gologs.New(os.Stdout).SetTimeFormatter(gologs.TimeUnix)
func New(w io.Writer) *Logger {
	log := &Logger{
		event: Event{
			scratch: make([]byte, 1, 2048),
			output:  &output{w: w},
		},
		level: uint32(Warning),
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
	atomic.StoreUint32((*uint32)(&log.level), uint32(level))
	return log
}

// SetDebug changes the Logger's level to Debug, which allows all events to be
// logged. The change is made without blocking.
func (log *Logger) SetDebug() *Logger {
	atomic.StoreUint32((*uint32)(&log.level), uint32(Debug))
	return log
}

// SetVerbose changes the Logger's level to Verbose, which causes all Debug
// events to be ignored, and all Verbose, Info, Warning, and Error events to
// be logged. The change is made without blocking.
func (log *Logger) SetVerbose() *Logger {
	atomic.StoreUint32((*uint32)(&log.level), uint32(Verbose))
	return log
}

// SetInfo changes the Logger's level to Info, which causes all Debug and
// Verbose events to be ignored, and all Info, Warning, and Error events to be
// logged. The change is made without blocking. The change is made without
// blocking.
func (log *Logger) SetInfo() *Logger {
	atomic.StoreUint32((*uint32)(&log.level), uint32(Info))
	return log
}

// SetWarning changes the Logger's level to Warning, which causes all Debug,
// Verbose, and Info events to be ignored, and all Warning, and Error events
// to be logged. The change is made without blocking.
func (log *Logger) SetWarning() *Logger {
	atomic.StoreUint32((*uint32)(&log.level), uint32(Warning))
	return log
}

// SetError changes the Logger's level to Error, which causes all Debug,
// Verbose, Info, and Warning events to be ignored, and all Error events to be
// logged. The change is made without blocking.
func (log *Logger) SetError() *Logger {
	atomic.StoreUint32((*uint32)(&log.level), uint32(Error))
	return log
}

// SetTimeFormatter updates the time formatting callback function that is
// invoked for every log message while it is being formatted, potentially
// blocking until any in progress log event has been written.
func (log *Logger) SetTimeFormatter(callback TimeFormatter) *Logger {
	log.event.setTimeFormatter(callback)
	return log
}

// Log returns an Event to be formatted and sent to the Logger's underlying
// io.Writer, regardless of the Logger's log level, and omitting the event log
// level in the output.
func (log *Logger) Log() *Event {
	return log.event.log(log.branch)
}

// Debug returns an Event to be formatted and sent to the Logger's underlying
// io.Writer when the Logger's level is Debug. If the Logger's level is above
// Debug, this method returns without blocking.
func (log *Logger) Debug() *Event {
	if log.tracing || Level(atomic.LoadUint32((*uint32)(&log.level))) <= Debug {
		return log.event.debug(log.branch)
	}
	return nil
}

// Verbose returns an Event to be formatted and sent to the Logger's
// underlying io.Writer when the Logger's level is Debug or Verbose. If the
// Logger's level is above Verbose, this method returns without blocking.
func (log *Logger) Verbose() *Event {
	if log.tracing || Level(atomic.LoadUint32((*uint32)(&log.level))) <= Verbose {
		return log.event.verbose(log.branch)
	}
	return nil
}

// Info returns an Event to be formatted and sent to the Logger's underlying
// io.Writer when the Logger's level is Debug, Verbose, or Info. If the
// Logger's level is above Info, this method returns without blocking.
func (log *Logger) Info() *Event {
	if log.tracing || Level(atomic.LoadUint32((*uint32)(&log.level))) <= Info {
		return log.event.info(log.branch)
	}
	return nil
}

// Warning returns an Event to be formatted and sent to the Logger's
// underlying io.Writer when the Logger's level is Debug, Verbose, Info, or
// Warning. If the Logger's level is above Warning, this method returns
// without blocking.
func (log *Logger) Warning() *Event {
	if log.tracing || Level(atomic.LoadUint32((*uint32)(&log.level))) <= Warning {
		return log.event.warning(log.branch)
	}
	return nil
}

// Error returns an Event to be formatted and sent to the Logger's underlying
// io.Writer.
func (log *Logger) Error() *Event {
	return log.event.error(log.branch)
}

// NewWriter creates an io.Writer that conveys all writes it receives to the
// underlying io.Writer as individual log events.
//
//	func main() {
//	    log := gologs.New(os.Stdout).SetTimeFormatter(gologs.TimeUnix)
//	    lw := log.NewWriter()
//	    scanner := bufio.NewScanner(os.Stdin)
//	    for scanner.Scan() {
//	        _, err := lw.Write(scanner.Bytes())
//	        if err != nil {
//	            fmt.Fprintf(os.Stderr, "%s\n", err)
//	            os.Exit(1)
//	        }
//	    }
//	    if err := scanner.Err(); err != nil {
//	        fmt.Fprintf(os.Stderr, "%s\n", err)
//	        os.Exit(1)
//	    }
//	}
func (log *Logger) NewWriter(level Level) *Writer {
	log.mutex.RLock()

	w := &Writer{
		event: Event{
			scratch:       make([]byte, 1, 2048),
			timeFormatter: log.event.timeFormatter,
			output:        log.event.output,
		},
		emitLevel: level,
		level:     atomic.LoadUint32((*uint32)(&log.level)),
	}
	if len(log.branch) > 0 {
		w.branch = make([]byte, len(log.branch))
		copy(w.branch, log.branch)
	}
	w.event.scratch[0] = '{'

	log.mutex.RUnlock()
	return w
}

// With returns an Intermediate Logger instance that inherits from log, but
// can be modified to add one or more additional properties for every outgoing
// log event.
//
//	log = log.With().String("s", "value").Bool("b", true).Logger()
func (log *Logger) With() *Intermediate {
	log.mutex.RLock()

	il := &Intermediate{
		timeFormatter: log.event.timeFormatter,
		output:        log.event.output,
		level:         atomic.LoadUint32((*uint32)(&log.level)),
	}
	if cap(log.branch) > 0 {
		if len(log.branch) > 0 {
			il.branch = make([]byte, len(log.branch), cap(log.branch))
			copy(il.branch, log.branch)
		} else {
			il.branch = make([]byte, 0, cap(log.branch))
		}
	}

	log.mutex.RUnlock()
	return il
}
