# gologs

Online documentation:
[![GoDoc](https://godoc.org/github.com/karrick/gologs?status.svg)](https://godoc.org/github.com/karrick/gologs)

## Example

```Go
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
```

## Description

### Creating a Logger Instance

Everything written to this logger is formatted according to the
provided template string, given a trailing newline, and written to the
underlying io.Writer. That io.Writer might be os.Stderr, or it might
be a log rolling library, which in turn, is writting to a set of
managed log files. The library provides a few default log template
strings, but in every case, when the logger is created, the template
string is compiled to tree of function pointers that is evaluated over
each log event to format the event according to the template. This is
in contrast to many other logging libraries that evaluate the template
string for each event to be logged.

```Go
    log, err := gologs.New(os.Stderr, gologs.DefaultServiceFormat)
    if err != nil {
        panic(err)
    }
    log.User("started program: v%s", ProgramVersion) // "2006/01/02 15:04:05 started program: v3.14"
```

### Think in Terms of Event Audience Rather Than Event Type

When writing software, much thought goes into choosing which level to
emit logs at for various events. Is this a debug level event? Or
verbose? What about the difference between warning and error?

Rather than selecting from the five common log levels, this library
focuses on the audience of each particular log event rather than what
type of event is being logged. There are three audiences for any log
event: the user running the program; the administrator running this
program; and the developers working on the program.

A regular user wants a program to work. One of the tennants of The
UNIX Philosophy is to _Avoid Unnecessary output_. When a program can
do a task quickly, just do it, and emit the results. The UNIX `rm foo`
command does not need to say, "foo deleted", but it should say why it
was not deleted if it could not delete `foo`. This is what User mode
logging is for. Tell me only the most important of things. But those
important things are not limited to errors. There are many events that
a user might deem important to know about even if not an error.

An administrator wants a program to work, and wants to know everything
a user might want to know about, but with a bit more information about
what the program is doing. An administrator might want to know when
the program started, if it's a service, what signals it received, when
it decides to re-read its configuration. When a service upon which it
depends failed and it had to resend a query. There are numerous
examples of recoverable errors, and non error events that an
administrator would want to know about.

A developer also wants a program to work, and usually want to know why
it's not working, so they can fix it. They want to see what a request
looks like when the request should work but is failing for some
reason, for instance.

The log levels are designed to help developers to think about the
audience of the logged event, and phrase the wording accordingly. In
contrast with most logging libraries, this does not map concepts such
as WARNING or ERROR messages into levels. There are errors only a
developer would want to see, errors that an administrator would want
to see, and errors that all users should see. It is perfectly
reasonable for a particular event to be an error, but intended for the
developer's attention.

Like most logging libraries, the basic logger provides methods to
change its log level, controling which events get logged and which get
ignored. Rembember this library exposes three log levels. Rather than
thinking in terms of what you are logging, with this library the
developer is thinking about who is looking at the logs. The person
running the program specifies what they would like to see. When
deciding what level to emit an event as, ask yourself if an
administrator would ever care about that event. If not, make it a Dev
event. Or ask if a user would ever care about that event. If not, make
it an Admin event.

```Go
    log.SetAdmin()
    log.User("this event gets logged")
    log.Admin("and so does this event")
    log.Dev("but this event gets ignored")

    log.SetLevel(gologs.Dev)
    log.Dev("this event does get logged")
```

Perhaps more idiomatic of a command line program log configuration:

```Go
	if *optDebug {
		log.SetDev()
	} else if *optVerbose {
		log.SetAdmin()
	} else {
		log.SetUser()
	}
```


```Go
    log.Dev("ERROR: formatting loop post condition violated: %d > %d", foo, bar)
```

Likewise is is perfectly reasonable for a particular event to be
addressed to a program administrator:

```Go
    log.Admin("WARNING: config threshold too low; using default %d: %d < %d", default, value, threshold)
```

When a logger is in User mode, only User events are logged. When a
logger is in Admin mode, only Admin and User events are logged. When a
logger is in Dev move, all Dev, Admin, and User events are logged.

Note the logger mode for a newly created Logger is User, which I feel
is in keeping with the UNIX philosophy to _Avoid unnecessary
output_. Simple command line programs will not need to set the log
level to prevent spewing too many events. While service application
developers will need to spend a few minutes to build in the ability to
configure the log level based on their service needs.

### A Tree of Logs with Multiple Branches

In managing several real world services, I discovered the need for
finer granularity in managing which events are logged in different
parts of the same running program. Sometimes all events in one
particular module of the service should be logged with great detail,
while a different part of the program is deemed functional and the
loggging of developer events would saturate the logs.

This library allows this workflow by allowing a developer to create a
tree of logs with multiple branches, and each branch can have an
independently controlled log level. These log branches are quite
lightweight, require no go routines to facilitate, and can even be
ephemeral, and demonstrated later in the Tracer Logging section.

#### Base of the Tree

To be able to independently control log levels of different parts of
the same program at runtime, this library provides for the creation of
what I like to call a tree of logs. At the base of the tree, events
are written to an underlying io.Writer. This allows a developer to
create a log and have it write to standard error, standard output, a
file handle, a log rolling library which writes to a file, or any
other io.Writer, thanks to Go interfaces.

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
and the program can set one logger to run at `Dev` mode, while the
other branches run at `Admin`, or `User` mode. These log levels are
also safe to asynchronously modify while other threads are actively
logging events to them.

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
            log: gologs.NewBranchWithPrefix(log, "[FOO] "),
        }
        go foo.run()

        bar := &Bar{
            log: gologs.NewBranchWithPrefix(log, "[BAR] "),
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
    log2 := gologs.NewBranch(log)
```

### Tracer Logging

I'm sure I'm not the only person who wanted to figure out why a
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
            log:   f.log,
            query: query,
        }
        if strings.HasSuffix("*") {
            r.log = gologs.NewTracer(r.log, fmt.Sprintf"[REQUEST %q] ", query)
        }
        // ...
    }

    func (r *Request) Process() error {
        r.log.Dev("beginning processing of request: %v", r)
        // ...
    }
```

It is important to remember that tracer events bypass all log level
filters. So `log`, `Foo`, and `Bar` all might be set for administrator
level, but you want to follow a particular request through the system,
without changing the log levels, also causing the system to log every
other request. Tracer logic is not meant to be added and removed while
debugging a program, but rather left in place, run in production, but
not used, unless some special developer or administrator requested
token marks a particular event as one for which all events should be
logged.

####### HERE

Log levels allow a program to control what gets logged and what gets
ignored. Log braches allow the developer to have simultaneously
different log levels in different parts of a program, each
independently controlled. Sometimes it is also convenient to emit one
or more events that will be written to the logs regardless of the log
level. Normally this is done by changing the log level a particular
event is emitted as. I've come to the conclusion that there is a
better way, and I'd like to describe it by means of an example.

```Go
    func (r *Request) Handler() {
        // It is inconvenient to branch log events each place you want to
        // emit a log event.
        if r.isSpecial {
            r.Log.Trace("handling request: %v", r)
        } else {
            r.Log.Dev("handling request: %v", r)
        }

        // Do some work, then need to log more:
        if r.isSpecial {
            r.Log.Trace("request.Cycles: %d", r.Cycles)
        } else {
            r.Log.Dev("request.Cycles: %d", r.Cycles)
        }
    }
```

I propose something better, where the developer does not need to
include conditional statements to branch based on whether the log
should receive Tracer status or Admin status for each log event. Yet,
when Tracer status, still get written to the log when something
requires it.

```Go
    func NewRequest(log *gologs.Logger, key string) (*Request, error) {
        r := &R{Log: log, Key: key}
        if r.isSpecial() {
            r.Log = gologs.NewTracer(r.Log, fmt.Sprintf("[REQUEST %s] ", r.Key))
        }
        return r, nil
    }

    func (r *Request) Handler() {
        r.Log.Dev("handling request: %v", r)

        // Do some work, then need to log more:
        r.Log.Dev("request.Cycles: %d", r.Cycles)
    }
```

Note the 

This allows the simplified log levels described above to be used for
all logs, but tracing can be turned on for a set of log events
pertaining to a single request. For instance in the past, I have done
this by suffixing `&debug=true` to the URI of a request. The
undocumented API would cause all events for that request to get
logged, regardless of the log level. If a user reported a problem with
a valid looking request, I could repeat their request with
`&debug=true` appended to it while watching the logs to follow the
request run its way through the service. It was an effective solution
I have used to debug many problems.

## Description

### A Tree of Logs?

### Tracer logging

