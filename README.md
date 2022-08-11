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
   logging needs, for both command line and long running services.

1. This should be lightweight. This should not spin up any go
   routines. This should only allocate when creating a new logger or a
   new log branch, or when the user specifically requires it by
   invoking the `Format` method. Events that do not get logged should
   not be formatted. This should not ask the OS for the system time if
   log format specification does not require it.

1. This should be correct. It should never invoke Write more than once
   per logged event.

[![GoDoc](https://godoc.org/github.com/karrick/gologs?status.svg)](https://godoc.org/github.com/karrick/gologs)

## Compliments

A while ago I significantly altered the API of this library based on
the amazing [zerolog](https://github.com/rs/zerolog) library. I hope
the authors of that library have heard the expression that imitation
is the most sincere form of flattery.

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
```

## Description

### Creating a Logger Instance

Everything written by this logger is formatted as a JSON event, given
a trailing newline, and written to the underlying io.Writer. That
io.Writer might be os.Stderr, or it might be a log rolling library,
which in turn, is writting to a set of managed log files. The library
provides a few time formatting functions, but the time is only
included when the Logger is updated to either one of the provided time
formatting functions or a user specified time formatter.

```Go
    log1 := gologs.New(os.Stderr)
    log1.Info().Msg("started program")
    // Output:
    // {"level":"info","message":"starting program"}

    log2 := gologs.New(os.Stderr).SetTimeFormatter(gologs.TimeUnix)
    log2.Info().Msg("started program")
    // Output:
    // {"time":1643776764,"level":"info","message":"starting program"}

    log3 := gologs.New(os.Stderr).SetTimeFormatter(gologs.TimeUnixNano)
    log3.Info().Msg("started program")
    // Output:
    // {"time":1643776794592630092,"level":"info","message":"starting program"}

    log4 := gologs.New(os.Stderr).SetTimeFormatter(gologs.TimeFormat(time.RFC3339))
    log4.Info().Msg("started program")
    // Output:
    // {"time":"2022-08-06T15:14:04-04:00","level":"info","message":"starting program"}

    log5 := gologs.New(os.Stderr).SetTimeFormatter(gologs.TimeFormat(time.Kitchen))
    log5.Info().Msg("started program")
    // Output:
    // {"time":"3:14PM","level":"info","message":"starting program"}
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
ephemeral, and demonstrated later in the Tracer Logging
section. Creating logger branches do allocate by copying the branch
byte slice from the parent to the child branch.

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
            log: log.With().String("module","FOO").Logger(),
        }
        go foo.run()

        bar := &Bar{
            log: log.With().String("module","BAR").Logger(),
        }
        go bar.run()
    }
```

In the above example both `Foo` and `Bar` are provided their own
individual logger to use, and both `Foo` and `Bar` can independently
control its own log level. It is important that they use that logger
to log all of their events during their lifetime, in order to be
effective.

### Tracer Logging

I am sure I'm not the only person who wanted to figure out why a
particular request or action was not working properly on a running
service, decided to activate DEBUG log levels to watch the single
request percolate through the service, to be reminded that the service
is actually serving tens of thousands of requests per second, and now
the additional slowdown that accompanies logging each and every single
log event in the entire program not only slows it down, but makes it
impossible to see the action or request in the maelstrom of log
messages scrolling by the terminal.

For instance, let's say an administrator or developer wants to send a
request through their running system, logging all events related to
that request, regardless of the log level, but not necessarily see
events for other requests.

For this example, remember that each module has a Logger it uses
whenever logging any event. Let's say the `Foo` module receives
requests to process. The `Foo` can create highly ephemeral Tracer
Loggers to be assigned to the request instance itself, and provided
that the request methods log using the provided logger, then those
events will bypass any filters in place between where the log event
was created to the base of the logging tree, and get written to the
underlying io.Writer.

```Go
    type Request struct {
        log   *gologs.Logger
        query string
        // ...
    }

    func (f *Foo) NewRequest(query string) (*Request, error) {
        r := &Request{
            log:   f.log.With().String("request", query).Logger(),
            query: query,
        }
        if strings.HasSuffix(key, "*") {
            r.log.SetTracing(true)
        }
        // ...
    }

    func (r *Request) Process() error {
        r.log.Debug().Msg("beginning processing of request")
        // ...
    }
```

It is important to remember that events sent to a Logger configured
for tracing will bypass all log level filters. So `log`, `Foo`, and
`Bar` all might be set for Warning level, but you want to follow a
particular request through the system, without changing the log
levels, also causing the system to log every other request. Tracer
logic is not meant to be added and removed while debugging a program,
but rather left in place, run in production, but not used, unless some
special developer or administrator requested token marks a particular
event as one for which all events should be logged.

Here's an example of what Tracer Loggers are trying to eliminate,
assuming a hypothetical `Logging.Trace` method existed:

```Go
    // Counter example: desired behavior sans tracer logic. Each log line
    // becomes a conditional, leading to code bloat.
    func (r *Request) Handler() {
        // It is inconvenient to branch log events each place you want to
        // emit a log event.
        if r.isSpecial {
            r.Log.Trace().Msg("handling request")
        } else {
            r.Log.Debug().Msg("handling request: %v", r)
        }

        // Do some work, then need to log more:
        if r.isSpecial {
            r.Log.Trace().Int("request-cycles", r.Cycles).Msg("")
        } else {
            r.Log.Debug().Int("request-cycles", r.Cycles).Msg("")
        }
    }
```

I propose something better, where the developer does not need to
include conditional statements to branch based on whether the log
should receive Tracer status or Verbose status for each log
event. Yet, when Tracer status, still get written to the log when
something requires it.

```Go
    func NewRequest(log *gologs.Logger, key string) (*Request, error) {
        log = log.With().
            String("key", key).
            Tracing(strings.HasSuffix(key, "*")).
            Logger()
        r := &R{
            log: log,
            Key: key,
        }
        return r, nil
    }

    func (r *Request) Handler() {
        r.Log.Debug().Msg("handling request")

        // Do some work, then need to log more:

        r.Log.Debug().Int("request-cycles", r.Cycles).Msg("")
    }
```
