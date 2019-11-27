package gologs

import (
	"bytes"
	"testing"
)

func TestLogger(t *testing.T) {
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
	t.Run("prefix", func(t *testing.T) {
		check := func(t *testing.T, callback func(*Logger), want string) {
			t.Helper()
			bb := new(bytes.Buffer)
			log, err := New(bb, "[A] {message}")
			if err != nil {
				t.Fatal(err)
			}
			callback(log)
			if got := string(bb.Bytes()); got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		}

		check(t, func(l *Logger) { NewBranchWithPrefix(l, "[B] ").User("%v", 3.14) }, "[A] [B] 3.14\n")
		check(t, func(l *Logger) { NewBranchWithPrefix(NewBranchWithPrefix(l, "[B] "), "[C] ").User("%v", 3.14) }, "[A] [B] [C] 3.14\n")
	})

	t.Run("filter", func(t *testing.T) {
		check := func(t *testing.T, callback func(*Logger), want string) {
			t.Helper()
			bb := new(bytes.Buffer)
			log, err := New(bb, "[BASE] {message}")
			if err != nil {
				t.Fatal(err)
			}
			callback(log)
			if got := string(bb.Bytes()); got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		}

		t.Run("should-ignore", func(t *testing.T) {
			t.Run("default-logger-dev-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.Dev("%v %v %v", 3.14, "hello", struct{}{}) }, "")
			})
			t.Run("admin-logger-dev-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.SetAdmin().Dev("%v %v %v", 3.14, "hello", struct{}{}) }, "")
			})
			t.Run("user-logger-dev-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.SetUser().Dev("%v %v %v", 3.14, "hello", struct{}{}) }, "")
			})
			t.Run("user-logger-admin-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.SetUser().Admin("%v %v %v", 3.14, "hello", struct{}{}) }, "")
			})
		})

		t.Run("should-convey", func(t *testing.T) {
			t.Run("admin-logger-admin-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.SetAdmin().Admin("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
			})
			t.Run("admin-logger-user-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.SetAdmin().User("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
			})
			t.Run("user-logger-user-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.SetUser().User("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
			})
			t.Run("dev-logger-admin-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.SetDev().Admin("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
			})
			t.Run("dev-logger-dev-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.SetDev().Dev("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
			})
			t.Run("dev-logger-user-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.SetDev().User("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
			})
		})
	})

	t.Run("tracer", func(t *testing.T) {
		t.Run("prefixes emitted in proper order", func(t *testing.T) {
			bb := new(bytes.Buffer)

			log, err := New(bb, "[BASE] {message}")
			if err != nil {
				t.Fatal(err)
			}

			tracer := NewTracer(NewTracer(log, "[TRACER1] "), "[TRACER2] ")

			tracer.Admin("%v %v %v", 3.14, "hello", struct{}{})
			if got, want := string(bb.Bytes()), "[BASE] [TRACER1] [TRACER2] 3.14 hello {}\n"; got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		})

		t.Run("tracers emitted regardless of intermediate branchs", func(t *testing.T) {
			bb := new(bytes.Buffer)

			log, err := New(bb, "[BASE] {message}")
			if err != nil {
				t.Fatal(err)
			}

			tracer := NewTracer(log.SetUser(), "[TRACER] ")

			tracer.Admin("%v %v %v", 3.14, "hello", struct{}{})
			if got, want := string(bb.Bytes()), "[BASE] [TRACER] 3.14 hello {}\n"; got != want {
				t.Errorf("GOT: %q; WANT: %q", got, want)
			}
		})
	})
}
