package gologs

import (
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
	var e *Event
	switch Level(atomic.LoadUint32((*uint32)(&w.event.level))) {
	case Debug:
		e = w.event.debug()
	case Verbose:
		e = w.event.verbose()
	case Info:
		e = w.event.info()
	case Warning:
		e = w.event.warning()
	case Error:
		e = w.event.error()
	}
	if e == nil {
		return len(buf), nil
	}
	err := e.Msg(string(buf))
	if err != nil {
		return 0, err
	}
	return len(buf), nil
}
