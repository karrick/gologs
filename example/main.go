package main

import (
	"log"
	"os"

	"github.com/karrick/gologs"
)

func main() {
	logs := gologs.NewDefaultLogger(os.Stderr)
	logs.Info("testing: %v, %v, %v", 1, "two", 1+2)
	log.Printf("[INFO] testing: %v, %v, %v", 1, "two", 1+2)
}
