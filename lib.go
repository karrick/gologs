package gologs

import (
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// DefaultLogFormat specifies a log format that is commonly used.
const DefaultLogFormat = "{timestamp} [{level}] {message}"

// Level type defines one of several possible log levels.
type Level uint32

const (
	Dev   Level = iota // show me all events
	Admin              // show me detailed operational events
	User               // show me major operational events
)

func (l Level) String() string {
	switch l {
	case Dev:
		return "DEV"
	case Admin:
		return "ADMIN"
	case User:
		return "USER"
	}
	panic(fmt.Sprintf("invalid log level: %v", uint32(l)))
}

type event struct {
	args   []interface{}
	format string
	prefix string
	level  Level
	when   time.Time
}

type Logger interface {
	Dev(string, ...interface{})
	Admin(string, ...interface{})
	User(string, ...interface{})
	log(*event)
}

// Base formats the event to a byte slice, ensuring it ends with a newline, and
// writes its output to its underlying io.Writer.
type Base struct {
	w          io.Writer
	l          sync.Mutex
	formatters []func(*event, *[]byte)
}

func New(w io.Writer, template string) *Base {
	return &Base{w: w, formatters: compileFormat(template)}
}

func (b *Base) log(e *event) {
	// ??? *if* want to sacrifice a bit of speed, might consider using a
	// pre-allocated byte slice to format the output.

	e.when = time.Now()

	p := make([]byte, 0, 128)

	// "{timestamp} [{level}] {message}"
	for _, formatter := range b.formatters {
		formatter(e, &p)
	}

	b.l.Lock()
	_, _ = b.w.Write(p)
	b.l.Unlock()
}

func (b *Base) Dev(format string, args ...interface{}) {
	b.log(&event{format: format, args: args, level: Dev})
}

func (b *Base) Admin(format string, args ...interface{}) {
	b.log(&event{format: format, args: args, level: Admin})
}

func (b *Base) User(format string, args ...interface{}) {
	b.log(&event{format: format, args: args, level: User})
}

// Filter Logger will only convey events at least the same level as the Filter
// is set for.
type Filter struct {
	logger Logger
	level  uint32
}

// NewFilter returns a Filter Logger.
func NewFilter(logger Logger) *Filter {
	return &Filter{logger: logger, level: uint32(User)}
}

func (l *Filter) SetLevel(level Level) *Filter {
	atomic.StoreUint32(&l.level, uint32(level))
	return l
}

func (l *Filter) SetDev() *Filter {
	atomic.StoreUint32(&l.level, uint32(Dev))
	return l
}

func (l *Filter) SetAdmin() *Filter {
	atomic.StoreUint32(&l.level, uint32(Admin))
	return l
}

func (l *Filter) SetUser() *Filter {
	atomic.StoreUint32(&l.level, uint32(User))
	return l
}

func (l *Filter) Dev(format string, args ...interface{}) {
	if Level(atomic.LoadUint32(&l.level)) > Dev {
		return
	}
	l.logger.log(&event{format: format, args: args, level: Dev})
}

func (l *Filter) Admin(format string, args ...interface{}) {
	if Level(atomic.LoadUint32(&l.level)) > Admin {
		return
	}
	l.logger.log(&event{format: format, args: args, level: Admin})
}

func (l *Filter) User(format string, args ...interface{}) {
	l.logger.log(&event{format: format, args: args, level: User})
}

func (l *Filter) log(e *event) {
	if Level(atomic.LoadUint32(&l.level)) > e.level {
		return
	}
	l.logger.log(e)
}

// Tracer Loggers log events with a tracer bit, that allows events to bypass
// filters. Additionally any events that pass through a Tracer Logger will have
// their tracer bit set, causing them to bypass filters on their way to the log.
type Tracer struct {
	logger Logger
	prefix string
}

// NewTracer returns a Tracer Logger.
//
//     tl := NewTracer(logger, "[QUERY-1234] ") // make a trace logger
//     tl.Dev("example: %f", 3.14)
func NewTracer(logger Logger, prefix string) *Tracer {
	return &Tracer{logger: logger, prefix: prefix}
}

func (l *Tracer) Dev(format string, args ...interface{}) {
	l.logger.log(&event{prefix: l.prefix, format: format, args: args, level: Dev | 4})
}

func (l *Tracer) Admin(format string, args ...interface{}) {
	l.logger.log(&event{prefix: l.prefix, format: format, args: args, level: Admin | 4})
}

func (l *Tracer) User(format string, args ...interface{}) {
	l.logger.log(&event{prefix: l.prefix, format: format, args: args, level: User | 4})
}

func (l *Tracer) log(e *event) {
	e.level |= 4
	e.prefix = l.prefix + e.prefix
	l.logger.log(e)
}

// compileFormat converts the format string into a slice of functions to invoke
// when creating a log line.  It's implemented as a state machine that
// alternates between 2 states: consuming runes to create a constant string to
// emit, and consuming runes to create a token that is intended to match one of
// the pre-defined format specifier tokens, or an undefined format specifier
// token that begins with "http-".
func compileFormat(format string) []func(*event, *[]byte) {
	// build slice of emitter functions, each will emit the requested
	// information
	var emitters []func(*event, *[]byte)

	// state machine alternating between two states: either capturing runes for
	// the next constant buffer, or capturing runes for the next token
	var buf, token []byte
	var capturingToken bool  // false, because start off capturing buffer runes
	var nextRuneEscaped bool // true when next rune has been escaped

	for _, rune := range format {
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
			// Stop capturing buf, and begin capturing token.
			// NOTE: undefined behavior if open curly brace when previous open
			// curly brace has not yet been closed.
			emitters = append(emitters, makeStringEmitter(string(buf)))
			buf = buf[:0]
			capturingToken = true
		} else if rune == '}' {
			// Stop capturing token, and begin capturing buffer.
			// NOTE: undefined behavior if close curly brace when not capturing
			// runes for a token.
			switch tok := string(token); tok {
			case "epoch":
				emitters = append(emitters, epochEmitter)
			case "iso8601":
				emitters = append(emitters, iSO8601Emitter)
			case "level":
				emitters = append(emitters, levelEmitter)
			case "message":
				emitters = append(emitters, messageEmitter)
			case "timestamp":
				emitters = append(emitters, timestampEmitter)
			default:
				// unknown token: just append to buf, wrapped in curly
				// braces
				buf = append(buf, '{')
				buf = append(buf, tok...)
				buf = append(buf, '}')
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
		buf = append(buf, '{') // token started with left curly brace, so it needs to precede the token
		buf = append(buf, token...)
	}
	buf = append(buf, '\n') // each log line terminated by newline byte
	emitters = append(emitters, makeStringEmitter(string(buf)))

	return emitters
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

func makeStringEmitter(value string) func(*event, *[]byte) {
	return func(_ *event, bb *[]byte) {
		*bb = append(*bb, value...)
	}
}

func epochEmitter(e *event, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(e.when.UTC().Unix(), 10)...)
}

func iSO8601Emitter(e *event, bb *[]byte) {
	*bb = append(*bb, e.when.UTC().Format(time.RFC3339)...)
}

func timestampEmitter(e *event, bb *[]byte) {
	// emulate timestamp format from stdlib log (log.LstdFlags)
	*bb = append(*bb, e.when.Format("2006/01/02 15:04:05")...)
}

func levelEmitter(e *event, bb *[]byte) {
	*bb = append(*bb, e.level.String()...)
}

func messageEmitter(e *event, bb *[]byte) {
	*bb = append(*bb, e.prefix...)
	*bb = append(*bb, fmt.Sprintf(e.format, e.args...)...)
}
