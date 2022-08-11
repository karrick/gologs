package gologs

import (
	"errors"
	"fmt"
	"sync/atomic"
)

type Writer struct {
	event Event
}

// Write creates a log event at the previously configured log Level and writes
// it to the underlying io.Writer.
//
// On success, it returns the length of buf and nil error. Note that the
// number of bytes it wrote to the underlying io.Writer will always be longer
// than the number of bytes it receives in buf.
//
// On failure, it returns the number of bytes it wrote to the underlying
// io.Writer, which may in fact be longer than the length of buf, along with
// the write error it received.
func (w *Writer) Write(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	w.event.mutex.Lock()

	var isWriting bool

	// Using defer here to prevent holding lock if underlying io.Writer
	// panics.
	defer func() {
		// When isWriting, there is no point in trying to log the failure to
		// write. The other potential cause of a panic is if the user supplied
		// time formatter panics. When that happens, then do log the
		// formatting error to this log.
		if r := recover(); r != nil && !isWriting {
			var err error
			switch t := r.(type) {
			case error:
				err = t
			case string:
				err = errors.New(t)
			default:
				err = fmt.Errorf("%v", t)
			}
			w.event.scratch = w.event.scratch[:1] // erase all but prefix '{'
			w.event.Err(err).Msg("panic when time formatter invoked")
		}

		w.event.scratch = w.event.scratch[:1] // erase all but prefix '{'
		w.event.mutex.Unlock()
	}()

	if w.event.timeFormatter != nil {
		w.event.scratch = w.event.timeFormatter(w.event.scratch)
	}

	// This log level affects at what level these events are written as,
	// rather than acting as a gate to determine when it may log.
	switch Level(atomic.LoadUint32((*uint32)(&w.event.level))) {
	case Debug:
		w.event.scratch = append(w.event.scratch, []byte("\"level\":\"debug\",")...)
	case Verbose:
		w.event.scratch = append(w.event.scratch, []byte("\"level\":\"verbose\",")...)
	case Info:
		w.event.scratch = append(w.event.scratch, []byte("\"level\":\"info\",")...)
	case Warning:
		w.event.scratch = append(w.event.scratch, []byte("\"level\":\"warning\",")...)
	case Error:
		w.event.scratch = append(w.event.scratch, []byte("\"level\":\"error\",")...)
	default:
		w.event.scratch = append(w.event.scratch, []byte("\"level\":\"unknown\",")...)
	}

	if w.event.branch != nil {
		w.event.scratch = append(w.event.scratch, w.event.branch...)
	}

	w.event.scratch = append(w.event.scratch, []byte(`"message":`)...)
	w.event.scratch = appendEncodedJSONFromString(w.event.scratch, string(buf))
	w.event.scratch = append(w.event.scratch, []byte{'}', '\n'}...)

	isWriting = true
	n, err := w.event.output.Write(w.event.scratch)
	if err != nil {
		return n, err // ??? n could be longer than len(buf), but shorter than len(w.events.scratch)
	}

	// NOTE: Even though this wrote an entire log line, which is longer than
	// the buffer it received, the caller expects this to return the number of
	// bytes it wrote to this method.
	return len(buf), nil
}
