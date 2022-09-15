package common

// All ObjectStore methods should wrap and construct errors using the bucketError
// and objectError functions to ensure that bucket name and file name are included.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/minio/minio-go/v7"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ErrBucketDoesNotExist is a proxy for detecting this particular error case in calling code.
var ErrBucketDoesNotExist = errors.New("bucket does not exist")
var ErrObjectNotFound = errors.New("object not found")

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
func (os ObjectStore) GetObject(ctx context.Context, file string) (_ []byte, _ ObjectInfo, lerr error) {
	ctx, span := tracer.Start(ctx, "object_store.GetObject", trace.WithAttributes(
		attribute.String("file", file),
	))
	defer span.End()
	defer span.RecordError(lerr)

	object, err := os.mc.GetObject(context.Background(), os.bucketName, file, minio.GetObjectOptions{
		// TODO: add support for VersionID ?
	})
	if err != nil {
		return nil, ObjectInfo{}, objectError(err, os.bucketName, file, "Could not get object")
	}

	defer func() {
		if err := object.Close(); err != nil && lerr == nil {
			lerr = objectError(err, os.bucketName, file, "Could not close object")
		} else if err != nil && lerr != nil {
			// record this error, but dont overwrite the existing lerr
			span.RecordError(err)
		}
	}()

	// object.Read/Stat calls are mutex-guarded so there is no parallelism speedup
	data, err := ioutil.ReadAll(object)
	if err != nil {
		return nil, ObjectInfo{}, objectError(err, os.bucketName, file, "Could not read object data.")
	}
	stat, err := object.Stat()
	if err != nil {
		return nil, ObjectInfo{}, objectError(err, os.bucketName, file, "Could not get object stat info.")
	}

	return data, objectInfoFromMinio(stat), nil
}

// PutObject uploads the object with pre-configured content type and content disposition.
func (os ObjectStore) PutObject(ctx context.Context, file string, data []byte) (_ ObjectInfo, diderr error) {
	ctx, span := tracer.Start(ctx, "object_store.PutObject", trace.WithAttributes(
		attribute.String("file", file),
	))
	defer span.End()
	defer span.RecordError(diderr)

	defer logObjectStoreAuditEvent(ctx, "Put", os.bucketName, file, diderr)
	info, err := os.mc.PutObject(ctx, os.bucketName, file, bytes.NewBuffer(data), int64(len(data)), minio.PutObjectOptions{
		ContentType:        os.config.ContentType,
		ContentDisposition: os.config.ContentDisposition,
		// NOTE: Also add support for ContentEncoding ?
	})
	if err != nil {
		return ObjectInfo{}, objectError(err, os.bucketName, file, "Could not update object data.")
	}
	return ObjectInfo{
		ETag:       info.ETag,
		Expiration: info.Expiration,
		Key:        info.Key,
	}, nil
}

// DeleteObject will attempt to remove the requested file/object.
func (os ObjectStore) DeleteObject(ctx context.Context, file string) (diderr error) {
	ctx, span := tracer.Start(ctx, "object_store.DeleteObject", trace.WithAttributes(
		attribute.String("file", file),
	))
	defer span.End()
	defer func() {
		if diderr != nil {
			span.RecordError(diderr)
		}
	}()
	defer logObjectStoreAuditEvent(ctx, "Delete", os.bucketName, file, diderr)
	err := os.mc.RemoveObject(ctx, os.bucketName, file, minio.RemoveObjectOptions{
		// TODO: add support for VersionID ?
	})
	if err != nil {
		return objectError(err, os.bucketName, file, "Could not remove object.")
	}
	return nil
}

// ListObjects performs a recursive object listing.
func (os ObjectStore) ListObjects(ctx context.Context) <-chan ObjectInfo {
	ctx, span := tracer.Start(ctx, "object_store.ListObjects")
	listing := os.mc.ListObjects(ctx, os.bucketName, minio.ListObjectsOptions{
		Recursive: true,
		// add support for WithVersions ?
	})

	objects := make(chan ObjectInfo, 10)
	go func() {
		defer span.End()
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

func (os ObjectStore) StoreName() string {
	return os.bucketName
}

func logObjectStoreAuditEvent(ctx context.Context, action, bucket, object string, err error) {
	ctx = WithBucket(ctx, bucket)
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
		return bucketError(err, bucketName, "Could not check if bucket exists.")
	}
	if !exists {
		return bucketError(minio.ErrorResponse{Code: "NoSuchBucket"}, bucketName, "Bucket does not exist.")
	}
	return nil
}

// bucketError is a helper function for wrapping bucket-op errors.
func bucketError(err error, bucket, msg string) error {
	return wrapMinioError(err).Caller(1).Str("bucket", bucket).Msg(msg)
}

// objectError is a helper function for wrapping object-op errors.
func objectError(err error, bucket, file, msg string) error {
	return wrapMinioError(err).Caller(1).Str("bucket", bucket).Str("file", file).Msg(msg)
}

// wrapMinioError is a helper for wrapping an API-specific error in our error type.
func wrapMinioError(err error) *Error {
	if merr, ok := err.(minio.ErrorResponse); ok {
		var lerr *Error
		switch merr.Code {
		case "NoSuchBucket":
			lerr = NewErrorE(http.StatusNotFound, ErrBucketDoesNotExist).
				Str("minio", err.Error())
		case "NoSuchKey":
			lerr = NewErrorE(http.StatusNotFound, ErrObjectNotFound).
				Str("minio", err.Error())
		default:
			lerr = NewErrorE(merr.StatusCode, err)
		}
		return lerr.Str("code", merr.Code)
	}
	return NewErrorE(http.StatusInternalServerError, err)
}
