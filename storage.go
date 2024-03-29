package common

import (
	"encoding/json"
	"io/ioutil"
	"log"

	zl "github.com/rs/zerolog/log"
)

const (
	INDEX_TYPE_SET    = "set"
	INDEX_TYPE_UNIQUE = "unique"
)

type ServiceStorageSpec struct {
	ServiceName string
	Buckets     []BucketSpec
}

type BucketSpec struct {
	TypeName         string
	DbTypeName       string // the name of the stored type
	BucketName       string
	Template         string
	Version          string
	IdName           *string // name of the the uuid/guid field
	SecondaryIndexes []SecondaryIndex
	GetAll           *bool
	FilenameFormat   string
	Config           *ObjectStoreConfig
}

type SecondaryIndex struct {
	Key  string           // sugar for using Keys[{oneKey}]
	Keys []IndexComponent // an ordered list of index keys for a composite index

	Name, Type, CacheKey string
	Optional             bool // is the whole index optional?
}

type IndexComponent struct {
	Key      string
	Param    string
	Optional bool
	KeyType  string

	// Exclude this component when generating the composite index?
	// Very useful when checking for an optional parent.
	ExclFromIndex bool
}

func ReadStorageSpec(filename string) ServiceStorageSpec {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Str("filename", filename).Msg("failed to load storage spec file")
	}

	var spec ServiceStorageSpec
	err = json.Unmarshal(file, &spec)
	if err != nil {
		zl.Fatal().Str("err", err.Error()).Str("filename", filename).Msg("failed to unmarshal storage spec file")
	}
	return spec
}

func IndexMethodName(settype, attrname string) string {
	var methodName string
	switch settype {
	case INDEX_TYPE_UNIQUE:
		methodName = "GetBy"
	case INDEX_TYPE_SET:
		methodName = "GetAllBy"
	default:
		log.Fatalf("method name: unknown index type '%s'\n", settype)
	}
	return methodName + attrname
}
