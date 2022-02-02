package main

import (
	"bufio"
	"flag"
	"io"
	"os"

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
	log := gologs.New(os.Stderr)

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
		String("version", ProgramVersion).
		Bool("debug", *optDebug).
		Bool("verbose", *optVerbose).
		Msg("starting service")

	log.Debug().Msg("something important to developers...")

	a := &Alpha{Log: log.NewBranchWithString("module", "ALPHA").SetVerbose()}
	if err := a.run(os.Stdin); err != nil {
		log.Warning().Msg(err.Error())
	}
}

type Alpha struct {
	Log *gologs.Logger
	// other fields...
}

func (a *Alpha) run(r io.Reader) error {
	a.Log.Verbose().Msg("started module")

	scan := bufio.NewScanner(r)

	for scan.Scan() {
		// Create a request instance with its own logger.
		request := &Request{
			Log:   a.Log, // Usually a request can be logged at same level as module.
			Query: scan.Text(),
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
	r.Log.Debug().String("query", r.Query).Msg("handling request")
}
