package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/karrick/gologs"
)

const ProgramVersion = "3.14"

func main() {
	optDebug := flag.Bool("debug", false, "Print debug output to stderr")
	optVerbose := flag.Bool("verbose", false, "Print verbose output to stderr")
	optQuiet := flag.Bool("quiet", false, "Print warning and error output to stderr")
	flag.Parse()

	// Create a local log variable, which will be used to create log branches
	// for other program modules.
	log := gologs.New(os.Stderr).With().String("version", ProgramVersion).Logger()

	// Configure log level according to command line flags.
	if *optDebug {
		log.SetDebug()
	} else if *optVerbose {
		log.SetVerbose()
	} else if *optQuiet {
		log.SetError()
	} else {
		log.SetInfo()
	}

	log.Verbose().
		Bool("debug", *optDebug).
		Bool("verbose", *optVerbose).
		Msg("starting service")

	log.Debug().Msg("something important to developers...")

	s := NewServer(log)

	if err := s.run(os.Stdin); err != nil {
		log.Warning().Err(err).Msg("cannot run")
	}
}

type Server struct {
	log *gologs.Logger // log is used for all log output by Server
	// plus any other fields...
}

func NewServer(log *gologs.Logger) *Server {
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

func (a *Server) run(r io.Reader) error {
	a.log.Verbose().Msg("Enter Server.run()")

	// Create a local logger instance for this method.
	log := a.log.With().String("method", "run").Logger()
	log.Verbose().Msg("starting loop")

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := a.handleRequest(log, scanner.Text()); err != nil {
			log.Warning().Err(err).Msg("cannot handle request")
		}
	}
	return scanner.Err()
}

func (a *Server) handleRequest(log *gologs.Logger, raw string) error {
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
	log   *gologs.Logger // Log is the logger for this particular request.
	Query string         // Query is the request payload.
}

func NewRequest(log *gologs.Logger, query string) (*Request, error) {
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
