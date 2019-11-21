package gologs

import (
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
const DefaultCommandFormat = "{program}: {message}"

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
	// NOT REACHED
	panic(fmt.Sprintf("invalid log level: %v", uint32(l)))
}

// event instances are created by loggers and flow through the log tree from the
// branch where they were created, down to the base, at which point, its
// arguments will be formatted immediately prior to writing the log message to
// the underlying log io.Writer.
type event struct {
	args   []interface{}
	prefix []string
	when   time.Time
	format string
	level  Level
}

type logger interface {
	log(*event) error
}

// base is at the bottom of the logger tree, and formats the event to a byte
// slice, ensuring it ends with a newline, and writes its output to its
// underlying io.Writer.
type base struct {
	formatters []func(*event, *[]byte)
	w          io.Writer
	c          int // c is count of bytes to allocate for formatting log line
	m          sync.Mutex
}

func (b *base) log(e *event) error {
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

// Logger provides methods to create events to be logged. Logger instances are
// created to emit events to their parent Logger instance, which may themselves
// either filter events based on a configured level, or prefix events with a
// configured string.
type Logger struct {
	prefix string // prefix is an option string, that when not empty, will prefix events
	parent logger // parent is the logger this branch sends events to
	level  Level  // level is the independent log level controls for this branch
	tracer Level  // tracer is an optional value that is boolean ORd with an event, so events created by this branch will pass through possible log level controls below.
}

// New returns a new Logger instance that emits logged events to w after
// formatting the event according to template.
func New(w io.Writer, template string) (*Logger, error) {
	formatters, err := compileFormat(template)
	if err != nil {
		return nil, err
	}
	// Create a dummy event to see how long the log line is with the provided
	// template.
	buf := make([]byte, 0, 64)
	var e event
	for _, formatter := range formatters {
		formatter(&e, &buf)
	}
	min := len(buf) + 64
	if min < 128 {
		min = 128
	}
	return &Logger{parent: &base{w: w, formatters: formatters, c: min}}, nil
}

// NewBranch returns a new Logger instance that logs to parent, but has its own
// log level that is independently controlled from parent.
//
// Note that events are filtered as the flow from their origin branch to the
// base. When a parent Logger has a more restrictive log level than a child
// Logger, the event might pass through from a child to its parent, but be
// filtered out at the parent.
func NewBranch(parent *Logger) *Logger {
	return &Logger{parent: parent}
}

// NewBranchWithPrefix returns a new Logger instance that logs to parent, but
// has its own log level that is independently controlled from
// parent. Furthermore, events that pass through the returned Logger will have
// prefix string prefixed to the event.
//
// Note that events are filtered as the flow from their origin branch to the
// base. When a parent Logger has a more restrictive log level than a child
// Logger, the event might pass through from a child to its parent, but be
// filtered out at the parent.
func NewBranchWithPrefix(parent *Logger, prefix string) *Logger {
	return &Logger{parent: parent, prefix: prefix}
}

// NewTracer returns a new Logger instance that sets the tracer bit for events
// that are logged to it.
//
//     tl := NewTracer(logger, "[QUERY-1234] ") // make a trace logger
//     tl.Dev("start handling: %f", 3.14)       // [QUERY-1234] start handling: 3.14
func NewTracer(parent *Logger, prefix string) *Logger {
	return &Logger{parent: parent, prefix: prefix, tracer: 4}
}

func (b *Logger) log(e *event) error {
	if b.tracer == 0 && Level(atomic.LoadUint32((*uint32)(&b.level))) > e.level {
		return nil
	}
	if b.prefix != "" {
		e.prefix = append([]string{b.prefix}, e.prefix...)
	}
	return b.parent.log(e)
}

// SetLevel allows changing the log level. Events must have the same log level
// or higher for events to be logged.
func (b *Logger) SetLevel(level Level) *Logger {
	atomic.StoreUint32((*uint32)(&b.level), uint32(level))
	return b
}

// SetDev changes the log level to Dev, which allows all events to be logged.
func (b *Logger) SetDev() *Logger {
	atomic.StoreUint32((*uint32)(&b.level), uint32(Dev))
	return b
}

// SetAdmin changes the log level to Admin, which allows all Admin and User
// events to be logged, and Dev events to be ignored.
func (b *Logger) SetAdmin() *Logger {
	atomic.StoreUint32((*uint32)(&b.level), uint32(Admin))
	return b
}

// SetUser changes the log level to User, which allows all User events to be
// logged, and ignores Dev and Admin level events.
func (b *Logger) SetUser() *Logger {
	atomic.StoreUint32((*uint32)(&b.level), uint32(User))
	return b
}

// Dev is used to inject an event considered interesting for developers into the
// log stream. Note the logger must have been set to the Dev log level for this
// event to be logged.
func (b *Logger) Dev(format string, args ...interface{}) error {
	if Level(atomic.LoadUint32((*uint32)(&b.level))) > Dev {
		return nil
	}
	var prefix []string
	if b.prefix != "" {
		prefix = []string{b.prefix}
	}
	return b.parent.log(&event{format: format, args: args, prefix: prefix, level: Dev | b.tracer})
}

// Admin is used to inject an event considered interesting for administrators
// into the log stream. Note the logger must have been set to the Dev or Admin
// level for this event to be logged.
func (b *Logger) Admin(format string, args ...interface{}) error {
	if Level(atomic.LoadUint32((*uint32)(&b.level))) > Admin {
		return nil
	}
	var prefix []string
	if b.prefix != "" {
		prefix = []string{b.prefix}
	}
	return b.parent.log(&event{format: format, args: args, prefix: prefix, level: Admin | b.tracer})
}

// User is used to inject an event considered interesting for users into the log
// stream. The created event will be logged regardless of the log level of the
// logger, as User events are considered the highest priority events.
func (b *Logger) User(format string, args ...interface{}) error {
	var prefix []string
	if b.prefix != "" {
		prefix = []string{b.prefix}
	}
	return b.parent.log(&event{format: format, args: args, prefix: prefix, level: User | b.tracer})
}

// compileFormat converts the format string into a slice of functions to invoke
// when creating a log line.  It's implemented as a state machine that
// alternates between 2 states: consuming runes to create a constant string to
// emit, and consuming runes to create a token that is intended to match one of
// the pre-defined format specifier tokens, or an undefined format specifier
// token that begins with "http-".
func compileFormat(format string) ([]func(*event, *[]byte), error) {
	// build slice of emitter functions, each will emit the requested
	// information
	var emitters []func(*event, *[]byte)

	// state machine alternating between two states: either capturing runes for
	// the next constant buffer, or capturing runes for the next token
	var buf, token []byte
	var capturingTokenIndex int
	var capturingToken bool  // false, because start off capturing buffer runes
	var nextRuneEscaped bool // true when next rune has been escaped
	var isFinalNewlineNeeded bool

	for ri, rune := range format {
		isFinalNewlineNeeded = rune != '\n'
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

	if isFinalNewlineNeeded {
		buf = append(buf, '\n') // each log line terminated by newline byte
	}
	if len(buf) > 0 {
		emitters = append(emitters, makeStringEmitter(string(buf)))
	}

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

func epochEmitter(e *event, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(e.when.UTC().Unix(), 10)...)
}

func levelEmitter(e *event, bb *[]byte) {
	*bb = append(*bb, e.level.String()...)
}

var program string

func makeProgramEmitter() func(e *event, bb *[]byte) {
	if program == "" {
		var err error
		program, err = os.Executable()
		if err != nil {
			program = os.Args[0]
		}
		program = filepath.Base(program)
	}
	return func(e *event, bb *[]byte) {
		*bb = append(*bb, program...)
	}
}

func makeStringEmitter(value string) func(*event, *[]byte) {
	return func(_ *event, bb *[]byte) {
		*bb = append(*bb, value...)
	}
}

func makeLocalTimestampEmitter(format string) func(e *event, bb *[]byte) {
	return func(e *event, bb *[]byte) {
		*bb = append(*bb, e.when.Format(format)...)
	}
}

func makeUTCTimestampEmitter(format string) func(e *event, bb *[]byte) {
	return func(e *event, bb *[]byte) {
		*bb = append(*bb, e.when.UTC().Format(format)...)
	}
}

func messageEmitter(e *event, bb *[]byte) {
	*bb = append(*bb, strings.Join(e.prefix, "")...)       // emit the event's prefix ???
	*bb = append(*bb, fmt.Sprintf(e.format, e.args...)...) // followed by the event message
}
