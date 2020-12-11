package main

import (
	"fmt"
	"os"

	"github.com/lingio/go-common/codegen/gen"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <read | write> <path-to-spec> <path-to-target-dir>")
		os.Exit(1)
	}
	mode := os.Args[1]
	spec := os.Args[2]
	target := os.Args[3]
	allFuncs := gen.ReadSpec(spec)

	// Split into read and write functions
	readFuncs := make([]gen.Func, 0)
	writeFuncs := make([]gen.Func, 0)
	for _, f := range allFuncs {
		if f.HttpMethod == "GET" {
			readFuncs = append(readFuncs, f)
		} else {
			writeFuncs = append(writeFuncs, f)
		}
	}
	gen.GenerateAll(readFuncs, target, false)
	if mode == "write" {
		gen.GenerateAll(writeFuncs, target, true)
	}
}
