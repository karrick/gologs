package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/karrick/golf"
	"github.com/karrick/gologs"
)

func main() {
	var ProgramName string
	var err error
	if ProgramName, err = os.Executable(); err != nil {
		ProgramName = os.Args[0]
	}
	ProgramName = filepath.Base(ProgramName)

	optQuiet := golf.BoolP('q', "quiet", false, "Do not print intermediate errors to stderr")
	optVerbose := golf.BoolP('v', "verbose", false, "Print verbose output to stderr")
	optDebug := golf.BoolP('d', "debug", false, "Print debug output to stderr")
	golf.Parse()

	// Create a filtered logger by compiling the log format string.
	log := gologs.NewFilter(gologs.New(os.Stderr, fmt.Sprintf("{localtime=2006-01-02T15:04:05} [%s] {message}", ProgramName)))

	// Initialize the logger mode based on the provided command line flags.
	if *optDebug {
		log.SetDev()
	} else if *optVerbose {
		log.SetAdmin()
	} else if *optQuiet {
		log.SetUser()
	} else {
		log.SetUser()
	}

	//
	log.Admin("Starting up service: %v %v %v", 3.14, "hello", struct{}{}) // Admin events not logged when filter set to User level

	rand.Seed(time.Now().Unix())

	// Handle some example requests from the command line arguments.
	for _, arg := range golf.Args() {
		// Create a request instance with its own logger.
		request := &Request{
			Log:   log,
			Query: arg,
		}
		if rand.Intn(10) < 5 {
			request.Log = gologs.NewTracer(log, fmt.Sprintf("arg=%s: ", arg))
		}
		request.Handle()
	}
}

// Request is a demonstration structure that has its own logger, which it uses
// to log all events relating to handling this request.
type Request struct {
	Log   gologs.Logger // Log is the logger for this particular request.
	Query string        // Query is the request payload.
}

func (r *Request) Handle() {
	// Anywhere in the call flow for the request, if it wants to log something,
	// it should log to the Request's logger.
	r.Log.Dev("this event rips through filter to get logged")
}
