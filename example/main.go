package main

import (
	"os"

	"github.com/karrick/gologs"
)

func main() {
	base := gologs.New(os.Stderr, "[LOG] ")
	base.User("%v %v %v", 3.14, "hello", struct{}{})
}
