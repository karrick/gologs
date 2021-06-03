package gologs

import (
	"bytes"
	"io"
	"testing"
)

func TestSingleNewLine(t *testing.T) {
	t.Run("no newlines", func(t *testing.T) {
		ensureBuffer(t, singleNewline([]byte("this has no newlines")), []byte("this has no newlines\n"))
	})
	t.Run("one newline", func(t *testing.T) {
		ensureBuffer(t, singleNewline([]byte("this has one newline\n")), []byte("this has one newline\n"))
	})
	t.Run("two newlines", func(t *testing.T) {
		ensureBuffer(t, singleNewline([]byte("this has two newlines\n\n")), []byte("this has two newlines\n"))
	})
	t.Run("hidden newline", func(t *testing.T) {
		ensureBuffer(t, singleNewline([]byte("this\nhas\nhidden\nnewlines\n\n")), []byte("this\nhas\nhidden\nnewlines\n"))
	})
	t.Run("all newlines", func(t *testing.T) {
		ensureBuffer(t, singleNewline([]byte("\n\n\n\n")), []byte("\n"))
	})
}

func TestLogger(t *testing.T) {
	t.Run("single newline", func(t *testing.T) {
		t.Run("without newline in log format", func(t *testing.T) {
			t.Run("without newline in event format", func(t *testing.T) {
				bb := new(bytes.Buffer)
				log, err := New(bb, "{message}")
				ensureError(t, err)

				err = log.Error("test")
				ensureError(t, err)

				if got, want := string(bb.Bytes()), "test\n"; got != want {
					t.Errorf("GOT: %q; WANT: %q", got, want)
				}
			})
			t.Run("with newline in event format", func(t *testing.T) {
				bb := new(bytes.Buffer)
				log, err := New(bb, "{message}")
				ensureError(t, err)

				err = log.Error("test\n")
				ensureError(t, err)

				if got, want := string(bb.Bytes()), "test\n"; got != want {
					t.Errorf("GOT: %q; WANT: %q", got, want)
				}
			})
		})
		t.Run("with newline in log format", func(t *testing.T) {
			t.Run("without newline in event format", func(t *testing.T) {
				bb := new(bytes.Buffer)
				log, err := New(bb, "{message}\n")
				ensureError(t, err)

				err = log.Error("test")
				ensureError(t, err)

				if got, want := string(bb.Bytes()), "test\n"; got != want {
					t.Errorf("GOT: %q; WANT: %q", got, want)
				}
			})
			t.Run("with newline in event format", func(t *testing.T) {
				bb := new(bytes.Buffer)
				log, err := New(bb, "{message}\n")
				ensureError(t, err)

				err = log.Error("test\n")
				ensureError(t, err)

				if got, want := string(bb.Bytes()), "test\n"; got != want {
					t.Errorf("GOT: %q; WANT: %q", got, want)
				}
			})
		})
	})

	t.Run("is time required", func(t *testing.T) {
		t.Run("is not required", func(t *testing.T) {
			_, isTimeRequired, err := compileFormat("{message}")
			ensureError(t, err)
			if got, want := isTimeRequired, false; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})
		t.Run("is required", func(t *testing.T) {
			t.Run("epoch", func(t *testing.T) {
				_, isTimeRequired, err := compileFormat("{epoch} {message}")
				ensureError(t, err)
				if got, want := isTimeRequired, true; got != want {
					t.Errorf("GOT: %v; WANT: %v", got, want)
				}
			})
			t.Run("iso8601", func(t *testing.T) {
				_, isTimeRequired, err := compileFormat("{iso8601} {message}")
				ensureError(t, err)
				if got, want := isTimeRequired, true; got != want {
					t.Errorf("GOT: %v; WANT: %v", got, want)
				}
			})
			t.Run("localtime=2006/01/02 15:04:05", func(t *testing.T) {
				_, isTimeRequired, err := compileFormat("{localtime=2006/01/02 15:04:05} {message}")
				ensureError(t, err)
				if got, want := isTimeRequired, true; got != want {
					t.Errorf("GOT: %v; WANT: %v", got, want)
				}
			})
			t.Run("utctime=2006/01/02 15:04:05", func(t *testing.T) {
				_, isTimeRequired, err := compileFormat("{utctime=2006/01/02 15:04:05} {message}")
				ensureError(t, err)
				if got, want := isTimeRequired, true; got != want {
					t.Errorf("GOT: %v; WANT: %v", got, want)
				}
			})
			t.Run("timestamp", func(t *testing.T) {
				_, isTimeRequired, err := compileFormat("{message} {timestamp}")
				ensureError(t, err)
				if got, want := isTimeRequired, true; got != want {
					t.Errorf("GOT: %v; WANT: %v", got, want)
				}
			})
		})
	})

	t.Run("filter", func(t *testing.T) {
		const message = "some message"

		check := func(t *testing.T, want string, callback func(*Logger)) {
			t.Helper()
			bb := new(bytes.Buffer)
			log, err := New(bb, "{message}")
			ensureError(t, err)
			callback(log)
			if got := string(bb.Bytes()); got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		}

		// default logger mode is warning
		check(t, "", func(f *Logger) { f.Debug(message) })
		check(t, "", func(f *Logger) { f.Verbose(message) })
		check(t, "", func(f *Logger) { f.Info(message) })
		check(t, message+"\n", func(f *Logger) { f.Warning(message) })
		check(t, message+"\n", func(f *Logger) { f.Error(message) })

		check(t, message+"\n", func(f *Logger) { f.SetDebug().Debug(message) })
		check(t, message+"\n", func(f *Logger) { f.SetDebug().Verbose(message) })
		check(t, message+"\n", func(f *Logger) { f.SetDebug().Info(message) })
		check(t, message+"\n", func(f *Logger) { f.SetDebug().Warning(message) })
		check(t, message+"\n", func(f *Logger) { f.SetDebug().Error(message) })

		check(t, "", func(f *Logger) { f.SetVerbose().Debug(message) })
		check(t, message+"\n", func(f *Logger) { f.SetVerbose().Verbose(message) })
		check(t, message+"\n", func(f *Logger) { f.SetVerbose().Info(message) })
		check(t, message+"\n", func(f *Logger) { f.SetVerbose().Warning(message) })
		check(t, message+"\n", func(f *Logger) { f.SetVerbose().Error(message) })

		check(t, "", func(f *Logger) { f.SetInfo().Debug(message) })
		check(t, "", func(f *Logger) { f.SetInfo().Verbose(message) })
		check(t, message+"\n", func(f *Logger) { f.SetInfo().Info(message) })
		check(t, message+"\n", func(f *Logger) { f.SetInfo().Warning(message) })
		check(t, message+"\n", func(f *Logger) { f.SetInfo().Error(message) })

		check(t, "", func(f *Logger) { f.SetWarning().Debug(message) })
		check(t, "", func(f *Logger) { f.SetWarning().Verbose(message) })
		check(t, "", func(f *Logger) { f.SetWarning().Info(message) })
		check(t, message+"\n", func(f *Logger) { f.SetWarning().Warning(message) })
		check(t, message+"\n", func(f *Logger) { f.SetWarning().Error(message) })

		check(t, "", func(f *Logger) { f.SetError().Debug(message) })
		check(t, "", func(f *Logger) { f.SetError().Verbose(message) })
		check(t, "", func(f *Logger) { f.SetError().Info(message) })
		check(t, "", func(f *Logger) { f.SetError().Warning(message) })
		check(t, message+"\n", func(f *Logger) { f.SetError().Error(message) })
	})

	t.Run("prefix", func(t *testing.T) {
		check := func(t *testing.T, want string, callback func(*Logger)) {
			t.Helper()
			bb := new(bytes.Buffer)
			log, err := New(bb, "[A] {message}")
			ensureError(t, err)
			callback(log)
			if got := string(bb.Bytes()); got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		}

		check(t, "[A] [B] 3.14\n", func(l *Logger) { l.NewBranchWithPrefix("[B] ").Error("%v", 3.14) })
		check(t, "[A] [B] [C] 3.14\n", func(l *Logger) { l.NewBranchWithPrefix("[B] ").NewBranchWithPrefix("[C] ").Error("%v", 3.14) })
	})

	t.Run("tracer", func(t *testing.T) {
		t.Run("prefixes emitted in proper order", func(t *testing.T) {
			bb := new(bytes.Buffer)

			log, err := New(bb, "[BASE] {message}")
			ensureError(t, err)

			tracer := log.NewTracer("[TRACER1] ").NewTracer("[TRACER2] ")

			tracer.Verbose("%v %v %v", 3.14, "hello", struct{}{})
			if got, want := string(bb.Bytes()), "[BASE] [TRACER1] [TRACER2] 3.14 hello {}\n"; got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		})

		t.Run("tracers emitted regardless of intermediate branchs", func(t *testing.T) {
			bb := new(bytes.Buffer)

			log, err := New(bb, "[BASE] {message}")
			ensureError(t, err)

			tracer := log.SetError().NewTracer("[TRACER] ")

			tracer.Verbose("%v %v %v", 3.14, "hello", struct{}{})
			if got, want := string(bb.Bytes()), "[BASE] [TRACER] 3.14 hello {}\n"; got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		})
	})

	t.Run("does not hold lock when writer panics", func(t *testing.T) {
		bb := new(bytes.Buffer)
		pw := &panicyWriter{w: bb}

		log, err := New(pw, "{message}")
		ensureError(t, err)

		log.SetDebug()

		log.Info("message 1")

		ensurePanic(t, "boom!", func() {
			pw.isTriggered = true
			log.Info("message 2")
		})

		pw.isTriggered = false
		log.Info("message 3")

		expected := "message 1\nmessage 2\nmessage 3\n"
		if got, want := string(bb.Bytes()), expected; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
}

type panicyWriter struct {
	w           io.Writer
	isTriggered bool
}

func (pw *panicyWriter) Write(buf []byte) (int, error) {
	n, err := pw.w.Write(buf)
	if pw.isTriggered {
		panic("boom!")
	}
	return n, err
}
