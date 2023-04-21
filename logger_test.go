package gologs

import (
	"bytes"
	"io"
	"testing"
)

// panicyWriter is a test structure used for this test that optionally panics.
type panicyWriter struct {
	w           io.Writer
	shouldPanic bool
}

func (pw *panicyWriter) Write(buf []byte) (int, error) {
	if pw.shouldPanic {
		panic("writer-boom!")
	}
	return pw.w.Write(buf)
}

func TestLogger(t *testing.T) {
	t.Run("panic protection", func(t *testing.T) {
		// The caller provides one or two dependencies that this structure
		// leverages. Either of them may panic when used, so we need to
		// provide testing to ensure this structure behaves properly when
		// either of those dependencies do panic.

		t.Run("writer", func(t *testing.T) {
			// Ensure properly handle when the io.Writer provided by the
			// caller panics during a Write call.
			bb := new(bytes.Buffer)
			pw := &panicyWriter{w: bb}
			log := New(pw).SetInfo()

			log.Info().Msg("message 1") // should not panic

			ensurePanic(t, "writer-boom!", func() {
				pw.shouldPanic = true
				log.Info().Msg("message 2")
			})

			pw.shouldPanic = false
			log.Info().Msg("message 3") // should not panic

			want := []byte("{\"level\":\"info\",\"message\":\"message 1\"}\n{\"level\":\"info\",\"message\":\"message 3\"}\n")
			ensureBytes(t, bb.Bytes(), want)
		})

		t.Run("time formatter", func(t *testing.T) {
			// Ensure properly handle when the time formatting function
			// provided by the caller panics while formatting the event time.
			bb := new(bytes.Buffer)
			log := New(bb).SetInfo()

			log.Info().Msg("message 1")

			log.SetTimeFormatter(func([]byte) []byte {
				panic("time-formatter-boom!")
			})

			log.Info().Msg("message 2") // this panics, but it should be handled

			log.SetTimeFormatter(func(buf []byte) []byte { return buf })
			log.Info().Msg("message 3")

			want := []byte("{\"level\":\"info\",\"message\":\"message 1\"}\n{\"error\":\"time-formatter-boom!\",\"message\":\"panic when time formatter invoked\"}\n{\"level\":\"info\",\"message\":\"message 3\"}\n")
			ensureBytes(t, bb.Bytes(), want)
		})
	})

	t.Run("cases", func(t *testing.T) {
		tests := []struct {
			name string
			want string
			call func(*Logger)
		}{
			{
				"level before threshold should not log",
				"",
				func(l *Logger) {
					l.Verbose().
						Bool("happy", true).
						Bool("sad", false).
						Float("usage", 42.3).
						Format("name", "%s %s", "First", "Last").
						Int("age", 42).
						String("eye-color", "brown").
						Uint("months", 123).
						Uint64("days", 1234).
						Msg("should not log")
				},
			},
			{
				"level at threshold should log",
				"{\"time\":123456789,\"level\":\"warning\",\"happy\":true,\"sad\":false,\"usage\":42.3,\"name\":\"First Last\",\"age\":42,\"i64\":42,\"eye-color\":\"brown\",\"months\":123,\"days\":1234,\"message\":\"should log\"}\n",
				func(l *Logger) {
					// Use custom time formatter to ensure it is called, and to be able to
					// use a specific time value for the purpose of validating the output.
					l.SetTimeFormatter(func(buf []byte) []byte {
						return append(buf, []byte(`"time":123456789,`)...)
					})

					l.Warning().
						Bool("happy", true).
						Bool("sad", false).
						Float("usage", 42.3).
						Format("name", "%s %s", "First", "Last").
						Int("age", 42).
						Int64("i64", 42).
						String("eye-color", "brown").
						Uint("months", 123).
						Uint64("days", 1234).
						Msg("should log")
				},
			},
			{
				"level above threshold should log",
				"{\"time\":123456789,\"level\":\"error\",\"happy\":true,\"sad\":false,\"usage\":42.3,\"name\":\"First Last\",\"age\":42,\"i64\":42,\"eye-color\":\"brown\",\"months\":123,\"days\":1234,\"message\":\"should log\"}\n",
				func(l *Logger) {
					// Use custom time formatter to ensure it is called, and to be able to
					// use a specific time value for the purpose of validating the output.
					l.SetTimeFormatter(func(buf []byte) []byte {
						return append(buf, []byte(`"time":123456789,`)...)
					})

					l.Error().
						Bool("happy", true).
						Bool("sad", false).
						Float("usage", 42.3).
						Format("name", "%s %s", "First", "Last").
						Int("age", 42).
						Int64("i64", 42).
						String("eye-color", "brown").
						Uint("months", 123).
						Uint64("days", 1234).
						Msg("should log")
				},
			},

			// Default level is warning
			{
				"debug when default level is warning",
				"",
				func(l *Logger) { l.Debug().Msg("some message") },
			},
			{
				"verbose when default level is warning",
				"",
				func(l *Logger) { l.Verbose().Msg("some message") },
			},
			{
				"info when default level is warning",
				"",
				func(l *Logger) { l.Info().Msg("some message") },
			},
			{
				"warning when default level is warning",
				"{\"level\":\"warning\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.Warning().Msg("some message") },
			},
			{
				"error when default level is warning",
				"{\"level\":\"error\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.Error().Msg("some message") },
			},

			// When level is debug
			{
				"debug when level is debug",
				"{\"level\":\"debug\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetDebug().Debug().Msg("some message") },
			},
			{
				"verbose when level is debug",
				"{\"level\":\"verbose\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetDebug().Verbose().Msg("some message") },
			},
			{
				"info when level is debug",
				"{\"level\":\"info\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetDebug().Info().Msg("some message") },
			},
			{
				"warning when level is debug",
				"{\"level\":\"warning\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetDebug().Warning().Msg("some message") },
			},
			{
				"error when level is debug",
				"{\"level\":\"error\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetDebug().Error().Msg("some message") },
			},

			// When level is verbose
			{
				"debug when level is verbose",
				"",
				func(l *Logger) { l.SetVerbose().Debug().Msg("some message") },
			},
			{
				"verbose when level is verbose",
				"{\"level\":\"verbose\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetVerbose().Verbose().Msg("some message") },
			},
			{
				"info when level is verbose",
				"{\"level\":\"info\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetVerbose().Info().Msg("some message") },
			},
			{
				"warning when level is verbose",
				"{\"level\":\"warning\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetVerbose().Warning().Msg("some message") },
			},
			{
				"error when level is verbose",
				"{\"level\":\"error\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetVerbose().Error().Msg("some message") },
			},

			// When level is info
			{
				"debug when level is info",
				"",
				func(l *Logger) { l.SetInfo().Debug().Msg("some message") },
			},
			{
				"verbose when level is info",
				"",
				func(l *Logger) { l.SetInfo().Verbose().Msg("some message") },
			},
			{
				"info when level is info",
				"{\"level\":\"info\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetInfo().Info().Msg("some message") },
			},
			{
				"warning when level is info",
				"{\"level\":\"warning\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetInfo().Warning().Msg("some message") },
			},
			{
				"error when level is info",
				"{\"level\":\"error\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetInfo().Error().Msg("some message") },
			},

			// When level is warning
			{
				"debug when level is warning",
				"",
				func(l *Logger) { l.SetWarning().Debug().Msg("some message") },
			},
			{
				"verbose when level is warning",
				"",
				func(l *Logger) { l.SetWarning().Verbose().Msg("some message") },
			},
			{
				"info when level is warning",
				"",
				func(l *Logger) { l.SetWarning().Info().Msg("some message") },
			},
			{
				"warning when level is warning",
				"{\"level\":\"warning\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetWarning().Warning().Msg("some message") },
			},
			{
				"error when level is warning",
				"{\"level\":\"error\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetWarning().Error().Msg("some message") },
			},

			// When level is error
			{
				"debug when level is error",
				"",
				func(l *Logger) { l.SetError().Debug().Msg("some message") },
			},
			{
				"verbose when level is error",
				"",
				func(l *Logger) { l.SetError().Verbose().Msg("some message") },
			},
			{
				"info when level is error",
				"",
				func(l *Logger) { l.SetError().Info().Msg("some message") },
			},
			{
				"warning when level is error",
				"",
				func(l *Logger) { l.SetError().Warning().Msg("some message") },
			},
			{
				"error when level is error",
				"{\"level\":\"error\",\"message\":\"some message\"}\n",
				func(l *Logger) { l.SetError().Error().Msg("some message") },
			},

			// errors
			{
				"error is nil",
				"{\"level\":\"warning\",\"pathname\":\"/some/path\",\"error\":null,\"message\":\"read file\"}\n",
				func(l *Logger) {
					l.Warning().String("pathname", "/some/path").Err(nil).Msg("read file")
				},
			},
			{
				"error is non-nil",
				"{\"level\":\"warning\",\"pathname\":\"/some/path\",\"error\":\"bytes.Buffer: too large\",\"message\":\"read file\"}\n",
				func(l *Logger) {
					l.Warning().String("pathname", "/some/path").Err(bytes.ErrTooLarge).Msg("read file")
				},
			},

			// branches and filtering
			{
				"different branches have different levels",
				`{"level":"verbose","module":"child1","message":"should be logged"}
{"level":"warning","module":"child2","message":"should be logged"}
`,
				func(l *Logger) {
					// NOTE: The log level of parent can be overridden by its
					// child branches. Set the parent to Error, and make sure
					// derived loggers still emit events.
					parent := l.SetError()

					child1 := parent.With().String("module", "child1").Logger().SetLevel(Verbose)
					child1.Debug().Msg("should not be logged")
					child1.Verbose().Msg("should be logged")

					child2 := parent.With().String("module", "child2").Logger().SetLevel(Warning)
					child2.Info().Msg("should not be logged")
					child2.Warning().Msg("should be logged")
				},
			},
			{
				"branches have cascading properties",
				"{\"level\":\"error\",\"module\":\"signals\",\"received\":\"term\",\"relay\":\"success\",\"float\":3.14}\n",
				func(l *Logger) {
					l.With().
						String("module", "signals").
						String("received", "term").
						String("relay", "success").
						Logger().
						Error().Float("float", 3.14).Msg("")
				},
			},

			// tracer
			{
				"all events logged when tracer is true",
				"{\"level\":\"verbose\",\"first\":\"first\",\"second\":\"second\",\"string\":\"hello\",\"float\":3.14}\n",
				func(l *Logger) {
					l = l.With().
						String("first", "first").
						String("second", "second").
						Tracing(true).
						Logger().
						SetError()
					// Normally a verbose message would not get through a
					// logger whose level is Error, however, this logger has
					// its tracing bit set.
					l.Verbose().String("string", "hello").Float("float", 3.14).Msg("")
				},
			},
		}

		for _, single := range tests {
			t.Run(single.name, func(t *testing.T) {
				bb := new(bytes.Buffer)
				single.call(New(bb))
				ensureBytes(t, bb.Bytes(), []byte(single.want))
			})
		}
	})
}

