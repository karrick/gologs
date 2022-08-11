package gologs

// Intermediate is an intermediate Logger that is not capable of logging
// events, but used while creating a new Logger that always includes one or
// more properties in each logged event.
//
// Logger.With() -> *Intermediate -> Bool() -> *Intermediate -> ... -> Logger() -> *Logger
type Intermediate struct {
	branch        []byte // branch holds potentially empty prefix of each log event
	timeFormatter TimeFormatter
	output        *output
	level         uint32
	tracing       bool
}

// Bool returns a new Intermediate Logger that has the name property set to
// the JSON encoded bool value.
func (il *Intermediate) Bool(name string, value bool) *Intermediate {
	il.branch = appendBool(il.branch, name, value)
	return il
}

// Float returns a new Intermediate Logger that has the name property set to
// the JSON encoded float64 value.
func (il *Intermediate) Float(name string, value float64) *Intermediate {
	il.branch = appendFloat(il.branch, name, value)
	return il
}

// Format returns a new Intermediate Logger that has the name property set to
// the JSON encoded string value derived from the formatted string and its
// arguments. This function will invoke fmt.Sprintf() function to format the
// formatting string with the provided arguments, allocating memory to do
// so. If no formatting is required, invoking Intermediate.String(string,
// string) will be faster.
func (il *Intermediate) Format(name, f string, args ...interface{}) *Intermediate {
	il.branch = appendFormat(il.branch, name, f, args...)
	return il
}

// Int returns a new Intermediate Logger that has the name property set to the
// JSON encoded int value.
func (il *Intermediate) Int(name string, value int) *Intermediate {
	il.branch = appendInt(il.branch, name, int64(value))
	return il
}

// Int64 returns a new Intermediate Logger that has the name property set to
// the JSON encoded int64 value.
func (il *Intermediate) Int64(name string, value int64) *Intermediate {
	il.branch = appendInt(il.branch, name, value)
	return il
}

// Logger converts the Intermediate Logger into a new Logger instance that
// includes the fields it was configured to contain.
func (il *Intermediate) Logger() *Logger {
	log := &Logger{
		event: Event{
			scratch:       make([]byte, 1, 2048),
			timeFormatter: il.timeFormatter,
			output:        il.output,
		},
		level:   il.level,
		tracing: il.tracing,
	}
	if cap(il.branch) > 0 {
		log.branch = make([]byte, len(il.branch), cap(il.branch))
		copy(log.branch, il.branch)
	}
	log.event.scratch[0] = '{'

	return log
}

// String returns a new Intermediate Logger that has the name property set to
// the JSON encoded string value.
func (il *Intermediate) String(name, value string) *Intermediate {
	il.branch = appendString(il.branch, name, value)
	return il
}

// Tracing returns a new Intermediate Logger that logs all events, regardless
// of the Logger level at the time an log event is created.
func (il *Intermediate) Tracing(value bool) *Intermediate {
	il.tracing = value
	return il
}

// Uint returns a new Intermediate Logger that has the name property set to
// the JSON encoded uint value.
func (il *Intermediate) Uint(name string, value uint) *Intermediate {
	il.branch = appendUint(il.branch, name, uint64(value))
	return il
}

// Uint64 returns a new Intermediate Logger that has the name property set to
// the JSON encoded uint64 value.
func (il *Intermediate) Uint64(name string, value uint64) *Intermediate {
	il.branch = appendUint(il.branch, name, value)
	return il
}
