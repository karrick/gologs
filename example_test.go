package gologs

import (
	"fmt"
	"os"
	"strings"
)

func ExampleLogger() {
	// Initialize the logger mode based on the provided command line flags.
	// Create a filtered logger by compiling the log format string.
	log, err := New(os.Stdout, "{message}")
	if err != nil {
		panic(err)
	}
	log.SetAdmin()
	log.Admin("Starting program")
	log.Dev("something important to developers...")

	a := &Alpha{Log: NewBranchWithPrefix(log, "[ALPHA] ").SetAdmin()}
	a.run([]string{"one", "@two", "three", "@four"})

	// Output:
	// Starting program
	// [ALPHA] Starting module
	// [ALPHA] [arg=@two] handling request: @two
	// [ALPHA] [arg=@four] handling request: @four
}

type Alpha struct {
	Log *Logger
	// other fields...
}

func (a *Alpha) run(args []string) {
	a.Log.Admin("Starting module")
	for _, arg := range args {
		// Create a request instance with its own logger.
		request := &Request{
			Log:   a.Log, // Usually a request can be logged at same level as module.
			Query: arg,
		}
		if strings.HasPrefix(arg, "@") {
			// For demonstration purposes, let's arbitrarily cause some of the
			// events to be logged with tracers.
			request.Log = NewTracer(request.Log, fmt.Sprintf("[arg=%s] ", arg))
		}
		request.Handle()
	}
}

// Request is a demonstration structure that has its own logger, which it uses
// to log all events relating to handling this request.
type Request struct {
	Log   *Logger // Log is the logger for this particular request.
	Query string  // Query is the request payload.
}

func (r *Request) Handle() {
	// Anywhere in the call flow for the request, if it wants to log something,
	// it should log to the Request's logger.
	r.Log.Dev("handling request: %v", r.Query)
}
