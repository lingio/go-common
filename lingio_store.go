package common

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
)

// AtomicBool
type AtomicBool int32

func (b *AtomicBool) IsSet() bool { return atomic.LoadInt32((*int32)(b)) != 0 }
func (b *AtomicBool) SetTrue()    { atomic.StoreInt32((*int32)(b), 1) }
func (b *AtomicBool) SetFalse()   { atomic.StoreInt32((*int32)(b), 0) }

// LingioStore is a simple file-based CRUD database interface.
type LingioStore interface {
	GetObject(file string) ([]byte, ObjectInfo, *Error)
	PutObject(ctx context.Context, file string, data []byte) (ObjectInfo, *Error)
	DeleteObject(ctx context.Context, file string) *Error
	ListObjects(context.Context) <-chan ObjectInfo
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
