package main

import (
	"fmt"
	"os"
	"path/filepath"

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

	log := gologs.NewFilter(gologs.New(os.Stderr, fmt.Sprintf("{localtime=2006-01-02T15:04:05} [%s] {message}", ProgramName)))

	if *optDebug {
		log.SetDev()
	} else if *optVerbose {
		log.SetAdmin()
	} else if *optQuiet {
		log.SetUser()
	} else {
		log.SetUser()
	}

	log.Admin("%v %v %v", 3.14, "hello", struct{}{}) // Admin events not logged when filter set to User level

	handleRequest(log, golf.Args())
}

func handleRequest(log gologs.Logger, args []string) {
	for _, arg := range args {
		tracer := gologs.NewTracer(log, fmt.Sprintf("arg=%s: ", arg))
		tracer.Dev("this event rips through filter to get logged")
	}
}
