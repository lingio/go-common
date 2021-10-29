package main

import (
	"encoding/json"
	"flag"
	"io"
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
// pipe Objects into | go run ./script/tofile --root=./
func main() {
	rootPath := flag.String("root", "./", "path to root folder where to output files")
	flag.Parse()

	trap(os.MkdirAll(*rootPath, 0755))

	decoder := json.NewDecoder(os.Stdin)
	for {
		var obj Object
		if err := decoder.Decode(&obj); err != nil && err != io.EOF {
			trap(err)
		} else if err == io.EOF {
			break
		}
		if err := os.WriteFile(path.Join(*rootPath, obj.Key), obj.Data, os.ModePerm); err != nil {
			trap(err)
		}
	}
}

func trap(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
