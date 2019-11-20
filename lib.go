package gologs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// DefaultCommandFormat specifies a log format might be more appropriate for a
// infrequently used command line program, where the name of the service is a
// recommended part of the log line, but the timestamp is not.
const DefaultCommandFormat = "{program} {message}"

// DefaultServiceFormat specifies a log format might be more appropriate for a
// service daemon, where the name of the service is implied by the filename the
// logs will eventually be written to. The default timestamp format is the same
// as what the standard library logs times as, but different timestamp formats
// are readily available, and the timestamp format is also customizable.
const DefaultServiceFormat = "{timestamp} [{level}] {message}"

// Level type defines one of several possible log levels.
type Level uint32

const (
	// Dev is for developers; show me all events. Default mode for Logger. As
	// one begins development of a program, one will likely want to log all
	// Developer events, so this is the default mode for the logger. Once a
	// program is reaching a higher level of maturity, it is more likely the
	// developer will have had an opportunity to allow setting the program log
	// levels according to regular conventions, and hide most of the developer
	// related log events.
	Dev Level = iota

	// Admin is for administrators; show me detailed operational events. A long
	// running program running as a daemon is most likely going to run with log
	// level set to emit events that an administrator is concerned with.
	Admin

	// User is for users; show me major operational events. A command line
	// program with no special command line options for logs will likely want to
	// run with logging set for user level events.
	User
)

func (l Level) String() string {
	switch l {
	case User:
		return "USER"
	case Admin:
		return "ADMIN"
	case Dev:
		return "DEV"
	}
	panic(fmt.Sprintf("invalid log level: %v", uint32(l)))
}

// Event instances are created by Loggers and flow through the log tree down to
// the base, at which point, its arguments will be formatted immediately prior
// to writing the log message to the underlying log io.Writer.
type Event struct {
	args   []interface{}
	prefix []string
	when   time.Time
	format string
	level  Level
}

// base formats the event to a byte slice, ensuring it ends with a newline, and
// writes its output to its underlying io.Writer.
type base struct {
	formatters []func(*Event, *[]byte)
	w          io.Writer
	c          int // c is count of bytes to allocate for formatting log line
	m          sync.Mutex
}

func (b *base) Log(e *Event) error {
	// ??? *If* want to sacrifice a bit of speed, might consider using a
	// pre-allocated byte slice to format the output. The pre-allocated slice
	// can be protected with the lock already being used to serialize output, or
	// even better, its own lock so one thread can be formatting an event while
	// a different thread is writing the formatted event to the underlying
	// writer.
	buf := make([]byte, 0, b.c)

	// NOTE: This logic allows for a race between two threads that both get the
	// time for an event, then race for the mutex below that serializes output
	// to the underlying io.Writer. While not dangerous, the logic might allow
	// two log lines to be emitted to the writer in opposite timestamp order.
	e.when = time.Now()

	// Format the event according to the compiled formatting functions created
	// when the logger was created, according to the log template, i.e.,
	// "{timestamp} [{level}] {message}".
	for _, formatter := range b.formatters {
		formatter(e, &buf)
	}

	// Serialize access to the underlying io.Writer.
	b.m.Lock()
	_, err := b.w.Write(buf)
	b.m.Unlock()
	return err
}

// Logger interface specifies something that can act as a logger. There are
// several structures in this library that provide the Logger interface. Logger
// structures can be composed and connected, much like io.Reader instances, to
// create required desired log handling.
type Logger interface {
	Log(*Event) error
}

// New returns a new Logger that emits logged events to w after formatting the
// event according to template. This returns a Filter Logger, so that an
// application's base logger itself can have its log level controlled without
// composition by another Filter on top of it, because having multiple log
// levels is pretty much standard practice.
func New(w io.Writer, template string) (*Filter, error) {
	if strings.HasSuffix(template, "\n") {
		return nil, errors.New("cannot create logger with final newline")
	}
	formatters, err := compileFormat(template)
	if err != nil {
		return nil, err
	}
	// Create a dummy event to see how long the log line is with the provided
	// template.
	buf := make([]byte, 0, 64)
	var e Event
	for _, formatter := range formatters {
		formatter(&e, &buf)
	}
	return NewFilter(&base{w: w, formatters: formatters, c: len(buf) + 64}), nil
}

