package common

import (
	"time"

	"github.com/minio/minio-go/v7"
)

// LingioStore is a simple file-based CRUD database interface.
type LingioStore interface {
	GetObject(file string) ([]byte, ObjectInfo, error)
	PutObject(file string, data []byte) (ObjectInfo, error)
	DeleteObject(file string) error
	ListObjects() <-chan ObjectInfo
}

// ObjectInfo
type ObjectInfo struct {
	Key        string
	Expiration time.Time
	ETag       string
}

func objectInfoFromMinio(info minio.ObjectInfo) ObjectInfo {
	// TODO: Add support for these as well?
	// info.Err
	// info.VersionID
	// info.IsLatest
	// info.LastModified
	return ObjectInfo{
		ETag:       info.ETag,
		Expiration: info.Expiration,
		Key:        info.Key,
	}
}
