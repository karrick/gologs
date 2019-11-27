package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/karrick/gologs"
)

// Rather than use the log standard library, this example creates a global log
// variable, and once initialized, uses it to log events.

var log *gologs.Logger

func main() {
	optDebug := flag.Bool("debug", false, "Print debug output to stderr")
	optVerbose := flag.Bool("verbose", false, "Print verbose output to stderr")
	flag.Parse()

	// Initialize the global log variable, which will be used very much like the
	// log standard library would be used.
	var err error
	log, err = gologs.New(os.Stderr, gologs.DefaultCommandFormat)
	if err != nil {
		panic(err)
	}

	// Configure log level according to command line flags.
	if *optDebug {
		log.SetDev()
	} else if *optVerbose {
		log.SetAdmin()
	} else {
		log.SetUser()
	}

	for _, arg := range flag.Args() {
		log.Admin("handling arg: %q", arg)
		if err := printSize(arg); err != nil {
			log.User("%s", err)
		}
	}
}

func printSize(pathname string) error {
	stat, err := os.Stat(pathname)
	if err != nil {
		return err
	}
	log.Dev("file stat: %v", stat)

	if (stat.Mode() & os.ModeType) == 0 {
		fmt.Printf("%s is %d bytes", pathname, stat.Size())
	}

	return nil
}
