# gologs

Overview: [![GoDoc](https://godoc.org/github.com/karrick/gologs?status.svg)](https://godoc.org/github.com/karrick/gologs)

## High Level Features

### Simplified Log Levels

When writing software, much thought goes into choosing which level to
emit logs at for various events. Is this a debug level event? Or
verbose? What about the difference between warning and error?

This library attempts to reduce the complexity of choosing an event
level for logged events, while also reducing the complexity of
choosing a log level for a running program. There are typically three
different reasons to view logs:

1. I am a user and would like to know basic information about this
   program's operations. Why did it die, for instance.

2. I am an administrator running the program and want detailed runtime
   information about this program's execution. When did the program
   start? When did the service experience connectivity issues, even
   though it is designed to reconnect on failures?

3. I am a developer trying to figure out how this program is working
   or not working properly. How did the program get down that path?

Rather than selecting from the five common log levels, this program
provides only three log levels, corresponding exactly to the above
list, then adds in the concept of Tracer logging, described in the
next section.

When a logger is in User mode, only User events are logged. When a
logger is in Admin mode, only Admin and User events are logged. When a
logger is in Dev move, all Dev, Admin, and User events are logged.

Note the logger mode for a newly created Logger is User, which I feel
is in keeping with the UNIX philosophy to _Avoid unnecessary
output_. Simple command line programs will not need to set the log
level to prevent spewing too many log events. While service
application developers are more likely to spend a few minutes to build
in the ability to configure the log level based on their service
needs.

### Tracer Logging

Orthogonal to log levels are a concept of Tracer events. This allows
the simplified log levels described above to be used for all logs, but
tracing can be turned on for a set of log events pertaining to a
single request. For instance in the past, I have done this by
suffixing `&debug=true` to the URI of a request. The undocumented API
would cause all events for that request to get logged, regardless of
the log level. If a user reported a problem with a valid looking
request, I could repeat their request with `&debug=true` appended to
it while watching the logs to follow the request run its way through
the service. It was an effective solution I have used to debug many
problems.

## Description

### A Tree of Logs?

The only thing we've done differently so far is reduce the complexity
of 5 log levels down to 3, which is nice, but nothing to justify use
of a different logging library. However, in managing several real
world services, I discovered the need for finer granularity in
managing which events are logged. Sometimes all events in one
particular module of the service should be logged. Other times every
event associated with a particular request should be logged. This
library is an attempt to satisfy those real world operational desires.

#### Base of the Tree

To do this, this library provides for the creation of what I like to
call a tree of loggers. Maybe a bad term, but stick with me for a
moment. At the base of the tree, events are written to an underlying
io.Writer. This allows a developer to create a logger and have it
write to standard output, standard error, a file handle, a log rolling
library which writes to a file, or any other io.Writer, thanks to Go
interfaces.

```Go
    log, err := gologs.New(os.Stderr, gologs.DefaultServiceFormat)
    if err != nil {
        panic(err)
    }
    log.User("program started") // "2006/01/02 15:04:05 [USER] program started"
```

Everything written to this logger is formatted according to the
provided template string, given a trailing newline, and written to the
underlying io.Writer. That io.Writer might be os.Stderr, or it might
be a log rolling library, which in turn, is writting to a set of
managed log files. The library provides a few default log template
strings, but in every case, when the logger is created, the template
string is compiled to tree of function pointers that is evaluated over
each log event to format the event according to the template. This is
in stark contrast to many other logging libraries that evaluate the
template string for each event to be logged.

#### Controlling Log Levels

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

### Tracer logging

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
