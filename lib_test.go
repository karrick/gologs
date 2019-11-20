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
func MustCompile(w io.Writer, template string) *Base {
	base, err := New(w, template)
	if err != nil {
		panic(err)
	}
	return base
}

func Example() {
	bb := new(bytes.Buffer)
	base, err := New(bb, "[BASE] {message}")
	if err != nil {
		os.Exit(1)
	}

	base.Dev("%v %v %v", 3.14, "hello", struct{}{})
	fmt.Printf("%s", bb.Bytes())
	// Output:
	// [BASE] 3.14 hello {}
}

func TestBase(t *testing.T) {
	// TODO: test for compiles line
	bb := new(bytes.Buffer)

	logs, err := New(bb, "[BASE] {message}")
	if err != nil {
		os.Exit(1)
	}

	logs.Dev("%v %v %v", 3.14, "hello", struct{}{})

	if got, want := string(bb.Bytes()), "[BASE] 3.14 hello {}\n"; got != want {
		t.Errorf("GOT: %q; WANT: %q", got, want)
	}
}

func TestBaseAppendsNewline(t *testing.T) {
	bb := new(bytes.Buffer)

	logs, err := New(bb, "[BASE] {message}")
	if err != nil {
		os.Exit(1)
	}
	logs.Dev("%v %v %v", 3.14, "hello", struct{}{})

	if got, want := string(bb.Bytes()), "[BASE] 3.14 hello {}\n"; got != want {
		t.Errorf("GOT: %q; WANT: %q", got, want)
	}
}

func TestFilter(t *testing.T) {
	check := func(t *testing.T, callback func(*Filter), want string) {
		bb := new(bytes.Buffer)
		a, err := New(bb, "[BASE] {message}")
		if err != nil {
			t.Fatal(err)
		}
		b := NewFilter(a)
		callback(b)
		if got := string(bb.Bytes()); got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	}

	t.Run("should-ignore", func(t *testing.T) {
		t.Run("admin-logger-dev-event", func(t *testing.T) {
			check(t, func(f *Filter) { f.SetAdmin().Dev("%v %v %v", 3.14, "hello", struct{}{}) }, "")
		})
		t.Run("user-logger-dev-event", func(t *testing.T) {
			check(t, func(f *Filter) { f.SetUser().Dev("%v %v %v", 3.14, "hello", struct{}{}) }, "")
		})
		t.Run("user-logger-admin-event", func(t *testing.T) {
			check(t, func(f *Filter) { f.SetUser().Admin("%v %v %v", 3.14, "hello", struct{}{}) }, "")
		})
	})

	t.Run("should-convey", func(t *testing.T) {
		t.Run("admin-logger-admin-event", func(t *testing.T) {
			check(t, func(f *Filter) { f.SetAdmin().Admin("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
		})
		t.Run("admin-logger-user-event", func(t *testing.T) {
			check(t, func(f *Filter) { f.SetAdmin().User("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
		})
		t.Run("user-logger-user-event", func(t *testing.T) {
			check(t, func(f *Filter) { f.SetUser().User("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
		})
		t.Run("dev-logger-admin-event", func(t *testing.T) {
			check(t, func(f *Filter) { f.SetDev().Admin("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
		})
		t.Run("dev-logger-dev-event", func(t *testing.T) {
			check(t, func(f *Filter) { f.SetDev().Dev("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
		})
		t.Run("dev-logger-user-event", func(t *testing.T) {
			check(t, func(f *Filter) { f.SetDev().User("%v %v %v", 3.14, "hello", struct{}{}) }, "[BASE] 3.14 hello {}\n")
		})
	})
}

func TestTracer(t *testing.T) {
	t.Run("prefixes emitted in proper order", func(t *testing.T) {
		bb := new(bytes.Buffer)

		base, err := New(bb, "[BASE] {message}")
		if err != nil {
			t.Fatal(err)
		}

		logs := NewTracer(NewTracer(base, "[TRACER1] "), "[TRACER2] ")

		logs.Admin("%v %v %v", 3.14, "hello", struct{}{})
		if got, want := string(bb.Bytes()), "[BASE] [TRACER1] [TRACER2] 3.14 hello {}\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})

	t.Run("tracers emitted regardless of intermediate filters", func(t *testing.T) {
		bb := new(bytes.Buffer)

		base, err := New(bb, "[BASE] {message}")
		if err != nil {
			t.Fatal(err)
		}

		logs := NewTracer(NewFilter(base).SetUser(), "[TRACER] ")

		logs.Admin("%v %v %v", 3.14, "hello", struct{}{})
		if got, want := string(bb.Bytes()), "[BASE] [TRACER] 3.14 hello {}\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
}
