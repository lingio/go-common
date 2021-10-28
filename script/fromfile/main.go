package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"path"

	"github.com/lingio/go-common"
)

// Object is a data structure for transforming and processing object store data.
type Object struct {
	common.ObjectInfo
	Data []byte
}

//
// Usage:
//
// ls -1 people/*.json | go run ./script/fromfile \
//	go run ./script/tofile --root=dir
func main() {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	// log to stderr to keep stdout for json data
	log.Default().SetOutput(os.Stderr)

	for scanner.Scan() {
		// read one file path per line
		filename := scanner.Text()
		data, err := os.ReadFile(filename)
		trap(err)
		// construct shared object structure
		obj := Object{
			ObjectInfo: common.ObjectInfo{
				Key: path.Base(filename),
			},
			Data: data,
		}
		// dump to stdout
		trap(encoder.Encode(obj))
	}
}

func trap(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
