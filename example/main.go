package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/karrick/golf"
	"github.com/karrick/gologs"
)

var log *gologs.Filter

func init() {
	// Create a filtered logger by compiling the log format string.
	var err error
	log, err = gologs.New(os.Stderr, "{program} {message}")
	if err != nil {
		panic(err)
	}
}

func main() {
	optDebug := golf.Bool("debug", false, "Print debug output to stderr")
	optVerbose := golf.Bool("verbose", false, "Print verbose output to stderr")
	golf.Parse()

	// Initialize the logger mode based on the provided command line flags.
	if *optDebug {
		log.SetDev()
	} else if *optVerbose {
		log.SetAdmin()
	} else {
		log.SetUser()
	}

	log.Admin("Starting service: %v %v %v", 3.14, "hello", struct{}{}) // Admin events not logged when filter set to User level

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