// Filter Logger will only convey events at the same level as the Filter is set
// for or higher.
type Filter struct {
	parent Logger
	level  uint32
}

// NewFilter returns a new Filter Logger that passes logged events to the
// underlying Logger depending on the Filter's configurable level and the level
// of the event.
func NewFilter(logger Logger) *Filter {
	return &Filter{parent: logger, level: uint32(User)}
}

// SetLevel allows changing the log level of the Filter Logger. Events must have
// the same log level or higher for the Filter Logger for events to be logged.
func (f *Filter) SetLevel(level Level) *Filter {
	atomic.StoreUint32(&f.level, uint32(level))
	return f
}

// SetDev changes the log level of the Filter Logger to Dev, which allows all
// events to be logged.
func (f *Filter) SetDev() *Filter {
	atomic.StoreUint32(&f.level, uint32(Dev))
	return f
}

// SetAdmin changes the log level of the Filter Logger to Admin, which allows
// all Admin and User events to be logged.
func (f *Filter) SetAdmin() *Filter {
	atomic.StoreUint32(&f.level, uint32(Admin))
	return f
}

// SetUser changes the log level of the Filter Logger to User, which allows all
// User events to be logged.
func (f *Filter) SetUser() *Filter {
	atomic.StoreUint32(&f.level, uint32(User))
	return f
}

// Dev is used to inject an event considered interesting for developers into the
// log stream. Note the Filter Logger must have been set to the Dev log level
// for this event to be logged.
func (f *Filter) Dev(format string, args ...interface{}) error {
	if Level(atomic.LoadUint32(&f.level)) > Dev {
		return nil
	}
	return f.parent.Log(&Event{format: format, args: args, level: Dev})
}

// Admin is used to inject an event considered interesting for administrators
// into the log stream. Note the Filter Logger must have been set to the Dev or
// Admin level for this event to be logged.
func (f *Filter) Admin(format string, args ...interface{}) error {
	if Level(atomic.LoadUint32(&f.level)) > Admin {
		return nil
	}
	return f.parent.Log(&Event{format: format, args: args, level: Admin})
}

// User is used to inject an event considered interesting for users into the log
// stream. The created event will be logged regardless of the log level of the
// Filter Logger, as User events are considered the highest priority events.
func (f *Filter) User(format string, args ...interface{}) error {
	return f.parent.Log(&Event{format: format, args: args, level: User})
}

// Log is used by Loggers that compose on top of this Filter Logger, and only
// allow appropriate events to pass through the filter, and drop events that
// have a level set lower than the Filter Logger has been configured for.
func (f *Filter) Log(e *Event) error {
	if Level(atomic.LoadUint32(&f.level)) > e.level {
		return nil
	}
	return f.parent.Log(e)
}

// Prefixer is a Logger that prefixes each logged event with a particular
// string.
type Prefixer struct {
	parent Logger
	prefix string
}

// NewPrefixer returns a Prefixer Logger.
//
//     pl := NewPrefixer(logger, "[REFRESH] ")  // make a prefix logger
//     pl.Dev("start handling: %f", 3.14)       // [REFRESH] start handling: 3.14
func NewPrefixer(logger Logger, prefix string) *Prefixer {
	return &Prefixer{parent: logger, prefix: prefix}
}

// Dev is used to inject an event considered interesting for developers into the
// log stream.
func (p *Prefixer) Dev(format string, args ...interface{}) error {
	return p.parent.Log(&Event{prefix: []string{p.prefix}, format: format, args: args, level: Dev})
}

// Admin is used to inject an event considered interesting for administrators
// into the log stream.
func (p *Prefixer) Admin(format string, args ...interface{}) error {
	return p.parent.Log(&Event{prefix: []string{p.prefix}, format: format, args: args, level: Admin})
}

