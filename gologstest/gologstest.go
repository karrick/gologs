// Package gologstest provides a *gologs.Logger for use in tests.
//
// It forwards log output to t.Log, so `go test` captures it per-test and
// prints it only when the test fails (or under -v) — the run-up-to-failure
// context that io.Discard throws away, while keeping passing tests quiet. It
// logs at debug verbosity so every level is captured.
//
// Mirrors the well-worn zaptest.NewLogger(t) pattern. It lives in a
// subpackage on purpose so the core gologs package never imports "testing"
// (which would pull the test framework into every consumer's build graph).
package gologstest

import (
	"bytes"
	"testing"

	"github.com/karrick/gologs"
)

// New returns a logger wired to t.Log at debug verbosity. Construct one per
// test: a gologs logger is cheap, and a fresh writer per test keeps tests
// isolated and parallel-safe (t.Log is per-test and goroutine-safe), unlike
// sharing one logger and swapping its writer.
func New(tb testing.TB) *gologs.Logger {
	return gologs.New(writer{tb}).SetDebug()
}

// writer adapts testing.TB to io.Writer, emitting one t.Log record per log
// line.
type writer struct{ tb testing.TB }

func (w writer) Write(p []byte) (int, error) {
	w.tb.Logf("%s", bytes.TrimRight(p, "\n"))
	return len(p), nil
}
