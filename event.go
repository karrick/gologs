package gologs

import (
	"sync"
	"sync/atomic"
)

// Event is an in progress log event being formatted before it is written upon
// calling its Msg() method. Callers never need to create an Event
// specifically, but rather receive an Event from calling Debug(), Verbose(),
// Info(), Warning(), or Error() methods of Logger instance.
type Event struct {
	branch        []byte // branch holds prefix of each log event
	scratch       []byte // scratch is where new log events are built
	timeFormatter func([]byte) []byte
	output        *output
	mutex         sync.RWMutex
	level         uint32
	isTracer      uint32
}

// Only called by Logger, but logic is here because event needs to manage its
// locks.
func (event *Event) newIntermediate() *Intermediate {
	event.mutex.RLock()

	il := &Intermediate{
		branch:        make([]byte, len(event.branch), cap(event.branch)),
		timeFormatter: event.timeFormatter,
		output:        event.output,
		level:         atomic.LoadUint32((*uint32)(&event.level)),
	}
	copy(il.branch, event.branch)

	event.mutex.RUnlock()
	return il
}

// Bool encodes a boolean property value to the Event using the specified
// name.
func (event *Event) Bool(name string, value bool) *Event {
	if event == nil {
		return nil
	}
	event.scratch = appendBool(event.scratch, name, value)
	return event
}

// Err encodes a possibly nil error property value to the Event. When err is
// nil, the error value is represented as a JSON null.
func (event *Event) Err(err error) *Event {
	if event == nil {
		return nil
	}
	if err != nil {
		event.scratch = append(event.scratch, []byte(`"error":`)...)
		event.scratch = appendEncodedJSONFromString(event.scratch, err.Error())
		event.scratch = append(event.scratch, ',')
	} else {
		event.scratch = append(event.scratch, []byte(`"error":null,`)...)
	}
	return event
}

// Float encodes a float64 property value to the Event using the specified
// name.
func (event *Event) Float(name string, value float64) *Event {
	if event == nil {
		return nil
	}
	event.scratch = appendFloat(event.scratch, name, value)
	return event
}

// Format encodes a string property value--formatting it with the provided
// arguments--to the Event using the specified name. This function will invoke
// fmt.Sprintf() function to format the formatting string with the provided
// arguments, allocating memory to do so. If no formatting is required,
// invoking Event.String(string, string) will be faster.
func (event *Event) Format(name, f string, args ...interface{}) *Event {
	if event == nil {
		return nil
	}
	event.scratch = appendFormat(event.scratch, name, f, args...)
	return event
}

// Int encodes a int property value to the Event using the specified name.
func (event *Event) Int(name string, value int) *Event {
	if event == nil {
		return nil
	}
	event.scratch = appendInt(event.scratch, name, int64(value))
	return event
}

// Int64 encodes a int64 property value to the Event using the specified name.
func (event *Event) Int64(name string, value int64) *Event {
	if event == nil {
		return nil
	}
	event.scratch = appendInt(event.scratch, name, value)
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

	// Using defer here to prevent holding lock if underlying io.Writer
	// panics.
	defer func() {
		// NOTE: There is nothing to be done to report problem to caller when
		// cannot invoke the provided io.Writer.
		event.scratch = event.scratch[:1] // erase all but prefix '{'
		event.mutex.Unlock()
	}()

	if s != "" {
		event.scratch = append(event.scratch, []byte(`"message":`)...)
		event.scratch = appendEncodedJSONFromString(event.scratch, s)
		event.scratch = append(event.scratch, []byte{'}', '\n'}...)
	} else {
		event.scratch[len(event.scratch)-1] = '}' // Overwrite final comma with close curly brace.
		event.scratch = append(event.scratch, '\n')
	}

	_, err := event.output.Write(event.scratch)
	return err
}

// String encodes a string property value to the Event using the specified
// name.
func (event *Event) String(name, value string) *Event {
	if event == nil {
		return nil
	}
	event.scratch = appendString(event.scratch, name, value)
	return event
}

// Uint encodes a uint property value to the Event using the specified name.
func (event *Event) Uint(name string, value uint) *Event {
	if event == nil {
		return nil
	}
	event.scratch = appendUint(event.scratch, name, uint64(value))
	return event
}

// Uint64 encodes a uint64 property value to the Event using the specified
// name.
func (event *Event) Uint64(name string, value uint64) *Event {
	if event == nil {
		return nil
	}
	event.scratch = appendUint(event.scratch, name, value)
	return event
}