// User is used to inject an event considered interesting for users into the log
// stream.
func (p *Prefixer) User(format string, args ...interface{}) error {
	return p.parent.Log(&Event{prefix: []string{p.prefix}, format: format, args: args, level: User})
}

// Log is used by Loggers that compose on top of this Prefixer Logger, and
// prefix events provided to it with the Prefixer's prefix.
func (p *Prefixer) Log(e *Event) error {
	e.prefix = append([]string{p.prefix}, e.prefix...)
	return p.parent.Log(e)
}

// Tracer Loggers log events with a tracer bit, that allows events to bypass
// filters. Additionally any events that pass through a Tracer Logger will have
// their tracer bit set, causing them to bypass filters on their way to the log.
type Tracer struct {
	parent Logger
	prefix string
}

// NewTracer returns a Tracer Logger.
//
//     tl := NewTracer(logger, "[QUERY-1234] ") // make a trace logger
//     tl.Dev("start handling: %f", 3.14)       // [QUERY-1234] start handling: 3.14
func NewTracer(logger Logger, prefix string) *Tracer {
	return &Tracer{parent: logger, prefix: prefix}
}

// Dev is used to inject an event considered interesting for developers into the
// log stream. Events logged to a Tracer Logger will pass through any configured
// Filter Loggers below it.
func (l *Tracer) Dev(format string, args ...interface{}) error {
	return l.parent.Log(&Event{prefix: []string{l.prefix}, format: format, args: args, level: Dev | 4})
}

// Admin is used to inject an event considered interesting for administrators
// into the log stream. Events logged to a Tracer Logger will pass through any
// configured Filter Loggers below it.
func (l *Tracer) Admin(format string, args ...interface{}) error {
	return l.parent.Log(&Event{prefix: []string{l.prefix}, format: format, args: args, level: Admin | 4})
}

// User is used to inject an event considered interesting for users into the log
// stream. Events logged to a Tracer Logger will pass through any configured
// Filter Loggers below it.
func (l *Tracer) User(format string, args ...interface{}) error {
	return l.parent.Log(&Event{prefix: []string{l.prefix}, format: format, args: args, level: User | 4})
}

// Log is used by Loggers that compose on top of this Tracer Logger, and prefix
// events provided to it with the Tracer's prefix, and set the tracer bit so
// that events will pass through Filter Loggers below it.
//
// ??? not sure whether this should be provided. Do we really want Tracer to be
// used to build loggers on top of? Tracers are supposed to be light-weight and
// ephemeral.
func (l *Tracer) Log(e *Event) error {
	e.level |= 4
	e.prefix = append([]string{l.prefix}, e.prefix...)
	return l.parent.Log(e)
}

