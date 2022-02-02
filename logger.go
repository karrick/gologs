package gologs

import (
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
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
func New(w io.Writer) *Logger {
	log := &Logger{
		event: Event{
			buf:   make([]byte, 1, 4096),
			o:     &output{w: w},
			level: uint32(Warning),
		},
	}
	log.event.buf[0] = '{'
	return log
}

// NewBranch returns a new Logger that writes to the same underlying io.Writer
// as the original log, but may have a potentially different log Level. To
// effectively use, consider setting the parent log level to Debug so all log
// messages are written to the underlying io.Writer, but then each child
// branch be given a more restrictive log level.
func (log *Logger) NewBranch() *Logger {
	log.event.mutex.Lock()

	child := &Logger{
		event: Event{
			buf:    make([]byte, 1, 4096),
			branch: log.event.branch,
			when:   log.event.when,
			o:      log.event.o,
			level:  atomic.LoadUint32((*uint32)(&log.event.level)),
		},
	}

	child.event.buf[0] = '{'

	log.event.mutex.Unlock()
	return child
}

// NewBranchWithString returns a new Logger that writes to the same underlying
// io.Writer as the original log, but may have a potentially different log
// Level than the parent. To effectively use, consider setting the parent log
// level to Debug so all log messages are written to the underlying io.Writer,
// but then each child branch be given a more restrictive log level.
//
// Logger events will include the property created by formatting the name and
// string, appended to whatever branch values the parent Logger might have. a
// known limitation is when the branch name is the same as an ancestor's
// branch name, it does not replace the original name, but will rather create
// invalid JSON objects for Logger events.
func (log *Logger) NewBranchWithString(name, value string) *Logger {
	child := log.NewBranch()

	child.event.branch = appendEncodedJSONFromString(child.event.branch, name)
	child.event.branch = append(child.event.branch, ':')
	child.event.branch = appendEncodedJSONFromString(child.event.branch, value)
	child.event.branch = append(child.event.branch, ',')
	child.event.branch = child.event.branch[:len(child.event.branch)]

	return child
}

// SetWriter directs all future writes to the specified io.Writer, potentially
// blocking until any in progress event is being written.
func (log *Logger) SetWriter(w io.Writer) *Logger {
	log.event.o.SetWriter(w)
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

// SetVerbose changes the Logger's level to Verbose, which causes all Debug events
// to be ignored, and all Verbose, Info, Warning, and Error events to be
// logged. The change is made without blocking.
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
// blocking until any in progress event is being written.
func (log *Logger) SetTimeFormatter(callback func([]byte) []byte) *Logger {
	log.event.mutex.Lock()
	log.event.when = callback
	log.event.mutex.Unlock()
	return log
}

// Debug returns an Event to be formatted and sent to the Logger's underlying
// io.Writer when the Logger's level is Debug. If the Logger's level is above
// Debug, this method returns without blocking.
func (log *Logger) Debug() *Event {
	if Level(atomic.LoadUint32((*uint32)(&log.event.level))) > Debug {
		return nil
	}
	log.event.mutex.Lock() // unlocked inside Event.Msg()
	if log.event.when != nil {
		log.event.buf = log.event.when(log.event.buf)
	}
	log.event.buf = append(log.event.buf, []byte("\"level\":\"debug\",")...)
	if log.event.branch != nil {
		log.event.buf = append(log.event.buf, log.event.branch...)
	}
	return &log.event
}

// Verbose returns an Event to be formatted and sent to the Logger's
// underlying io.Writer when the Logger's level is Debug or Verbose. If the
// Logger's level is above Verbose, this method returns without blocking.
func (log *Logger) Verbose() *Event {
	if Level(atomic.LoadUint32((*uint32)(&log.event.level))) > Verbose {
		return nil
	}
	log.event.mutex.Lock() // unlocked inside Event.Msg()
	if log.event.when != nil {
		log.event.buf = log.event.when(log.event.buf)
	}
	log.event.buf = append(log.event.buf, []byte("\"level\":\"verbose\",")...)
	if log.event.branch != nil {
		log.event.buf = append(log.event.buf, log.event.branch...)
	}
	return &log.event
}

// Info returns an Event to be formatted and sent to the Logger's underlying
// io.Writer when the Logger's level is Debug, Verbose, or Info. If the
// Logger's level is above Info, this method returns without blocking.
func (log *Logger) Info() *Event {
	if Level(atomic.LoadUint32((*uint32)(&log.event.level))) > Info {
		return nil
	}
	log.event.mutex.Lock() // unlocked inside Event.Msg()
	if log.event.when != nil {
		log.event.buf = log.event.when(log.event.buf)
	}
	log.event.buf = append(log.event.buf, []byte("\"level\":\"info\",")...)
	if log.event.branch != nil {
		log.event.buf = append(log.event.buf, log.event.branch...)
	}
	return &log.event
}

// Warning returns an Event to be formatted and sent to the Logger's
// underlying io.Writer when the Logger's level is Debug, Verbose, Info, or
// Warning. If the Logger's level is above Warning, this method returns
// without blocking.
func (log *Logger) Warning() *Event {
	if Level(atomic.LoadUint32((*uint32)(&log.event.level))) > Warning {
		return nil
	}
	log.event.mutex.Lock() // unlocked inside Event.Msg()
	if log.event.when != nil {
		log.event.buf = log.event.when(log.event.buf)
	}
	log.event.buf = append(log.event.buf, []byte("\"level\":\"warning\",")...)
	if log.event.branch != nil {
		log.event.buf = append(log.event.buf, log.event.branch...)
	}
	return &log.event
}

// Error returns an Event to be formatted and sent to the Logger's underlying
// io.Writer.
func (log *Logger) Error() *Event {
	if Level(atomic.LoadUint32((*uint32)(&log.event.level))) > Error {
		return nil
	}
	log.event.mutex.Lock() // unlocked inside Event.Msg()
	if log.event.when != nil {
		log.event.buf = log.event.when(log.event.buf)
	}
	log.event.buf = append(log.event.buf, []byte("\"level\":\"error\",")...)
	if log.event.branch != nil {
		log.event.buf = append(log.event.buf, log.event.branch...)
	}
	return &log.event
}

// Event is an in progress log event being formatted before it is written upon
// calling its Msg() method. Callers never need to create an Event
// specifically, but rather receive an Event from calling Debug(), Verbose(),
// Info(), Warning(), or Error() methods of Logger instance.
type Event struct {
	buf    []byte
	branch []byte
	when   func([]byte) []byte
	o      *output
	mutex  sync.Mutex
	level  uint32
}

// Bool encodes a boolean property value to the Event using the specified
// name.
func (event *Event) Bool(name string, value bool) *Event {
	if event == nil {
		return nil
	}
	event.buf = appendEncodedJSONFromString(event.buf, name)
	event.buf = append(event.buf, ':')
	if value {
		event.buf = append(event.buf, []byte("true,")...)
	} else {
		event.buf = append(event.buf, []byte("false,")...)
	}
	return event
}

// Err encodes a possibly nil error property value to the Event. When err is
// nil, the error value is represented as a JSON null.
func (event *Event) Err(err error) *Event {
	if event == nil {
		return nil
	}
	if err != nil {
		event.buf = append(event.buf, []byte(`"error":`)...)
		event.buf = appendEncodedJSONFromString(event.buf, err.Error())
		event.buf = append(event.buf, ',')
	} else {
		event.buf = append(event.buf, []byte(`"error":null,`)...)
	}
	return event
}

// Float encodes a float64 property value to the Event using the specified
// name.
func (event *Event) Float(name string, value float64) *Event {
	if event == nil {
		return nil
	}
	event.buf = appendEncodedJSONFromString(event.buf, name)
	event.buf = append(event.buf, ':')
	event.buf = appendEncodedJSONFromFloat(event.buf, value)
	event.buf = append(event.buf, ',')
	return event
}

// Format encodes a string property value--formatting it with the provided
// arguments--to the Event using the specified name. This function will invoke
// fmt.Sprintf() function to format the formatting string with the provided
// arguments, allocating memory to do so. If no formatting is required,
// invoking Event.String(string) will be faster.
func (event *Event) Format(name, f string, args ...interface{}) *Event {
	if event == nil {
		return nil
	}
	value := fmt.Sprintf(f, args...) // must allocate when caller passes formatting string
	event.buf = appendEncodedJSONFromString(event.buf, name)
	event.buf = append(event.buf, ':')
	event.buf = appendEncodedJSONFromString(event.buf, value)
	event.buf = append(event.buf, ',')
	return event
}

// Int encodes a int property value to the Event using the specified name.
func (event *Event) Int(name string, value int) *Event {
	if event == nil {
		return nil
	}
	event.buf = appendEncodedJSONFromString(event.buf, name)
	event.buf = append(event.buf, ':')
	event.buf = strconv.AppendInt(event.buf, int64(value), 10)
	event.buf = append(event.buf, ',')
	return event
}

// Int64 encodes a int64 property value to the Event using the specified name.
func (event *Event) Int64(name string, value int64) *Event {
	if event == nil {
		return nil
	}
	event.buf = appendEncodedJSONFromString(event.buf, name)
	event.buf = append(event.buf, ':')
	event.buf = strconv.AppendInt(event.buf, value, 10)
	event.buf = append(event.buf, ',')
	return event
}

// String encodes a string property value to the Event using the specified
// name.
func (event *Event) String(name, value string) *Event {
	if event == nil {
		return nil
	}
	event.buf = appendEncodedJSONFromString(event.buf, name)
	event.buf = append(event.buf, ':')
	event.buf = appendEncodedJSONFromString(event.buf, value)
	event.buf = append(event.buf, ',')
	return event
}

// Msg adds the specified message to the Event for the message property, and
// writes the Event to Logger's io.Writer. The caller may provide an empty
// string, which will elide inclusion of the message property in the written
// log event. This method must be invoked to complete every Event. This method
// returns any error from attempting to write to the Logger's io.Writer.
func (event *Event) Msg(s string) error {
	if event == nil {
		return nil
	}

	if s != "" {
		event.buf = append(event.buf, []byte(`"message":`)...)
		event.buf = appendEncodedJSONFromString(event.buf, s)
		event.buf = append(event.buf, ',')
	}

	event.buf[len(event.buf)-1] = '}'   // Overwrite final comma with close curly brace.
	event.buf = append(event.buf, '\n') // Append newline.
	_, err := event.o.Write(event.buf)
	event.buf = event.buf[:1] // Clear everything after initial open curly brace.

	event.mutex.Unlock()
	return err
}

// TimeUnix appends the current Unix second time to buf as a JSON property
// name and value.
func TimeUnix(buf []byte) []byte {
	buf = append(buf, []byte(`"time":`)...)
	buf = strconv.AppendInt(buf, time.Now().Unix(), 10)
	return append(buf, ',')
}

// TimeUnix appends the current Unix millisecond time to buf as a JSON
// property name and value.
func TimeUnixMilli(buf []byte) []byte {
	buf = append(buf, []byte(`"time":`)...)
	buf = strconv.AppendInt(buf, time.Now().UnixMilli(), 10)
	return append(buf, ',')
}

// TimeUnix appends the current Unix microsecond time to buf as a JSON
// property name and value.
func TimeUnixMicro(buf []byte) []byte {
	buf = append(buf, []byte(`"time":`)...)
	buf = strconv.AppendInt(buf, time.Now().UnixMicro(), 10)
	return append(buf, ',')
}

// TimeUnix appends the current Unix nanosecond time to buf as a JSON property
// name and value.
func TimeUnixNano(buf []byte) []byte {
	buf = append(buf, []byte(`"time":`)...)
	buf = strconv.AppendInt(buf, time.Now().UnixNano(), 10)
	return append(buf, ',')
}

// output merely ensures only a single Write is invoked at once.
type output struct {
	w     io.Writer
	mutex sync.Mutex
}

// SetWriter directs all future writes to the specified io.Writer, potentially
// blocking until any in progress event is being written.
func (o *output) SetWriter(w io.Writer) {
	o.mutex.Lock()
	o.w = w
	o.mutex.Unlock()
}

// Write writes buf to the underlying io.Writer, potentially blocking until
// any in progress event is being written.
func (o *output) Write(buf []byte) (int, error) {
	o.mutex.Lock()

	// Using defer here to prevent holding lock if underlying io.Writer
	// panics.
	defer o.mutex.Unlock()

	return o.w.Write(buf)
}
