package main

import (
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

//
// Usage:
//
// ENCRYPTION_KEY=256bit-key go run ./script/encrypt [--decrypt]
func main() {
	log.Default().SetOutput(os.Stderr)
	log.Default().SetPrefix("[encrypt]")

	decrypt := flag.Bool("decrypt", false, "decrypt stdin (instead of encrypt)")
	cryptostore := flag.String("crypto", "", "crypto store to pass through: v1 (insecure) or v2 (secure)")
	serviceKey := os.Getenv("ENCRYPTION_KEY")
	flag.Parse()

	if serviceKey == "" {
		trap(errors.New("missing ENCRYPTION_KEY environment variable"))
	}

	ds := &dummyStore{
		decoder: json.NewDecoder(os.Stdin),
		encoder: json.NewEncoder(os.Stdout),
	}

	var err error
	var store common.LingioStore
	if *cryptostore == "v1" {
		store, err = common.NewInsecureEncryptedStore(ds, serviceKey)
	} else if *cryptostore == "v2" {
		store, err = common.NewEncryptedStore(ds, serviceKey)
	} else {
		trap(fmt.Errorf("unknown crypto store: %q - use either v1 (insecure) or v2 (secure)", *cryptostore))
	}
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
