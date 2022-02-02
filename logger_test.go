package gologs

import (
	"bytes"
	"testing"
)

func TestLogger(t *testing.T) {
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
			Msg("should not log")

		if got, want := bb.Bytes(), []byte(""); !bytes.Equal(got, want) {
			t.Errorf("GOT: %q; WANT: %q", string(got), string(want))
		}
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
			String("eye-color", "brown").
			Msg("should log")

		want := []byte("{\"time\":123456789,\"level\":\"debug\",\"happy\":true,\"sad\":false,\"usage\":42.3,\"name\":\"First Last\",\"age\":42,\"eye-color\":\"brown\",\"message\":\"should log\"}\n")

		if got := bb.Bytes(); !bytes.Equal(got, want) {
			t.Errorf("GOT: %q; WANT: %q", string(got), string(want))
		}
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

			if got := bb.Bytes(); !bytes.Equal(got, want) {
				t.Errorf("GOT:\n\t%v\nWANT:\n\t%v\n", string(got), string(want))
			}
		})

		t.Run("cascading", func(t *testing.T) {
			check := func(t *testing.T, want string, callback func(*Logger)) {
				t.Helper()
				bb := new(bytes.Buffer)
				log := New(bb).NewBranchWithString("module", "signals")
				callback(log)
				if got := string(bb.Bytes()); got != want {
					t.Errorf("GOT:\n\t%q\n\tWANT:\n\t%q", got, want)
				}
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
			if got := string(bb.Bytes()); got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
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
}

func BenchmarkLogger(b *testing.B) {
	bb := bytes.NewBuffer(make([]byte, 0, 4096))
	f := New(bb)

	b.Run("should not log", func(b *testing.B) {
		f.SetLevel(Error)

		for i := 0; i < b.N; i++ {
			f.Debug().Bool("happy", true).Bool("sad", false).Msg("")
			if got, want := bb.Bytes(), []byte{}; !bytes.Equal(got, want) {
				b.Errorf("GOT: %v; WANT: %v", string(got), string(want))
			}

			// NOTE: do not need to invoke bb.Reset() because nothing should be written.
		}
	})

	b.Run("should log", func(b *testing.B) {
		b.Run("without string formatting", func(b *testing.B) {
			want := []byte("{\"level\":\"debug\",\"happy\":true,\"sad\":false,\"usage\":42.3,\"age\":42,\"eye-color\":\"brown\",\"message\":\"should log\"}\n")

			f.SetLevel(Debug)

			for i := 0; i < b.N; i++ {
				f.Debug().
					Bool("happy", true).
					Bool("sad", false).
					Float("usage", 42.3).
					Int("age", 42).
					String("eye-color", "brown").
					Msg("should log")

				if got := bb.Bytes(); !bytes.Equal(got, want) {
					b.Errorf("GOT: %v; WANT: %v", string(got), string(want))
				}

				bb.Reset()
			}
		})

		b.Run("with string formatting", func(b *testing.B) {
			want := []byte("{\"level\":\"info\",\"happy\":true,\"sad\":false,\"usage\":42.3,\"name\":\"First Last\",\"age\":42,\"eye-color\":\"brown\",\"message\":\"with string formatting\"}\n")

			f.SetLevel(Debug)

			for i := 0; i < b.N; i++ {
				f.Info().
					Bool("happy", true).
					Bool("sad", false).
					Float("usage", 42.3).
					Format("name", "%s %s", "First", "Last").
					Int("age", 42).
					String("eye-color", "brown").
					Msg("with string formatting")

				if got := bb.Bytes(); !bytes.Equal(got, want) {
					b.Errorf("GOT: %v; WANT: %v", string(got), string(want))
				}

				bb.Reset()
			}
		})
	})
}
