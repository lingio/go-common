package common

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

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
	if err := initBucket(mc, bucketName, config); err != nil {
		return nil, err
	}

	return &ObjectStore{
		mc:         mc,
		bucketName: bucketName,
		config:     config,
	}, nil
}

// GetObject attempts to get metadata and read data from the specified file.
func (os ObjectStore) GetObject(file string) ([]byte, ObjectInfo, error) {
	object, err := os.mc.GetObject(context.Background(), os.bucketName, file, minio.GetObjectOptions{
		// TODO: add support for VersionID ?
	})
	if err != nil {
		return nil, ObjectInfo{}, fmt.Errorf("get object: %w", err)
	}
	data, err := ioutil.ReadAll(object)
	if err != nil {
		return nil, ObjectInfo{}, fmt.Errorf("read object: %w", err)
	}
	stat, err := object.Stat()
	if err != nil {
		return nil, ObjectInfo{}, fmt.Errorf("stat object: %w", err)
	}

	return data, objectInfoFromMinio(stat), nil
}

// PutObject uploads the object with pre-configured content type and content disposition.
func (os ObjectStore) PutObject(ctx context.Context, file string, data []byte) (_ ObjectInfo, diderr error) {
	defer os.auditLog(ctx, "Put", file, diderr)
	info, err := os.mc.PutObject(context.Background(), os.bucketName, file, bytes.NewBuffer(data), int64(len(data)), minio.PutObjectOptions{
		ContentType:        os.config.ContentType,
		ContentDisposition: os.config.ContentDisposition,
		// NOTE: Also add support for ContentEncoding ?
	})
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{
		ETag:       info.ETag,
		Expiration: info.Expiration,
		Key:        info.Key,
	}, nil
}

// DeleteObject will attempt to remove the requested file/object.
func (os ObjectStore) DeleteObject(ctx context.Context, file string) (diderr error) {
	defer os.auditLog(ctx, "Delete", file, diderr)
	err := os.mc.RemoveObject(context.Background(), os.bucketName, file, minio.RemoveObjectOptions{
		// TODO: add support for VersionID ?
	})
	if err != nil {
		return err
	}
	return nil
}

// ListObjects performs a recursive object listing.
func (os ObjectStore) ListObjects() <-chan ObjectInfo {
	listing := os.mc.ListObjects(context.Background(), os.bucketName, minio.ListObjectsOptions{
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

// initBucket ensures that a bucket exists and is configured as requested.
func initBucket(mc *minio.Client, bucketName string, config ObjectStoreConfig) error {
	exists, err := mc.BucketExists(context.Background(), bucketName)
	if err != nil {
		return fmt.Errorf("bucket exists: %s: %w", bucketName, err)
	}
	if !exists {
		err := mc.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{
			ObjectLocking: config.ObjectLocking,
		})
		if err != nil {
			return fmt.Errorf("make bucket: %w", err)
		}
	}
	if config.Versioning {
		if err := mc.EnableVersioning(context.Background(), bucketName); err != nil {
			return fmt.Errorf("enable versioning: %w", err)
		}
	}
	if config.Lifecycle != nil {
		if err := mc.SetBucketLifecycle(context.Background(), bucketName, config.Lifecycle); err != nil {
			return fmt.Errorf("set lifecycle: %w", err)
		}
	}
	return nil
}

func (os ObjectStore) auditLog(ctx context.Context, action, object string, err error) {
	ctx = WithBucket(ctx, os.bucketName)
	ctx = WithAction(ctx, action)
	ctx = WithObject(ctx, object)
	if err == nil {
		LogAuditEvent(ctx)
	}
}
