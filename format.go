package gologs

import (
	"io"
	"strconv"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// FormattedLogger formats and logs events to its io.Writer based on its log
// Level.
type FormattedLogger struct {
	e  []func(event, *[]byte) // compiled emitters to build log line
	mw mutexWriter
	l  uint32 // current log level for this logger
}

// NewFormattedLogger returns a FormattedLogger that emits logs formatted in
// accordance with format to w.
//
// The following format string operators are recognized:
//   * {level}
//   * {message}
//   * Various time format strings:
//       {epoch}
//       {iso8601}
//       {timestamp}
func NewFormattedLogger(w io.Writer, format string) *FormattedLogger {
	return &FormattedLogger{
		e:  compileFormat(format),
		mw: mutexWriter{w: w},
		l:  uint32(Info),
	}
}

func (lp *FormattedLogger) SetLevel(l Level) { atomic.StoreUint32(&lp.l, uint32(l)) }
func (lp *FormattedLogger) SetQuiet()        { atomic.StoreUint32(&lp.l, uint32(Warning)) }
func (lp *FormattedLogger) SetVerbose()      { atomic.StoreUint32(&lp.l, uint32(Verbose)) }

func (lp *FormattedLogger) Debug(format string, a ...interface{}) {
	if Level(atomic.LoadUint32(&lp.l)) == Debug {
		lp.mw.Write(lp.format(newEvent(Debug, format, a...)))
	}
}

func (lp *FormattedLogger) Verbose(format string, a ...interface{}) {
	if Level(atomic.LoadUint32(&lp.l)) <= Verbose {
		lp.mw.Write(lp.format(newEvent(Verbose, format, a...)))
	}
}

func (lp *FormattedLogger) Info(format string, a ...interface{}) {
	if Level(atomic.LoadUint32(&lp.l)) <= Info {
		lp.mw.Write(lp.format(newEvent(Info, format, a...)))
	}
}

func (lp *FormattedLogger) Warning(format string, a ...interface{}) {
	if Level(atomic.LoadUint32(&lp.l)) <= Warning {
		lp.mw.Write(lp.format(newEvent(Warning, format, a...)))
	}
}

func (lp *FormattedLogger) Error(format string, a ...interface{}) {
	lp.mw.Write(lp.format(newEvent(Error, format, a...)))
}

func (lp *FormattedLogger) format(e event) []byte {
	buf := make([]byte, 0, 128)
	for _, emitter := range lp.e {
		emitter(e, &buf)
	}
	// The final byte must always be newline, even if empty log, for whatever
	// reason.
	if l := len(buf); l == 0 || buf[l-1] != '\n' {
		return append(buf, '\n')
	}
	return buf
}

// DefaultLogFormat specifies a log format that is commonly used.
const DefaultLogFormat = "{timestamp} [{level}] {message}"

// compileFormat converts the format string into a slice of functions to invoke
// when creating a log line.  It's implemented as a state machine that
// alternates between 2 states: consuming runes to create a constant string to
// emit, and consuming runes to create a token that is intended to match one of
// the pre-defined format specifier tokens, or an undefined format specifier
// token that begins with "http-".
func compileFormat(format string) []func(event, *[]byte) {
	// build slice of emitter functions, each will emit the requested
	// information
	var emitters []func(event, *[]byte)

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

func makeStringEmitter(value string) func(event, *[]byte) {
	return func(_ event, bb *[]byte) {
		*bb = append(*bb, value...)
	}
}

func epochEmitter(e event, bb *[]byte) {
	*bb = append(*bb, strconv.FormatInt(e.when.UTC().Unix(), 10)...)
}

func iSO8601Emitter(e event, bb *[]byte) {
	*bb = append(*bb, e.when.UTC().Format(time.RFC3339)...)
}

func timestampEmitter(e event, bb *[]byte) {
	// emulate timestamp format from stdlib log (log.LstdFlags)
	*bb = append(*bb, e.when.Format("2006/01/02 15:04:05")...)
}

func levelEmitter(e event, bb *[]byte) {
	switch e.level {
	case Debug:
		*bb = append(*bb, "DEBUG"...)
	case Verbose:
		*bb = append(*bb, "VERBOSE"...)
	case Info:
		*bb = append(*bb, "INFO"...)
	case Warning:
		*bb = append(*bb, "WARNING"...)
	case Error:
		*bb = append(*bb, "ERROR"...)
	}
}

func messageEmitter(e event, bb *[]byte) {
	*bb = append(*bb, e.message...)
}
