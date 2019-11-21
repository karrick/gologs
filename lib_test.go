package gologs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

// MustCompile returns a new Base Logger, or will panic if the template is not
// valid.
//
// ??? This function is not yet part of the library, but is a likely candidate
// for future inclusion.
func MustCompile(w io.Writer, template string) *Logger {
	base, err := New(w, template)
	if err != nil {
		panic(err)
	}
	return base
}

func Example() {
	bb := new(bytes.Buffer)
	log, err := New(bb, "[BASE] {message}")
	if err != nil {
		os.Exit(1)
	}

	log.User("%v %v %v", 3.14, "hello", struct{}{})
	fmt.Printf("%s", bb.Bytes())
	// Output:
	// [BASE] 3.14 hello {}
}

func TestLogger(t *testing.T) {
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
			t.Run("default-logger-dev-event", func(t *testing.T) {
				check(t, func(f *Logger) { f.Dev("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
			})
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
