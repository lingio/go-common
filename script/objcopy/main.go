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
	"path"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/lingio/go-common"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Object is a data structure for transforming and processing object store data.
type Object struct {
	common.ObjectInfo
	Data []byte
}

type config struct {
	Minio minioConfig
}
type minioConfig struct {
	Host, AccessKeyID string
	SSL               bool
}

//
// Usage:
//
// MINIO_SECRET=yaya ./script/objcopy --from=<config> | \
//	ENCRYPTION_KEY=256bit-key ./script/encrypt | \
// 	MINIO_SECRET=yaya ./script/objcopy --to=<config>
//
func main() {
	srcEnv := flag.String("from", "", "json config file with minio source to read from")
	dstEnv := flag.String("to", "", "json config file with minio target to write to")
	bucket := flag.String("bucket", "", "bucket to read from or write to")
	renameFmt := flag.String("rename", "{KEY}{EXT}", "rename object using key and parsed extension")
	nobjsPerSec := flag.Int("obj-per-sec", 1000, "rate limit object writes per second")
	minioSecret := os.Getenv("MINIO_SECRET")
	flag.Parse()

	// Stdout is safe for potential consumers.
	log.Default().SetOutput(os.Stderr)
	log.Default().SetPrefix("[objcopy] ")

	if minioSecret == "" {
		trap(errors.New("missing MINIO_SECRET environment variable"))
	}
	if len(*srcEnv) == 0 && len(*dstEnv) == 0 {
		trap(errors.New("either --from or --to must be specified"))
	}

	var env string
	if len(*srcEnv) > 0 {
		env = *srcEnv
	} else if len(*dstEnv) > 0 {
		env = *dstEnv
	} else {
		trap(errors.New("both --from and --to are empty"))
	}

	configData, err := os.ReadFile(env)
	trap(err)
	var config config
	trap(json.Unmarshal(configData, &config))

	minioClient, err := minio.New(config.Minio.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(config.Minio.AccessKeyID, minioSecret, ""),
		Secure: config.Minio.SSL,
	})
	trap(err)

	if bucketExists, err := minioClient.BucketExists(context.TODO(), *bucket); err != nil {
		trap(err)
	} else if !bucketExists {
		trap(fmt.Errorf("bucket '%s': does not exist", *bucket))
	}

	store, err := common.NewObjectStore(minioClient, *bucket, common.ObjectStoreConfig{})
	trap(err)

	if len(*srcEnv) > 0 {
		log.Println("reading ...")
		// Read from store and write json-encoding to stdout
		var n int
		encoder := json.NewEncoder(os.Stdout)
		for obj := range readAllFromStore(store) {
			encoder.Encode(obj)
			n++
		}
		log.Println("done:", n, "objects read")
	} else {
		// Read json-encoded data from stdin and write to store
		decoder := json.NewDecoder(os.Stdin)
		objchan := make(chan Object, 10)
		go func() {
			defer close(objchan)
			for {
				var obj Object
				if err := decoder.Decode(&obj); err != nil && err != io.EOF {
					log.Println("decoder err:", err)
					break
				} else if err == io.EOF {
					break
				}
				obj.Key = rename(*renameFmt, obj.Key)
				objchan <- obj
			}
		}()
		log.Printf("writing %d objects/sec\n", *nobjsPerSec)
		// wait on store instead of decoding stdin
		n := writeIntoStore(store, *nobjsPerSec, objchan)
		log.Println("done:", n, "objects written")
	}
}

func readAllFromStore(store *common.ObjectStore) <-chan Object {
	const workers = 10
	listing := store.ListObjects(context.Background())
	objchan := make(chan Object, workers*2)
	errchan := make([]chan error, workers)
	for i := 0; i < workers; i++ {
		errchan[i] = make(chan error, 1)
		go func(workerId int) {
			defer close(errchan[workerId])
			for req := range listing {
				data, info, err := store.GetObject(req.Key)
				if err != nil {
					errchan[workerId] <- fmt.Errorf("read: %w", err)
					return
				}

				objchan <- Object{
					ObjectInfo: info,
					Data:       data,
				}
			}
		}(i)
	}

	// Return first worker error. If worker exits without error, it will simply close the channel.
	go func() {
		defer close(objchan)
		var firsterr error
		for _, worker := range errchan {
			if err := <-worker; err != nil && firsterr == nil {
				firsterr = err
			}
		}
		trap(firsterr)
	}()

	return objchan
}

func writeIntoStore(store *common.ObjectStore, objectsPerSecond int, objects <-chan Object) uint64 {
	var n uint64

	const workers = 5
	errchan := make([]chan error, workers)
	for i := 0; i < workers; i++ {
		errchan[i] = make(chan error, 1)
		go func(workerId int) {
			rl := rate.NewLimiter(rate.Every(time.Second/time.Duration(objectsPerSecond/workers)), objectsPerSecond/workers)
			var wn uint64
			defer close(errchan[workerId])
			defer func() { atomic.AddUint64(&n, wn) }()
			for obj := range objects {
				for {
					if err := rl.Wait(context.Background()); err != nil {
						log.Fatalln("rate limit:", err)
					}

					if !obj.Expiration.IsZero() {
						trap(errors.New("writing objects with expiration time is not yet implemented"))
					}
					_, err := store.PutObject(context.TODO(), obj.Key, obj.Data)
					if err != nil && err.HttpStatusCode != http.StatusInternalServerError {
						log.Println("got 500, will retry in 5s")
						time.Sleep(5 * time.Second)
						continue
					} else if err != nil {
						errchan[workerId] <- fmt.Errorf("write: %w", err)
						return
					}

					break
				}
				wn++
			}
		}(i)
	}

	var firsterr error
	for _, worker := range errchan {
		if err := <-worker; err != nil && firsterr == nil {
			firsterr = err
		}
	}
	trap(firsterr)
	return n
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
