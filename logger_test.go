package gologs

import (
	"bytes"
	"io"
	"testing"
)

func TestLogger(t *testing.T) {
	t.Run("panic protection", func(t *testing.T) {
		// The caller provides one or two dependencies that this structure
		// leverages. Either of them may panic when used, so we need to
		// provide testing to ensure this structure behaves properly when
		// either of those dependencies do panic.

		t.Run("writer", func(t *testing.T) {
			bb := new(bytes.Buffer)
			pw := &panicyWriter{w: bb} // a test structure used for this test that optionally panics
			log := New(pw).SetInfo()

			log.Info().Msg("message 1")

			ensurePanic(t, "writer-boom!", func() {
				pw.isTriggered = true
				log.Info().Msg("message 2")
			})

			pw.isTriggered = false
			log.Info().Msg("message 3")

			want := []byte("{\"level\":\"info\",\"message\":\"message 1\"}\n{\"level\":\"info\",\"message\":\"message 3\"}\n")
			ensureBytes(t, bb.Bytes(), want)
		})

		t.Run("time formatter", func(t *testing.T) {
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

	t.Run("should not log", func(t *testing.T) {
		bb := new(bytes.Buffer)
		f := New(bb)
		f.SetLevel(Error)
		f.SetTimeFormatter(TimeUnixNano)
		f.Debug().
			Bool("happy", true).
			Bool("sad", false).
			Float("usage", 42.3).
			Format("name", "%s %s", "First", "Last").
			Int("age", 42).
			String("eye-color", "brown").
			Uint("months", 123).
			Uint64("days", 1234).
			Msg("should not log")

		ensureBytes(t, bb.Bytes(), nil)
	})

	t.Run("should log", func(t *testing.T) {
		bb := new(bytes.Buffer)
		f := New(bb)
		f.SetLevel(Debug)

		// Use custom time formatter to ensure it is called, and to be able to
		// use a specific time value for the purpose of validating the output.
		f.SetTimeFormatter(func(buf []byte) []byte {
			return append(buf, []byte(`"time":123456789,`)...)
		})

		f.Debug().
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

		want := []byte("{\"time\":123456789,\"level\":\"debug\",\"happy\":true,\"sad\":false,\"usage\":42.3,\"name\":\"First Last\",\"age\":42,\"i64\":42,\"eye-color\":\"brown\",\"months\":123,\"days\":1234,\"message\":\"should log\"}\n")

		ensureBytes(t, bb.Bytes(), want)
	})

	t.Run("errors", func(t *testing.T) {
		t.Run("nil", func(t *testing.T) {
			bb := new(bytes.Buffer)

			New(bb).Warning().String("pathname", "/some/path").Err(nil).Msg("read file")

			want := []byte("{\"level\":\"warning\",\"pathname\":\"\\/some\\/path\",\"error\":null,\"message\":\"read file\"}\n")
			ensureBytes(t, bb.Bytes(), want)
		})

		t.Run("non-nil", func(t *testing.T) {
			bb := new(bytes.Buffer)

			New(bb).Warning().String("pathname", "/some/path").Err(bytes.ErrTooLarge).Msg("read file")

			want := []byte("{\"level\":\"warning\",\"pathname\":\"\\/some\\/path\",\"error\":\"bytes.Buffer: too large\",\"message\":\"read file\"}\n")
			ensureBytes(t, bb.Bytes(), want)
		})
	})

	t.Run("branches", func(t *testing.T) {
		t.Run("filtering", func(t *testing.T) {
			bb := new(bytes.Buffer)
			parent := New(bb).SetLevel(Debug)

			child1 := parent.NewBranchWithString("module", "child1").SetLevel(Verbose)
			child1.Debug().Msg("should not be logged")
			child1.Verbose().Msg("should be logged")

			child2 := parent.NewBranchWithString("module", "child2").SetLevel(Warning)
			child2.Info().Msg("should not be logged")
			child2.Warning().Msg("should be logged")

			want := []byte(`{"level":"verbose","module":"child1","message":"should be logged"}
{"level":"warning","module":"child2","message":"should be logged"}
`)

			ensureBytes(t, bb.Bytes(), want)
		})

		t.Run("cascading", func(t *testing.T) {
			check := func(t *testing.T, want string, callback func(*Logger)) {
				t.Helper()
				bb := new(bytes.Buffer)
				log := New(bb).NewBranchWithString("module", "signals")
				callback(log)
				ensureBytes(t, bb.Bytes(), []byte(want))
			}

			check(t, "{\"level\":\"error\",\"module\":\"signals\",\"float\":3.14}\n", func(l *Logger) {
				l.Error().Float("float", 3.14).Msg("")
			})

			check(t, "{\"level\":\"error\",\"module\":\"signals\",\"received\":\"int\",\"float\":3.14}\n", func(l *Logger) {
				l.NewBranchWithString("received", "int").Error().Float("float", 3.14).Msg("")
			})

			check(t, "{\"level\":\"error\",\"module\":\"signals\",\"received\":\"int\",\"relay\":\"success\",\"float\":3.14}\n", func(l *Logger) {
				l.NewBranchWithString("received", "int").NewBranchWithString("relay", "success").Error().Float("float", 3.14).Msg("")
			})
		})
	})

	t.Run("filter", func(t *testing.T) {
		const message = "some message"

		check := func(t *testing.T, want string, callback func(*Logger)) {
			t.Helper()
			bb := new(bytes.Buffer)
			log := New(bb)
			callback(log)
			ensureBytes(t, bb.Bytes(), []byte(want))
		}

		// default logger mode is warning
		check(t, "", func(f *Logger) { f.Debug().Msg(message) })
		check(t, "", func(f *Logger) { f.Verbose().Msg(message) })
		check(t, "", func(f *Logger) { f.Info().Msg(message) })
		check(t, "{\"level\":\"warning\",\"message\":\"some message\"}\n", func(f *Logger) { f.Warning().Msg(message) })
		check(t, "{\"level\":\"error\",\"message\":\"some message\"}\n", func(f *Logger) { f.Error().Msg(message) })

		check(t, "{\"level\":\"debug\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetDebug().Debug().Msg(message) })
		check(t, "{\"level\":\"verbose\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetDebug().Verbose().Msg(message) })
		check(t, "{\"level\":\"info\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetDebug().Info().Msg(message) })
		check(t, "{\"level\":\"warning\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetDebug().Warning().Msg(message) })
		check(t, "{\"level\":\"error\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetDebug().Error().Msg(message) })

		check(t, "", func(f *Logger) { f.SetVerbose().Debug().Msg(message) })
		check(t, "{\"level\":\"verbose\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetVerbose().Verbose().Msg(message) })
		check(t, "{\"level\":\"info\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetVerbose().Info().Msg(message) })
		check(t, "{\"level\":\"warning\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetVerbose().Warning().Msg(message) })
		check(t, "{\"level\":\"error\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetVerbose().Error().Msg(message) })

		check(t, "", func(f *Logger) { f.SetInfo().Debug().Msg(message) })
		check(t, "", func(f *Logger) { f.SetInfo().Verbose().Msg(message) })
		check(t, "{\"level\":\"info\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetInfo().Info().Msg(message) })
		check(t, "{\"level\":\"warning\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetInfo().Warning().Msg(message) })
		check(t, "{\"level\":\"error\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetInfo().Error().Msg(message) })

		check(t, "", func(f *Logger) { f.SetWarning().Debug().Msg(message) })
		check(t, "", func(f *Logger) { f.SetWarning().Verbose().Msg(message) })
		check(t, "", func(f *Logger) { f.SetWarning().Info().Msg(message) })
		check(t, "{\"level\":\"warning\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetWarning().Warning().Msg(message) })
		check(t, "{\"level\":\"error\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetWarning().Error().Msg(message) })

		check(t, "", func(f *Logger) { f.SetError().Debug().Msg(message) })
		check(t, "", func(f *Logger) { f.SetError().Verbose().Msg(message) })
		check(t, "", func(f *Logger) { f.SetError().Info().Msg(message) })
		check(t, "", func(f *Logger) { f.SetError().Warning().Msg(message) })
		check(t, "{\"level\":\"error\",\"message\":\"some message\"}\n", func(f *Logger) { f.SetError().Error().Msg(message) })
	})

	t.Run("tracer", func(t *testing.T) {
		bb := new(bytes.Buffer)

		log := New(bb).
			SetError().
			NewBranchWithString("first", "first").
			NewBranchWithString("second", "second").
			SetTracing(true)

		// Normally a verbose message would not get through a logger whose
		// level is Error, however, this logger has its tracing bit set.
		log.Verbose().String("string", "hello").Float("float", 3.14).Msg("")

		want := []byte("{\"level\":\"verbose\",\"first\":\"first\",\"second\":\"second\",\"string\":\"hello\",\"float\":3.14}\n")
		ensureBytes(t, bb.Bytes(), want)
	})
}

type panicyWriter struct {
	w           io.Writer
	isTriggered bool
}

func (pw *panicyWriter) Write(buf []byte) (int, error) {
	if pw.isTriggered {
		panic("writer-boom!")
	}
	return pw.w.Write(buf)
}

func BenchmarkLogger(b *testing.B) {
	bb := bytes.NewBuffer(make([]byte, 0, 4096))
	f := New(bb)

	b.Run("should not log", func(b *testing.B) {
		f.SetLevel(Error)

		for i := 0; i < b.N; i++ {
			f.Debug().Bool("happy", true).Bool("sad", false).Msg("")
			ensureBytes(b, bb.Bytes(), nil)

			// NOTE: do not need to invoke bb.Reset() because nothing should be written.
		}
	})

	b.Run("should log", func(b *testing.B) {
		b.Run("without string formatting", func(b *testing.B) {
			want := []byte("{\"level\":\"debug\",\"happy\":true,\"sad\":false,\"usage\":42.3,\"age\":42,\"eye-color\":\"brown\",\"months\":123,\"days\":1234,\"message\":\"should log\"}\n")

			f.SetLevel(Debug)

			for i := 0; i < b.N; i++ {
				f.Debug().
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
			want := []byte("{\"level\":\"info\",\"happy\":true,\"sad\":false,\"usage\":42.3,\"name\":\"First Last\",\"age\":42,\"eye-color\":\"brown\",\"months\":123,\"days\":1234,\"message\":\"with string formatting\"}\n")

			f.SetLevel(Debug)

			for i := 0; i < b.N; i++ {
				f.Info().
					Bool("happy", true).
					Bool("sad", false).
					Float("usage", 42.3).
					Format("name", "%s %s", "First", "Last").
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