func BenchmarkLogger(b *testing.B) {
	bb := bytes.NewBuffer(make([]byte, 0, 4096))
	l := New(bb)

	b.Run("should not log", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			l.Debug().Bool("happy", true).Bool("sad", false).Msg("")
			ensureBytes(b, bb.Bytes(), nil)
			// NOTE: do not need to invoke bb.Reset() because nothing should be written.
		}
	})

	b.Run("should log", func(b *testing.B) {
		b.Run("sans string formatting", func(b *testing.B) {
			want := []byte("{\"level\":\"warning\",\"happy\":true,\"sad\":false,\"usage\":42.3,\"age\":42,\"eye-color\":\"brown\",\"months\":123,\"days\":1234,\"message\":\"should log\"}\n")

			for i := 0; i < b.N; i++ {
				l.Warning().
					Bool("happy", true).
					Bool("sad", false).
					Float("usage", 42.3).
					Int("age", 42).
					String("eye-color", "brown").
					Uint("months", 123).
					Uint64("days", 1234).
					Msg("should log")
				ensureBytes(b, bb.Bytes(), want)
				bb.Reset()
			}
		})

		b.Run("with string formatting", func(b *testing.B) {
			want := []byte("{\"level\":\"warning\",\"happy\":true,\"sad\":false,\"usage\":42.3,\"name\":\"First Last\",\"age\":42,\"eye-color\":\"brown\",\"months\":123,\"days\":1234,\"message\":\"with string formatting\"}\n")

			for i := 0; i < b.N; i++ {
				l.Warning().
					Bool("happy", true).
					Bool("sad", false).
					Float("usage", 42.3).
					Format("name", "%s %s", "First", "Last"). // NOTE: While formatting is not slow, it does require allocation.
					Int("age", 42).
					String("eye-color", "brown").
					Uint("months", 123).
					Uint64("days", 1234).
					Msg("with string formatting")
				ensureBytes(b, bb.Bytes(), want)
				bb.Reset()
			}
		})
	})
}
