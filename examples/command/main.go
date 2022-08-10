package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/karrick/gologs"
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

	log.SetTimeFormatter(gologs.TimeRFC3339)
	// log.SetTimeFormatter(gologs.TimeFormat(time.Kitchen))

	// For sake of example, invoke printSize with a child logger that includes
	// the function name in the JSON properties of the log message.
	clog := log.With().String("function", "printSize").Logger()

	for _, arg := range flag.Args() {
		// NOTE: Sends event to parent logger.
		log.Verbose().String("arg", arg).Msg("")

		// NOTE: Sends events to child logger.
		if err := printSize(clog, arg); err != nil {
			log.Warning().Err(err).Msg("")
		}
	}
}

func printSize(log *gologs.Logger, pathname string) error {
	log.Debug().String("pathname", pathname).Msg("stat")
	stat, err := os.Stat(pathname)
	if err != nil {
		return err
	}

	size := stat.Size()
	log.Debug().String("pathname", pathname).Int64("size", size).Msg("")

	if (stat.Mode() & os.ModeType) == 0 {
		fmt.Printf("%s is %d bytes\n", pathname, size)
	}

	return nil
}
