package gologs

import (
	"bytes"
	"fmt"
	"testing"
)

func Example() {
	bb := new(bytes.Buffer)
	base := New(bb, "[BASE] ")
	base.Dev("%v %v %v", 3.14, "hello", struct{}{})
	fmt.Printf("%s", bb.Bytes())
	// Output:
	// [BASE] 3.14 hello {}
}

func TestBase(t *testing.T) {
	// TODO: test for compiles line
	bb := new(bytes.Buffer)

	logs := New(bb, "[BASE] {message}")
	logs.Dev("%v %v %v", 3.14, "hello", struct{}{})

	if got, want := string(bb.Bytes()), "[BASE] 3.14 hello {}\n"; got != want {
		t.Errorf("GOT: %q; WANT: %q", got, want)
	}
}

func TestBaseAppendsNewline(t *testing.T) {
	bb := new(bytes.Buffer)

	logs := New(bb, "[BASE] {message}")
	logs.Dev("%v %v %v", 3.14, "hello", struct{}{})

	if got, want := string(bb.Bytes()), "[BASE] 3.14 hello {}\n"; got != want {
		t.Errorf("GOT: %q; WANT: %q", got, want)
	}
}

func TestPrefix(t *testing.T) {
	t.Run("prefixes emitted in proper order", func(t *testing.T) {
		bb := new(bytes.Buffer)

		logs := NewPrefix(NewPrefix(New(bb, "[A] "), "[B] "), "[C] ")

		logs.Admin("%v %v %v", 3.14, "hello", struct{}{})
		if got, want := string(bb.Bytes()), "[A] [B] [C] 3.14 hello {}\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
}

func TestFilter(t *testing.T) {
	check := func(t *testing.T, callback func(*Filter), want string) {
		bb := new(bytes.Buffer)
		a := New(bb, "[BASE] {message}")
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
