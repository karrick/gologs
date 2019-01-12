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
provides only three log levels, along with a concept of Tracer
logging.

1. User

2. Admin

3. Dev

## Supports Tracer logging

Orthogonal to log levels are a concept of Tracer events. This allows
the simplified log levels described above to be used for all logs, but
Tracing can be turned on for a Logger allowing all events created by
or passing through that Tracer logger to have the event's tracer bit
set.
