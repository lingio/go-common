package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

var ErrBucketDoesNotExist = errors.New("bucket does not exist")

// ObjectStore implements the Lingio CRUD database interface on top of minio's object storage engine.
type ObjectStore struct {
	mc         *minio.Client
	bucketName string
	config     ObjectStoreConfig
}

type ObjectStoreConfig struct {
	// Versioning enables/disables per-object versions. Remember to specify retain policy to avoid unlimited version history. Defaults to false.
	Versioning bool
	// ObjectLocking enables/disables per-object lock against unwanted changes and removal. Defaults to false.
	ObjectLocking bool
	// Lifecycle describes the bucket lifecycle configuration. Defaults to nil (disabled).
	Lifecycle *lifecycle.Configuration
	// ContentType specifies the object content mime-type. Defaults to "application/json".
	ContentType string
	// ContentDisposition specifies the object disposition mime-type. Defaults to "".
	ContentDisposition string
}

// NewObjectStore attempts to initialize a new bucket if it does not already exist.
func NewObjectStore(mc *minio.Client, bucketName string, config ObjectStoreConfig) (*ObjectStore, error) {
	if err := checkBucket(mc, bucketName); err != nil {
		return nil, err
	}

	return &ObjectStore{
		mc:         mc,
		bucketName: bucketName,
		config:     config,
	}, nil
}

// GetObject attempts to get metadata and read data from the specified file.
func (os ObjectStore) GetObject(file string) ([]byte, ObjectInfo, *Error) {
	object, err := os.mc.GetObject(context.Background(), os.bucketName, file, minio.GetObjectOptions{
		// TODO: add support for VersionID ?
	})
	if err != nil {
		return nil, ObjectInfo{}, objectStoreError(err, os.bucketName, file)
	}
	data, err := ioutil.ReadAll(object)
	if err != nil {
		return nil, ObjectInfo{}, objectStoreError(err, os.bucketName, file)
	}
	stat, err := object.Stat()
	if err != nil {
		return nil, ObjectInfo{}, objectStoreError(err, os.bucketName, file)
	}

	return data, objectInfoFromMinio(stat), nil
}

// PutObject uploads the object with pre-configured content type and content disposition.
func (os ObjectStore) PutObject(ctx context.Context, file string, data []byte) (_ ObjectInfo, diderr *Error) {
	defer os.auditLog(ctx, "Put", file, diderr)
	info, err := os.mc.PutObject(ctx, os.bucketName, file, bytes.NewBuffer(data), int64(len(data)), minio.PutObjectOptions{
		ContentType:        os.config.ContentType,
		ContentDisposition: os.config.ContentDisposition,
		// NOTE: Also add support for ContentEncoding ?
	})
	if err != nil {
		return ObjectInfo{}, objectStoreError(err, os.bucketName, file)
	}
	return ObjectInfo{
		ETag:       info.ETag,
		Expiration: info.Expiration,
		Key:        info.Key,
	}, nil
}

// DeleteObject will attempt to remove the requested file/object.
func (os ObjectStore) DeleteObject(ctx context.Context, file string) (diderr *Error) {
	defer os.auditLog(ctx, "Delete", file, diderr)
	err := os.mc.RemoveObject(ctx, os.bucketName, file, minio.RemoveObjectOptions{
		// TODO: add support for VersionID ?
	})
	if err != nil {
		return objectStoreError(err, os.bucketName, file)
	}
	return nil
}

// ListObjects performs a recursive object listing.
func (os ObjectStore) ListObjects(ctx context.Context) <-chan ObjectInfo {
	listing := os.mc.ListObjects(ctx, os.bucketName, minio.ListObjectsOptions{
		Recursive: true,
		// add support for WithVersions ?
	})

	objects := make(chan ObjectInfo, 10)
	go func() {
		defer close(objects)
		for objectInfo := range listing {
			if objectInfo.Err == io.EOF {
				return
			}
			objects <- objectInfoFromMinio(objectInfo)
		}
	}()

	return objects
}

func (os ObjectStore) auditLog(ctx context.Context, action, object string, err error) {
	ctx = WithBucket(ctx, os.bucketName)
	ctx = WithAction(ctx, action)
	ctx = WithObject(ctx, object)
	if err == nil {
		LogAuditEvent(ctx)
	}
}

// checkBucket checks if the bucket exists and that the client has some form
// of access to it. If the bucket does not exist, an error is returned.
func checkBucket(mc *minio.Client, bucketName string) error {
	exists, err := mc.BucketExists(context.Background(), bucketName)
	if err != nil {
		return objectStoreError(err, bucketName, "").
			Msg("error calling s3::BucketExists")
	}
	if !exists {
		err := fmt.Errorf("%w: %s", ErrBucketDoesNotExist, bucketName)
		return objectStoreError(err, bucketName, "").
			Msg("bucket does not exist")
	}
	return nil
}

// objectStoreError returns a http error message, by attempting to cast the
// provided error as a minio.ErrorResponse, falling back to a 500 status error.
// It is expected that the caller fills in the Msg field.
func objectStoreError(err error, bucket, key string) *Error {
	var lerr *Error
	if s3err, ok := err.(minio.ErrorResponse); ok {
		lerr = NewErrorE(s3err.StatusCode, err).
			Str("minio.Message", s3err.Message).
			Str("minio.Code", s3err.Code)
	} else if nerr := errors.Unwrap(err); nerr == ErrBucketDoesNotExist {
		lerr = NewErrorE(http.StatusNotFound, err)
	} else {
		lerr = NewErrorE(http.StatusInternalServerError, err)
	}

	if bucket != "" {
		lerr.Str("minio.BucketName", bucket)
	}
	if key != "" {
		lerr.Str("minio.Key", key)
	}
	return lerr
}
