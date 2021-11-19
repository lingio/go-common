package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/lingio/go-common"
)

// Object is a data structure for transforming and processing object store data.
type Object struct {
	common.ObjectInfo
	Data        []byte
	originalKey string // internal data used for deduplication
}

var (
	newFilenameLen = flag.Int("newfilenamelen", 41, "Length of the filename without partner id. Used to filter unknown partner filter and to select new file in dedup.")
	dedup          = flag.Bool("dedup", false, "specify to run deduplication by stripping partner IDs internally")
	strip          = flag.Bool("strip", false, "specify to strip partner IDs prefix in object keys")
	filter         = flag.Bool("filter", false, "specify to filter out objects with an unknown partner")
)

func main() {
	flag.CommandLine.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage: xform [OPTIONS] [-dedup] [-strip] [-filter]

Example:

MINIO_SECRET=xyz ./objcopy --from=cfg.json > stored.json

# Filter out unknown partners and deduplicate objects
cat stored.json | ./xform -dedup -filter > deduped.json
diff stored.json deduped.json

# Filter, deduplicate and strip partner prefix
cat stored.json | ./xform -dedup -filter -strip > stripped-deduped.json

Note:

Running with -filter will produce an unknown-objects.json file for objects with an unknown partner ID.`)
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
	}
	flag.Parse()

	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	// log to stderr to keep stdout for json data
	log.Default().SetFlags(0)
	log.Default().SetOutput(os.Stderr)

	var objects []Object

	for {
		var obj Object
		if err := decoder.Decode(&obj); err == io.EOF {
			break
		} else {
			trap(err)
		}
		objects = append(objects, obj)
	}

	if *filter {
		var unknown []Object
		objects, unknown = removeUnknownPartners(objects)
		trap(dumpArray(unknown, "./unknown-objects.json"))
	}
	if *dedup || *strip {
		objects = stripPartnersAndDeduplicate(objects)
	}

	for _, obj := range objects {
		trap(encoder.Encode(obj))
	}
}

// We want to be able to run either strip or dedup and produce diffable json output.
func stripPartnersAndDeduplicate(objects []Object) []Object {
	type dedupInfo struct {
		stripped bool
		index    int
	}
	dict := make(map[string]dedupInfo)

	var i int
	for _, obj := range objects {
		var stripped bool
		for _, partnerID := range knownPartners {
			if strings.HasPrefix(obj.Key, partnerID) {
				obj.originalKey = obj.Key
				obj.Key = obj.Key[len(partnerID)+1:]
				stripped = true
				break
			}
		}

		if *dedup {
			// We deduplicate rather conservatively: only replace old files (i.e. stripped).
			if info, exists := dict[obj.Key]; exists && !info.stripped && stripped {
				// if we stripped a partner from this filename and we've already stored a file
				// we ignore this file, because we already have stored our new file
				log.Println("dedup", obj.Key)
				continue
			} else if !exists && !stripped && len(obj.Key) != *newFilenameLen {
				// error on files that couldn't be stripped (no partner prefix exists) which
				// could not be determined to be a new file (without partner prefix).
				log.Fatalf("fatal: '%s' was not stripped and is not a new file (%d != %d)", obj.Key, len(obj.Key), *newFilenameLen)
			} else if exists && info.stripped && !stripped && len(obj.Key) == *newFilenameLen {
				// we have stored a stripped and found a new file, so dedup the stored old file
				log.Println("dedup", obj.Key)
				objects[info.index] = obj
				dict[obj.Key] = dedupInfo{
					stripped: stripped,
					index:    info.index,
				}
			} else {
				dict[obj.Key] = dedupInfo{
					stripped: stripped,
					index:    i,
				}
				// if we were requested not to output stripped, then restore original key
				if !*strip && stripped {
					obj.Key = obj.originalKey
				}
				objects[i] = obj
				i++
			}
		}

		if *strip && !*dedup {
			objects[i] = obj
			i++
		}
	}
	return objects[0:i]
}

func removeUnknownPartners(objects []Object) ([]Object, []Object) {
	var i int
	var unknown []Object
	for _, obj := range objects {
		var isActive bool
		// Skip filtering new files
		if len(obj.Key) == *newFilenameLen {
			isActive = true
		} else {
			for _, partnerID := range knownPartners {
				if strings.HasPrefix(obj.Key, partnerID) {
					isActive = true
					break
				}
			}
		}
		if !isActive {
			log.Println("filter", obj.Key, obj.Key[0:max(0, len(obj.Key)-*newFilenameLen-1)])
			unknown = append(unknown, obj)
		} else {
			objects[i] = obj
			i++
		}
	}
	return objects[0:i], unknown
}

func dumpArray(objects []Object, filename string) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	trap(err)
	defer func() {
		trap(f.Close())
	}()
	encoder := json.NewEncoder(f)
	for _, obj := range objects {
		trap(encoder.Encode(obj))
		if err != nil {
			return err
		}
	}
	return nil
}

func trap(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
