package gologs

import (
	"sync/atomic"
)

// Writer is an io.Writer that conveys all writes it receives to the
// underlying io.Writer as individual log events.
type Writer struct {
	event     Event
	branch    []byte // branch holds potentially empty prefix of each log event
	emitLevel Level  // emitLevel is the level events will always be emitted as
	level     uint32 // level is the current log level of this Writer
}

// SetLevel changes the Writer's level to the specified Level without
// blocking. This causes all writes to the Writer to be logged with the
// specified Level.
func (w *Writer) SetLevel(level Level) *Writer {
	atomic.StoreUint32((*uint32)(&w.level), uint32(level))
	return w
}

// SetDebug changes the Writer's level to Debug, which causes all writes to
// the Writer to be logged to the underlying Logger with a level of Debug. The
// change is made without blocking.
func (w *Writer) SetDebug() *Writer {
	atomic.StoreUint32((*uint32)(&w.level), uint32(Debug))
	return w
}

// SetVerbose changes the Writer's level to Verbose, which causes all writes
// to the Writer to be logged to the underlying Logger with a level of
// Verbose. The change is made without blocking.
func (w *Writer) SetVerbose() *Writer {
	atomic.StoreUint32((*uint32)(&w.level), uint32(Verbose))
	return w
}

// SetInfo changes the Writer's level to Info, which causes all writes to the
// Writer to be logged to the underlying Logger with a level of Info. The
// change is made without blocking.
func (w *Writer) SetInfo() *Writer {
	atomic.StoreUint32((*uint32)(&w.level), uint32(Info))
	return w
}

// SetWarning changes the Writer's level to Warning, which causes all writes
// to the Writer to be logged to the underlying Logger with a level of
// Warning. The change is made without blocking.
func (w *Writer) SetWarning() *Writer {
	atomic.StoreUint32((*uint32)(&w.level), uint32(Warning))
	return w
}

// SetError changes the Writer's level to Error, which causes all writes to
// the Writer to be logged to the underlying Logger with a level of Error. The
// change is made without blocking.
func (w *Writer) SetError() *Writer {
	atomic.StoreUint32((*uint32)(&w.level), uint32(Error))
	return w
}

// Write creates and emits a log event with its message set to the text of buf
// and at the log level with which it was instantiated.
//
// On success, it returns the length of buf and nil error. Note that the
// number of bytes it wrote to the underlying io.Writer will always be longer
// than the number of bytes it receives in buf.
//
// On failure, it returns 0 for the number of bytes it wrote to the underlying
// io.Writer along with the write error.
func (w *Writer) Write(buf []byte) (int, error) {
	if Level(atomic.LoadUint32((*uint32)(&w.level))) > w.emitLevel {
		return len(buf), nil
	}

	var e *Event

	switch w.emitLevel {
	case Debug:
		e = w.event.debug(w.branch)
	case Verbose:
		e = w.event.verbose(w.branch)
	case Info:
		e = w.event.info(w.branch)
	case Warning:
		e = w.event.warning(w.branch)
	default:
		e = w.event.error(w.branch)
	}
	if err := e.Msg(string(buf)); err != nil {
		return 0, err
	}
	return len(buf), nil
}
