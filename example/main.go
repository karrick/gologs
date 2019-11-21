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

func main() {
	optDebug := flag.Bool("debug", false, "Print debug output to stderr")
	optVerbose := flag.Bool("verbose", false, "Print verbose output to stderr")
	flag.Parse()

	// Initialize the logger mode based on the provided command line flags.
	// Create a filtered logger by compiling the log format string.
	log, err := gologs.New(os.Stderr, "{program} {message}")
	if err != nil {
		panic(err)
	}
	if *optDebug {
		log.SetDev()
	} else if *optVerbose {
		log.SetAdmin()
	} else {
		log.SetUser()
	}
	log.Admin("Starting program; debug: %v; verbose: %v", *optDebug, *optVerbose)
	log.Dev("something important to developers...")

	a := &Alpha{Log: gologs.NewBranchWithPrefix(log, "[ALPHA] ").SetAdmin()}
	if err := a.run(os.Stdin); err != nil {
		log.User("%s", err)
	}
}

type Alpha struct {
	Log *gologs.Logger
	// other fields...
}

func (a *Alpha) run(r io.Reader) error {
	a.Log.Admin("Started module")

	scan := bufio.NewScanner(r)

	for scan.Scan() {
		// Create a request instance with its own logger.
		request := &Request{
			Log:   a.Log, // Usually a request can be logged at same level as module.
			Query: scan.Text(),
		}
		if strings.HasPrefix(request.Query, "@") {
			// For demonstration purposes, let's arbitrarily cause some of the
			// events to be logged with tracers.
			request.Log = gologs.NewTracer(request.Log, fmt.Sprintf("[REQUEST %s] ", request.Query))
		}
		request.Handle()
	}

	return scan.Err()
}

// Request is a demonstration structure that has its own logger, which it uses
// to log all events relating to handling this request.
type Request struct {
	Log   *gologs.Logger // Log is the logger for this particular request.
	Query string         // Query is the request payload.
}

func (r *Request) Handle() {
	// Anywhere in the call flow for the request, if it wants to log something,
	// it should log to the Request's logger.
	r.Log.Dev("handling request: %v", r.Query)
}
