package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"strings"

	"github.com/lingio/go-common"
)

// Object is a data structure for transforming and processing object store data.
type Object struct {
	common.ObjectInfo
	Data []byte
}

// Objify converts a normal json object to the shared object storage format:
// cat some.jsonl | go run ./scripts/objify -id=userId -filenameformat="data-{id}.json"
func main() {
	log.Default().SetOutput(os.Stderr)

	idkey := flag.String("id", "id", "id field name")
	fnformat := flag.String("filenameformat", "{id}.json", "filename format, replaces {id} with extracted id")
	flag.Parse()

	stdin := json.NewDecoder(os.Stdin)
	stdout := json.NewEncoder(os.Stdout)

	for {
		in := make(map[string]interface{})
		if err := stdin.Decode(&in); err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("objify: decode input: %s\n", err)
		}

		var obj Object

		if id, ok := in[*idkey]; !ok {
			log.Fatalf("objify: no id found in object %v\n", in)
		} else if id, ok := id.(string); ok {
			obj.Key = strings.ReplaceAll(*fnformat, "{id}", id)
		}

		var err error
		obj.Data, err = json.Marshal(in)
		if err != nil {
			log.Fatalf("objify: marshalling output: %s\n", err)
		}

		stdout.Encode(obj)
	}
}
