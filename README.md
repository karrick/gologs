# gologs

## Why

Why yet another logging library?

1. Create a log, split it into branches, give each branch different
   log prefixes, and set each branch to independent log level.

## Goals

1. This should work within the Go ecosystem. Specifically, it should
   emit logs to any io.Writer.

1. This should be flexible enough to provide for use cases not
   originally envisioned, yet be easy enough to use to facilitate
   adoption. I should want to reach for this library for all my
   logging needs, for both command line and long running daemons.

1. This should be lightweight. This should not spin up any go
   routines. This should only allocate when creating a new logger, a
   new log branch, or when the user specifically requires it by
   invoking the `Format` method. Events that do not get logged should
   not be formatted. This should not ask the OS for the system time if
   log format specification does not require it.

1. This should be correct. It should never invoke Write more than once
   per logged event.

[![GoDoc](https://godoc.org/github.com/karrick/gologs?status.svg)](https://godoc.org/github.com/karrick/gologs)

## Usage Example

```Go
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
```

## Description

### Creating a Logger Instance

Everything written by this logger is formatted as a JSON event, given
a trailing newline, and written to the underlying io.Writer. That
io.Writer might be os.Stderr, or it might be a log rolling library,
which in turn, is writting to a set of managed log files. The library
provides a few time formatting functions, but the time is only
included if the Logger is updated to either one of the provided time
formatting functions or a user specified one.

```Go
    log1 := gologs.New(os.Stderr)
    log1.Info().String("version", "3.14").Msg("started program")
    // Output:
	// {"level":"info","version":"3.14","message":"starting program"}

    log2 := gologs.New(os.Stderr).SetTimeFormatter(gologs.TimeUnix)
    log2.Info().String("version", ProgramVersion).Msg("started program")
    // Output:
	// {"time":1643776764,"level":"info","version":"3.14","message":"starting program"}

    log3 := gologs.New(os.Stderr).SetTimeFormatter(gologs.TimeUnixNano)
    log3.Info().String("version", ProgramVersion).Msg("started program")
    // Output:
	// {"time":1643776794592630092,"level":"info","version":"3.14","message":"starting program"}
```

### Log Levels

Like most logging libraries, the basic logger provides methods to
change its log level, controling which events get logged and which get
ignored.

```Go
    log.SetVerbose()
    log.Info().Msg("this event gets logged")
    log.Verbose().Msg("and so does this event")
    log.Debug().Msg("but this event gets ignored")

    log.SetLevel(gologs.Debug)
    log.Debug().Msg("this event does get logged")
```

When a logger is in Error mode, only Error events are logged. When a
logger is in Warning mode, only Error and Warning events are
logged. When a logger is in Info mode, only Error, Warning, and Info
events are logged. When a logger is in Verbose mode, only Error,
Warning, Info, and Verbose events are logged. When a logger is in
Debug mode, all events are logged.

Note the logger mode for a newly created Logger is Warning, which I
feel is in keeping with the UNIX philosophy to _Avoid unnecessary
output_. Simple command line programs will not need to set the log
level to prevent spewing too many events. While service application
developers will need to spend a few minutes to build in the ability to
configure the log level based on their service needs.

Perhaps more idiomatic of a command line program log configuration:

```Go
	if *optDebug {
		log.SetDebug()
	} else if *optVerbose {
		log.SetVerbose()
	} else if *optQuiet {
		log.SetError()
	} else {
		log.SetInfo()
	}
```

### A Tree of Logs with Multiple Branches

In managing several real world services, I discovered the need for
finer granularity in managing which events are logged in different
parts of the same running program. Sometimes all events in one
particular module of the service should be logged with great detail,
while a different part of the program is deemed functional and the
loggging of developer events would saturate the logs.

This library allows this workflow by allowing a developer to create a
tree of logs with multiple branches, and each branch can have an
independently controlled log level. These log branches are
lightweight, require no go routines to facilitate, and can even be
ephemeral, and demonstrated later in the Tracer Logging section.

#### Base of the Tree

To be able to independently control log levels of different parts of
the same program at runtime, this library provides for the creation of
what I like to call a tree of logs. At the base of the tree, events
are written to an underlying io.Writer. This allows a developer to
create a log and have it write to standard error, standard output, a
file handle, a log rolling library which writes to a file, or any
other structure that implements the io.Writer interface.

#### Creating New Branches for the Log Tree

Different logging configurations can be effected by creating a logging
tree, and while the tree may be arbitrarily complex, a simple tree is
likely more developer friendly than a complex one. For instance, I
have adopted the pattern of creating a very small tree, with a base
logger for the entire application, and a logger branch for each major
module of the program. Each of those branches can have a different log
level, each of which can be controlled at runtime using various means,
always by invoking one of its log level control methods. Additionally
each branch can have a particular string prefix provided that will
prefix the logged events.

This allows each branch to have an independently controlled log level,
and the program can set one logger to run at `Debug` mode, while the
other branches run at different levels. These log levels are also safe
to asynchronously modify while other threads are actively logging
events to them.

```Go
    // Foo is a module of the program with its own logger.
    type Foo struct {
        log *gologs.Logger
        // ...
    }

    // Bar is a module of the program with its own logger.
    type Bar struct {
        log *gologs.Logger
        // ...
    }

    func example1() {
        // log defined as in previous examples...
        foo := &Foo{
            // NOTE: the branch prefix has a trailing space in order to
            // format nicely. You may prefer "FOO: " as your prefix, or
            // even just "FOO:".
            log: log.NewBranchWithString("module","FOO"),
        }
        go foo.run()

        bar := &Bar{
            log: log.NewBranchWithString("module","BAR"),
        }
        go bar.run()
    }
```

In the above example both `Foo` and `Bar` are provided their own
individual logger to use, and both `Foo` and `Bar` can independently
control its own log level. It is important that they use that logger
to log all of their events during their lifetime, in order to be
effective.

It is possible to create a branch of a logger that does not have a
prefix. In the below example, `log2` merely branches the logs so that
the developer can independently control the log level of that
particular branch of logs.

```Go
    log2 := log.NewBranch()
```
