package main

import (
	"encoding/json"
	"flag"
	"io"
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
// pipe Objects into | go run ./script/tofile --root=./
func main() {
	rootPath := flag.String("root", "./", "path to root folder where to output files")
	renameFmt := flag.String("rename", "{KEY}{EXT}", "rename filename using object key and parsed extension")
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
		filename := rename(*renameFmt, obj.Key)
		if err := os.WriteFile(path.Join(*rootPath, filename), obj.Data, os.ModePerm); err != nil {
			trap(err)
		}
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
