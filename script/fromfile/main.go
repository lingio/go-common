package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path"
	"strings"

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
// find ../people-files -maxdepth 1 -not -type d | go run ./script/fromfile \
//	go run ./script/tofile --root=dir
func main() {
	renameFmt := flag.String("rename", "{KEY}{EXT}", "set object key to filename and parsed extension")
	flag.Parse()

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
				Key: rename(*renameFmt, path.Base(filename)),
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

func rename(format, key string) string {
	if format == "{KEY}{EXT}" {
		return key
	}
	ext := path.Ext(key)
	if ext != "" {
		key = key[0 : len(key)-len(ext)]
	}
	filename := format
	filename = strings.ReplaceAll(filename, "{KEY}", key)
	filename = strings.ReplaceAll(filename, "{EXT}", ext)
	return filename
}
