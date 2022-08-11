package bench

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/karrick/gologs"
	"github.com/rs/zerolog"
)

const fakeMessage = "Test logging, but use a somewhat realistic message length."

func BenchmarkLogEmpty(b *testing.B) {
	b.Run("gologs", func(b *testing.B) {
		logger := gologs.New(ioutil.Discard)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Log().Msg("")
			}
		})
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(ioutil.Discard)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Log().Msg("")
			}
		})
	})
}

func BenchmarkLogDisabled(b *testing.B) {
	b.Run("gologs", func(b *testing.B) {
		logger := gologs.New(ioutil.Discard).SetError()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info().Msg(fakeMessage)
			}
		})
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(ioutil.Discard).Level(zerolog.Disabled)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info().Msg(fakeMessage)
			}
		})
	})
}

func BenchmarkLogInfo(b *testing.B) {
	b.Run("gologs", func(b *testing.B) {
		logger := gologs.New(ioutil.Discard).SetInfo()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info().Msg(fakeMessage)
			}
		})
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(ioutil.Discard)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info().Msg(fakeMessage)
			}
		})
	})
}

func BenchmarkContextFields(b *testing.B) {
	b.Run("gologs", func(b *testing.B) {
		logger := gologs.New(ioutil.Discard).SetInfo().
			SetTimeFormatter(gologs.TimeUnix).
			With().
			String("string", "four!").
			Int("int", 123).
			Float("float", -2.203230293249593).
			Logger()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info().Msg(fakeMessage)
			}
		})
	})

	b.Run("zerolog", func(b *testing.B) {
		logger := zerolog.New(ioutil.Discard).
			With().
			Str("string", "four!").
			Time("time", time.Time{}).
			Int("int", 123).
			Float32("float", -2.203230293249593).
			Logger()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info().Msg(fakeMessage)
			}
		})
	})
}

func BenchmarkLogFields(b *testing.B) {
	b.Run("gologs", func(b *testing.B) {
		logger := gologs.New(ioutil.Discard).SetInfo().SetTimeFormatter(gologs.TimeUnix)
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
	})

	b.Run("zerolog", func(b *testing.B) {
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
	})
}