// compileFormat converts the format string into a slice of functions to invoke
// when creating a log line.  It's implemented as a state machine that
// alternates between 2 states: consuming runes to create a constant string to
// emit, and consuming runes to create a token that is intended to match one of
// the pre-defined format specifier tokens, or an undefined format specifier
// token that begins with "http-".
func compileFormat(format string) ([]func(*Event, *[]byte), error) {
	// build slice of emitter functions, each will emit the requested
	// information
	var emitters []func(*Event, *[]byte)

	// state machine alternating between two states: either capturing runes for
	// the next constant buffer, or capturing runes for the next token
	var buf, token []byte
	var capturingTokenIndex int
	var capturingToken bool  // false, because start off capturing buffer runes
	var nextRuneEscaped bool // true when next rune has been escaped

	for ri, rune := range format {
		if nextRuneEscaped {
			// when this rune has been escaped, then just write it out to
			// whichever buffer we're collecting to right now
			if capturingToken {
				appendRune(&token, rune)
			} else {
				appendRune(&buf, rune)
			}
			nextRuneEscaped = false
			continue
		}
		if rune == '\\' {
			// Format specifies that next rune ought to be escaped.  Handy when
			// extra curly braces are desired in the log line format.
			nextRuneEscaped = true
			continue
		}
		if rune == '{' {
			if capturingToken {
				return nil, fmt.Errorf("cannot compile log format with embedded curly braces; runes %d and %d", capturingTokenIndex, ri)
			}
			// Stop capturing buf, and begin capturing token.  NOTE: Because I
			// did not want to allow Base Logger creation to fail, undefined
			// behavior if open curly brace when previous open curly brace has
			// not yet been closed.
			emitters = append(emitters, makeStringEmitter(string(buf)))
			buf = buf[:0]
			capturingToken = true
			capturingTokenIndex = ri
		} else if rune == '}' {
			if !capturingToken {
				return nil, fmt.Errorf("cannot compile log format with unmatched closing curly braces; rune %d", ri)
			}
			// Stop capturing token, and begin capturing buffer.
			switch tok := string(token); tok {
			case "epoch":
				emitters = append(emitters, epochEmitter)
			case "iso8601":
				emitters = append(emitters, makeUTCTimestampEmitter(time.RFC3339))
			case "level":
				emitters = append(emitters, levelEmitter)
			case "message":
				emitters = append(emitters, messageEmitter)
			case "program":
				emitters = append(emitters, makeProgramEmitter())
			case "timestamp":
				// Emulate timestamp format from stdlib log (log.LstdFlags).
				emitters = append(emitters, makeUTCTimestampEmitter("2006/01/02 15:04:05"))
			default:
				// ??? Not sure how I feel about the below API.
				if strings.HasPrefix(tok, "localtime=") {
					emitters = append(emitters, makeLocalTimestampEmitter(tok[10:]))
				} else if strings.HasPrefix(tok, "utctime=") {
					emitters = append(emitters, makeUTCTimestampEmitter(tok[8:]))
				} else {
					return nil, fmt.Errorf("cannot compile log format with unknown formatting verb %q", token)
				}
			}
			token = token[:0]
			capturingToken = false
		} else {
			// append to either token or buffer
			if capturingToken {
				appendRune(&token, rune)
			} else {
				appendRune(&buf, rune)
			}
		}
	}
	if capturingToken {
		return nil, fmt.Errorf("cannot compile log format with unmatched opening curly braces; rune %d", capturingTokenIndex)
	}

	buf = append(buf, '\n') // each log line terminated by newline byte
	emitters = append(emitters, makeStringEmitter(string(buf)))

	return emitters, nil
}

func appendRune(buf *[]byte, r rune) {
	if r < utf8.RuneSelf {
		*buf = append(*buf, byte(r))
		return
	}
	olen := len(*buf)
	*buf = append(*buf, 0, 0, 0, 0)              // grow buf large enough to accommodate largest possible UTF8 sequence
	n := utf8.EncodeRune((*buf)[olen:olen+4], r) // encode rune into newly allocated buf space
	*buf = (*buf)[:olen+n]                       // trim buf to actual size used by rune addition
}

func epochEmitter(e *Event, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(e.when.UTC().Unix(), 10)...)
}

func levelEmitter(e *Event, bb *[]byte) {
	*bb = append(*bb, e.level.String()...)
}

var program string

func makeProgramEmitter() func(e *Event, bb *[]byte) {
	if program == "" {
		var err error
		program, err = os.Executable()
		if err != nil {
			program = os.Args[0]
		}
		program = filepath.Base(program)
	}
	return func(e *Event, bb *[]byte) {
		*bb = append(*bb, program...)
	}
}

func makeStringEmitter(value string) func(*Event, *[]byte) {
	return func(_ *Event, bb *[]byte) {
		*bb = append(*bb, value...)
	}
}

func makeLocalTimestampEmitter(format string) func(e *Event, bb *[]byte) {
	return func(e *Event, bb *[]byte) {
		*bb = append(*bb, e.when.Format(format)...)
	}
}

func makeUTCTimestampEmitter(format string) func(e *Event, bb *[]byte) {
	return func(e *Event, bb *[]byte) {
		*bb = append(*bb, e.when.UTC().Format(format)...)
	}
}

func messageEmitter(e *Event, bb *[]byte) {
	*bb = append(*bb, strings.Join(e.prefix, "")...)       // emit the event's prefix
	*bb = append(*bb, fmt.Sprintf(e.format, e.args...)...) // followed by the event message
}
