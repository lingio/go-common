package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/lingio/go-common"
)

// Object is a data structure for transforming and processing object store data.
type Object struct {
	common.ObjectInfo
	Data []byte
}

type dummyStore struct {
	decoder *json.Decoder
	encoder *json.Encoder
}

var noV2 bool
var v2header = [...]byte{'v', 2, '/'}

//
// Usage:
//
// ENCRYPTION_KEY=256bit-key go run ./script/encrypt [--decrypt]
func main() {
	log.Default().SetOutput(os.Stderr)
	log.Default().SetPrefix("[encrypt]")

	decrypt := flag.Bool("decrypt", false, "decrypt stdin (instead of encrypt)")
	flag.BoolVar(&noV2, "nov2", false, "if set, panics when trying to decrypt v2 crypto data")
	serviceKey := os.Getenv("ENCRYPTION_KEY")
	flag.Parse()

	if serviceKey == "" {
		trap(errors.New("missing ENCRYPTION_KEY environment variable"))
	}

	ds := &dummyStore{
		decoder: json.NewDecoder(os.Stdin),
		encoder: json.NewEncoder(os.Stdout),
	}

	store, err := common.NewEncryptedStore(ds, serviceKey)
	trap(err)

	if *decrypt {
		for {
			data, info, err := store.GetObject("dummyfilename16b") // read from stdin and decrypt
			if err != nil && err.Unwrap() == io.EOF {
				break
			} else if err != nil {
				trap(fmt.Errorf("read: %w", err))
			}
			_, lerr := ds.PutObject(context.TODO(), info.Key, data) // write plain text to stdout
			trap(lerr)
		}
	} else {
		for {
			data, info, err := ds.GetObject("") // read directly from stdin
			if err != nil && err.Unwrap() == io.EOF {
				break
			} else if err != nil {
				trap(fmt.Errorf("read: %w", err))
			}
			_, lerr := store.PutObject(context.TODO(), info.Key, data) // write encrypted to stdout
			trap(lerr)
		}
	}
}

func trap(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

// GetObject is supposed to be called when we're trying to decrypt an encrypted stdin.
func (ds dummyStore) GetObject(filename string) ([]byte, common.ObjectInfo, *common.Error) {
	var obj Object
	if err := ds.decoder.Decode(&obj); err != nil {
		return nil, common.ObjectInfo{}, common.NewErrorE(http.StatusInternalServerError, err)
	}
	if noV2 {
		if bytes.HasPrefix([]byte(obj.Key), v2header[:]) {
			panic(fmt.Errorf("object key %q starts with v2 magic header: %s", obj.Key, v2header))
		}
		if bytes.HasPrefix(obj.Data, v2header[:]) {
			panic(fmt.Errorf("object data %v starts with v2 magic header: %s", obj.Data, v2header))
		}
	}
	return obj.Data, obj.ObjectInfo, nil
}

// PutObject is supposed to be called when we're trying to encrypt a plain-text stdin.
func (ds dummyStore) PutObject(ctx context.Context, file string, data []byte) (common.ObjectInfo, *common.Error) {
	if err := ds.encoder.Encode(Object{
		Data: data,
		ObjectInfo: common.ObjectInfo{
			Key: file,
		},
	}); err != nil {
		return common.ObjectInfo{}, common.NewErrorE(http.StatusInternalServerError, err)
	}
	return common.ObjectInfo{}, nil
}

func (ds dummyStore) DeleteObject(ctx context.Context, file string) *common.Error {
	return nil
}
func (ds dummyStore) ListObjects(ctx context.Context) <-chan common.ObjectInfo {
	return nil
}
func (ds dummyStore) StoreName() string {
	return "dummy store"
}
