package gologstest

import "testing"

// TestWriterForwards checks the io.Writer contract: it returns len(p), nil
// and forwards to t.Log without panicking. (testing.TB cannot be faked — it
// has an unexported method — so this uses the real *testing.T; the forwarded
// line only shows under -v or on failure.)
func TestWriterForwards(t *testing.T) {
	w := writer{t}
	n, err := w.Write([]byte("hello\n"))
	if err != nil {
		t.Fatalf("Write err = %v", err)
	}
	if n != len("hello\n") {
		t.Fatalf("Write n = %d, want %d", n, len("hello\n"))
	}
}

// TestNew smoke-tests that New returns a usable logger and that a log call is
// forwarded (captured by go test; visible under -v).
func TestNew(t *testing.T) {
	log := New(t)
	if log == nil {
		t.Fatal("New returned nil")
	}
	log.Info().Msg("gologstest smoke")
}
