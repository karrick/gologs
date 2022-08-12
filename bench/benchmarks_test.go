package main

import (
	"io/ioutil"
	_ "net/http/pprof"
	"testing"
	"time"

	"github.com/karrick/gologs"
	"github.com/rs/zerolog"
)

func BenchmarkLogEmptyGologs(b *testing.B) {
	logger := gologs.New(ioutil.Discard)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		logger = logger.With().Logger()
		for pb.Next() {
			logger.Log().Msg("")
		}
	})
}

func BenchmarkLogEmptyZerolog(b *testing.B) {
	logger := zerolog.New(ioutil.Discard)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Log().Msg("")
		}
	})
}

func BenchmarkLogDisabledGologs(b *testing.B) {
	logger := gologs.New(ioutil.Discard).SetError()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		logger = logger.With().Logger()
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkLogDisabledZerolog(b *testing.B) {
	logger := zerolog.New(ioutil.Discard).Level(zerolog.Disabled)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkLogInfoGologs(b *testing.B) {
	logger := gologs.New(ioutil.Discard).SetInfo()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		logger = logger.With().Logger()
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkLogInfoZerolog(b *testing.B) {
	logger := zerolog.New(ioutil.Discard)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkContextFieldsGologs(b *testing.B) {
	logger := gologs.New(ioutil.Discard).SetInfo().
		SetTimeFormatter(gologs.TimeUnix).
		With().
		String("string", "four!").
		Int("int", 123).
		Float("float", -2.203230293249593).
		Logger()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		logger = logger.With().Logger()
		for pb.Next() {
			logger.Info().Msg(fakeMessage)
		}
	})
}

func BenchmarkContextFieldsZerolog(b *testing.B) {
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
}

func BenchmarkLogFieldsGologs(b *testing.B) {
	logger := gologs.New(ioutil.Discard).
		SetTimeFormatter(gologs.TimeUnix).
		SetInfo()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		logger = logger.With().Logger()
		for pb.Next() {
			logger.Info().
				String("string", "four!").
				Int("int", 123).
				Float("float", -2.203230293249593).
				Msg(fakeMessage)
		}
	})
}

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

var flameIterations = 10000000

func BenchmarkFlamegraphGologs(b *testing.B) {
	logger := gologs.New(ioutil.Discard).
		SetTimeFormatter(gologs.TimeUnix).
		SetInfo()

	for i := 0; i < flameIterations; i++ {
		logger.Info().
			String("string", "four!").
			Int("int", 123).
			Float("float", -2.203230293249593).
			Msg(fakeMessage)
	}
}

func BenchmarkFlamegraphZerolog(b *testing.B) {
	logger := zerolog.New(ioutil.Discard)

	for i := 0; i < flameIterations; i++ {
		logger.Info().
			Str("string", "four!").
			Time("time", time.Now()).
			Int("int", 123).
			Float32("float", -2.203230293249593).
			Msg(fakeMessage)
	}
}
