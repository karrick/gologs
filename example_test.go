package gologs

import (
	"os"
)

func ExampleLogger() {
	// Initialize the logger mode based on the provided command line flags.
	// Create a filtered logger by compiling the log format string.
	log := New(os.Stdout)
	log.SetVerbose()
	log.Verbose().Msg("starting program")
	log.Debug().Msg("something important to developers...")

	a := &Alpha{Log: log.NewBranchWithString("module", "alpha").SetVerbose()}
	a.run([]string{"one", "@two", "three", "@four"})

	// Output:
	// {"level":"verbose","message":"starting program"}
	// {"level":"verbose","module":"alpha","message":"starting module"}
}

type Alpha struct {
	Log *Logger
	// other fields...
}

func (a *Alpha) run(args []string) {
	a.Log.Verbose().Msg("starting module")
	for _, arg := range args {
		// Create a request instance with its own logger.
		request := &Request{
			Log:   a.Log, // Usually a request can be logged at same level as module.
			Query: arg,
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
	r.Log.Debug().String("query", r.Query).Msg("handling request")
}
