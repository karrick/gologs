package gologs

import (
	"fmt"
	"os"
	"strings"
)

func ExampleLogger() {
	// A Logger needs a io.Writer to which it will write all log messages.
	log := New(os.Stdout)

	// By default, a Logger has a log level of Warning, which is closer to the
	// UNIX philosophy of avoiding unnecessary output. This example is
	// intended to be more verbose for demonstrative purposes.
	log.SetVerbose()
	// log.SetDebug()

	log.Verbose().Msg("initializing program")

	// When creating structure instances, consider sending the log instance to
	// the structure's constructor so it can prefix its log messages
	// accordingly. This is especially useful when the instantiated structure
	// might spin off goroutines to perform tasks.
	s := NewServer(log)

	s.run([]string{"one=1", "@two=2", "three=3", "@four=4"})

	// Output:
	// {"level":"verbose","message":"initializing program"}
	// {"level":"verbose","message":"Enter NewServer()"}
	// {"level":"verbose","structure":"Server","message":"Enter Server.run()"}
	// {"level":"verbose","structure":"Server","method":"run","message":"starting loop"}
}

type Server struct {
	log *Logger // log is used for all log output by Server
	// plus any other fields...
}

func NewServer(log *Logger) *Server {
	// Sometimes it is helpful to know when enter and leave a function.
	log.Verbose().Msg("Enter NewServer()")

	// However, when deferring a log entry, ensure to call it from a composed
	// function.
	defer func() { log.Debug().Msg("Leave NewServer()") }()

	// When creating a new runtime subsystem, create a new branch of the
	// provided log to be used by that component. Each branch of the log may
	// have potentially different log levels.
	log = log.With().String("structure", "Server").Logger()

	return &Server{log: log}
}

func (a *Server) run(args []string) {
	a.log.Verbose().Msg("Enter Server.run()")

	// Create a local logger instance for this method.
	log := a.log.With().String("method", "run").Logger()
	log.Verbose().Msg("starting loop")

	for _, arg := range args {
		if err := a.handleRequest(log, arg); err != nil {
			log.Warning().Err(err).Msg("cannot handle request")
		}
	}
}

func (a *Server) handleRequest(log *Logger, raw string) error {
	log.Debug().Msg("Enter Server.handleRequest()")
	defer func() { log.Debug().Msg("Leave Server.handleRequest()") }()

	request, err := NewRequest(log, raw)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}
	if err = request.Handle(); err != nil {
		return fmt.Errorf("cannot process request: %w", err)
	}
	return nil
}

// Request is a demonstration structure that has its own logger, which it uses
// to log all events relating to handling this request.
type Request struct {
	log   *Logger // Log is the logger for this particular request.
	Query string  // Query is the request payload.
}

func NewRequest(log *Logger, query string) (*Request, error) {
	fields := strings.Split(query, "=")
	if len(fields) != 2 {
		return nil, fmt.Errorf("cannot parse query: %q", query)
	}

	log = log.With().
		String("request", query).
		String("left", fields[0]).
		String("right", fields[1]).
		Logger()

	log.Debug().Msg("new request")
	return &Request{log: log, Query: query}, nil
}

func (r *Request) Handle() error {
	// Anywhere in the call flow for the request, if it wants to log
	// something, it should log to the Request's logger.
	log := r.log

	log.Debug().Msg("handling request")
	return nil
}
