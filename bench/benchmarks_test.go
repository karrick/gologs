package bench

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/karrick/gologs"
	"github.com/rs/zerolog"
)

// go test -bench=. -benchmem
// goos: freebsd
// goarch: amd64
// pkg: github.com/karrick/gologs/bench
// cpu: AMD Ryzen Threadripper 3960X 24-Core Processor
// BenchmarkLogFieldsZerolog-48    70580732  17.92    ns/op  0  B/op  0  allocs/op
// BenchmarkLogFieldsGologs-48   1000000000   0.1212  ns/op  0  B/op  0  allocs/op
// PASS
// ok  	github.com/karrick/gologs/bench	1.430s

const fakeMessage = "Test logging, but use a somewhat realistic message length."

func BenchmarkLogFieldsZerolog(b *testing.B) {
	logger := zerolog.New(ioutil.Discard)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().
				Str("string", "four!").
				Time("time", time.Time{}).
				Int("int", 123).
				Float32("float", -2.203230293249593).
				Msg(fakeMessage)
		}
	})
}

func BenchmarkLogFieldsGologs(b *testing.B) {
	logger := gologs.New(ioutil.Discard).SetTimeFormatter(gologs.TimeUnix)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().
				String("string", "four!").
				Int("int", 123).
				Float("float", -2.203230293249593).
				Msg(fakeMessage)
		}
	})
}
