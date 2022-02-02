package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/karrick/gologs/v2"
)

func main() {
	optDebug := flag.Bool("debug", false, "Print debug output to stderr")
	optVerbose := flag.Bool("verbose", false, "Print verbose output to stderr")
	optQuiet := flag.Bool("quiet", false, "Print warning and error output to stderr")
	flag.Parse()

	// Initialize the global log variable, which will be used very much like the
	// log standard library would be used.
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

	// For sake of example, invoke printSize with a logger that includes the
	// function name in the JSON properties of the log message.
	pl := log.NewBranchWithString("function", "printSize")

	for _, arg := range flag.Args() {
		log.Verbose().String("arg", arg).Msg("")
		if err := printSize(pl, arg); err != nil {
			log.Warning().Msg(err.Error())
		}
	}
}

func printSize(log *gologs.Logger, pathname string) error {
	stat, err := os.Stat(pathname)
	if err != nil {
		return err
	}
	log.Debug().Int("size", int64(stat.Size())).Msg("")

	if (stat.Mode() & os.ModeType) == 0 {
		fmt.Printf("%s is %d bytes\n", pathname, stat.Size())
	}

	return nil
}
