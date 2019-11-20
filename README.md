# gologs

## Simplified log levels

When writing software, much thought goes into choosing which level to
emit logs at for various events. Is this a debug level event? Or
verbose? What about the difference between warning and error?

This library attempts to reduce the complexity of choosing an event
level for logged events, while also reducing the complexity of
choosing a log level when running a program. There are typically three
different reasons to view logs:

1. I am a user and would like to know basic information about this
   program's operations.

2. I am an administrator running the program and want detailed runtime
   information about this program's execution.

3. I am a developer trying to figure out how this program is working
   or not working properly.

Rather than selecting from the common five log levels, this program
provides only three log levels, and adds in the concept of Tracer
logging.

1. User
2. Admin
3. Dev

Different logging configurations can be effected by creating a
composable logging tree. For instance at the base is the Logger that
writes to some io.Writer. A custom log line template can be provided,
or a default is readily available.

```Go
    var base gologs.Logger
    var err error
    log, err = gologs.New(os.Stderr, gologs.DefaultLogFormat)
    if err != nil {
        panic(err)
    }
    log.User("base logger created")
```

Everything written to the base logger is formatted according to the
provided template string, given a trailing newline, and written to the
underlying io.Writer. That io.Writer might be os.Stderr, or it might
be a log rolling library, which in turn, is writting to a set of
managed log files.

The base level logger might even be wrapped by a Filter that controls
which events get logged and which get ignored:

```Go
    log = gologs.NewFilter(log).SetAdmin()
    log.User("a filter logger controls which events are logged")
```

In the above example all logged events pass through `filter`, a Filter
Logger, and it determines which events will pass through to the base
logger, get formatted, and written to the underlying io.Writer. Note
that Filter Logger instances can have their log level safely changed
asynchronously to events being logged to them.

```Go
    log.SetLevel(foo)
```

As mentioned before, these Loggers may be composed. Perhaps on top of
the Filter Logger, two sub modules of a program are running, and the
developer would like to prefix logged events from each submodule with
the respective submodule name. Provided all the `Foo` methods log to
`Foo`'s `log` field, the logged events will be given the specified
prefix.

```Go
    type Foo struct {
        log gologs.Logger
        // ...
    }

    type Bar struct {
        log gologs.Logger
        // ...
    }

    func example1() {
        // log defined as before...
        foo := &Foo{
            log: gologs.NewPrefix(log, "[FOO] "), // NOTE the trailing space
        }
        go foo.run()

        bar := &Bar{
            log: gologs.NewPrefix(log, "[BAR] "), // NOTE the trailing space
        }
        go bar.run()
    }
```

In the above example both `Foo` and `Bar` will log events through the
underlying logger. Remember `log` here is set to a Filter Logger which
controls which events are emitted and which are ignored. But each of
the modules can also wrap their Logger with a Filter if the developer
wants to be able to set `Foo` to one log level, and `Bar` to another.

```Go
    func example2() {
        // log defined as before...
        foo := &Foo{
            log: gologs.NewFilter(gologs.NewPrefix(log, "[FOO] ")),
        }
        go foo.run()

        bar := &Bar{
            log: gologs.NewFilter(gologs.NewPrefix(log, "[BAR] ")),
        }
        go bar.run()
    }
```

In the above example, both `Foo` and `Bar` can independently control
its own log level. However, note that even if the Filter Logger from
`Foo` or `Bar` allow a log event to pass through it, it might still
get blocked by `log` which is the Logger used to create `Foo`'s and
`Bar`'s Logger instances. When multiple Filter Loggers are in series,
each of the Filters must be configured to allow desired events to pass
through them.

For example, say the developer is working on some bug in `Bar`. They
would like to see all developer and above events logged by `Bar`, but
only log user level events in `Foo`. This is possible by setting each
Filter Logger accordingly. Don't forget that `log` must be set to
allow events to pass through it. Even if `Bar` is set to `Dev`, if
`log`, which is also a filter in the log tree, is configured to log
only administrator and user events, then the developer events from
`Bar` will pass through the `Bar` filter, pass through `Bar`'s
Prefixer, and get dropped at `log`. One solution is to set the global
application `log` filter to the lowest setting by calling
`log.SetDev()`, and then controlling the log filter level of each
module for `Foo` and `Bar`:

```Go
    log.SetDev()
    foo.log.SetUser()
    bar.log.SetDev()
```

Another suggestion is to simply not create multiple filters in series
for an application.

```Go
    var log gologs.Logger

    func example3() {
        var err error
        log, err = gologs.New(os.Stderr, gologs.DefaultLogFormat)
        if err != nil {
            panic(err)
        }

        foo := &Foo{
            log: gologs.NewFilter(gologs.NewPrefix(log, "[FOO] ")),
        }
        go foo.run()

        bar := &Bar{
            log: gologs.NewFilter(gologs.NewPrefix(log, "[BAR] ")),
        }
        go bar.run()
    }
```

This makes it more easy to only have to control the log filter levels
of each module. But does eliminate the flexibility that one might have
of creating a master log level right above the base of the log tree.

## Supports Tracer logging

Orthogonal to log levels are a concept of Tracer events. This allows
the simplified log levels described above to be used for all logs, but
Tracing can be turned on for a Logger allowing all events created by
or passing through that Tracer logger to have the event's tracer bit
set.

For instance, let's say an administrator or developer wants to send a
request through their running system, logging all events related to
that request, regardless of the log level. Tracer Loggers allow this
flexibility.

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
        log   gologs.Logger
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
        r.log.Dev("beginning processing of request")
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
logged. In the past, I have done this by suffixing `&debug=true` to
the URI of a request. The undocumented API was simple enough, and when
present, would cause all events for that request to get logged. If a
user reported a problem with a request, I could repeat their request
with `&debug=true` appended to it while watching the logs to follow
the request run its way through the service. It was an effective
solution I have used to debug many problems.
