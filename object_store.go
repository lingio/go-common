package common

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/minio/minio-go/v7"
)

// ObjectStore implements the Lingio CRUD database interface on top of minio's object storage engine.
type ObjectStore struct {
	mc         *minio.Client
	bucketName string
	config     ObjectStoreConfig
}

type ObjectStoreConfig struct {
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
		return nil, ObjectInfo{}, minioErrToObjectStoreError(err)
	}
	data, err := ioutil.ReadAll(object)
	if err != nil {
		return nil, ObjectInfo{}, objectStoreErrorE(http.StatusInternalServerError, os.bucketName, "read object", err).Str("minio.Key", file)
	}
	stat, err := object.Stat()
	if err != nil {
		return nil, ObjectInfo{}, minioErrToObjectStoreError(err)
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
		return ObjectInfo{}, minioErrToObjectStoreError(err)
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
		return minioErrToObjectStoreError(err)
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

// checkBucket ensures that a bucket exists and is configured as requested.
func checkBucket(mc *minio.Client, bucketName string) *Error {
	exists, err := mc.BucketExists(context.Background(), bucketName)
	if err != nil {
		return minioErrToObjectStoreError(err)
	}
	if !exists {
		return objectStoreError(http.StatusNotFound, bucketName, "check bucket: bucket does not exist")
	}
	return nil
}

func minioErrToObjectStoreError(err error) *Error {
	minioErr := err.(minio.ErrorResponse)
	return NewErrorE(minioErr.StatusCode, err).
		Str("minio.Message", minioErr.Message).
		Str("minio.Code", minioErr.Code).
		Str("minio.BucketName", minioErr.BucketName).
		Str("minio.Key", minioErr.Key)
}

func objectStoreError(code int, bucket, message string) *Error {
	return NewError(http.StatusNotFound).
		Str("minio.BucketName", bucket).
		Str("minio.Message", message)
}

func objectStoreErrorE(code int, bucket, message string, err error) *Error {
	return NewErrorE(http.StatusNotFound, err).
		Str("minio.BucketName", bucket).
		Str("minio.Message", message)
}
