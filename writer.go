package gologs

import (
	"io"
	"sync"
)

// mutexWriter ensures only one Write call goes to underlying writer at any given moment.
type mutexWriter struct {
	w io.Writer
	l sync.Mutex
}

func (mw *mutexWriter) Write(p []byte) (int, error) {
	mw.l.Lock()
	n, err := mw.w.Write(p)
	mw.l.Unlock()
	return n, err
}
