package gologs

import (
	"io"
	"sync"
)

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
