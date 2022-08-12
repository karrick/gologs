package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/karrick/gologs"
	"github.com/rs/zerolog"
)

func main() {
	optHttp := flag.Int("http", 8080, "specify http port")
	optIterations := flag.Int("iterations", 1000, "number of iterations")
	flag.Parse()

	clearSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", *optHttp),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}

	go func() {
		err := clearSrv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "cannot serve: %s\n", err)
			os.Exit(1)
		}
	}()

	go func() {
		benchmarkFlamegraphGologs(*optIterations)
	}()
	go func() {
		benchmarkFlamegraphZerolog(*optIterations)
	}()

	signals := make(chan os.Signal, 2) // buffered channel
	signal.Notify(signals, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGTERM)

	for {
		select {
		case sig := <-signals:
			fmt.Fprintf(os.Stderr, "received signal: %s; shutting down...\n", sig)
			clearSrv.Shutdown(context.Background())
			fmt.Fprintf(os.Stderr, "shutdown complete; exiting...\n")
			os.Exit(0)
		}
	}
}

const fakeMessage = "Test logging, but use a somewhat realistic message length."

func benchmarkFlamegraphGologs(flameIterations int) {
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

func benchmarkFlamegraphZerolog(flameIterations int) {
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
