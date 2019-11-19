package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/karrick/gologs"
)

func main() {
	var ProgramName string
	var err error
	if ProgramName, err = os.Executable(); err != nil {
		ProgramName = os.Args[0]
	}
	ProgramName = filepath.Base(ProgramName)

	base := gologs.New(os.Stderr, fmt.Sprintf("[%s] {message}", ProgramName))
	base.User("%v %v %v", 3.14, "hello", struct{}{})
}
