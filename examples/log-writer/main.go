package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/karrick/gologs"
)

func main() {
	log := gologs.New(os.Stdout).SetTimeFormatter(gologs.TimeUnix)
	lw := log.NewWriter()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		_, err := lw.Write(scanner.Bytes())
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
