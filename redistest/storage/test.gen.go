package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lingio/go-common/redistest/models"

	"github.com/go-redis/redis/v8"
	"github.com/lingio/go-common"
	"github.com/minio/minio-go/v7"
	uuid "github.com/satori/go.uuid"

	zl "github.com/rs/zerolog/log"
)

const TestCacheKeyID = "id"
const TestCacheKeyTopic = "topic"
const TestCacheKeyTopicAndSubtopic = "subtopic"

var TestStoreConfig common.ObjectStoreConfig

func init() {
	err := json.Unmarshal([]byte(`
{
	"ContentType": "application/json",
	"ContentDisposition": ""
}
	`), &TestStoreConfig)
	if err != nil {
		panic(fmt.Errorf("error parsing store config: %w", err))
	}
}

type TestStore struct {
	backend common.LingioStore
	cache   TestCache
	ready   common.AtomicBool
}

type TestCache interface {
	Initialized() (bool, error)
	AcquireInitLock(context.Context) error
	ReleaseInitLock(context.Context) error
	Init(context.Context, common.LingioStore) error

	// Primary key operations
	Put(context.Context, models.Test, time.Duration, string) error
	Get(context.Context, string) (*models.Test, string, error)
	Delete(context.Context, string) error

	// Secondary index operations
	GetAllByTopic(ctx context.Context, topic string) ([]models.Test, string, error)
	GetAllByTopicAndSubtopic(ctx context.Context, topic, subtopic string) ([]models.Test, string, error)
}

// testCacheObject is the internally stored cached object.
type testCacheObject struct {
	ETag   string
	Entity models.Test
}

// TestCacheIngest is used during initialization to fill the cache with data from the backend.
type TestCacheIngest struct {
	common.ObjectInfo
	Entity models.Test
	Err    error
}

// NewTestStore configures a new store and initializes the provided cache if required.
func NewTestStore(ctx context.Context, mc *minio.Client, cache TestCache, serviceKey string) (*TestStore, error) {
	// DefaultOjbectStoreConfig || deserialize
	objectStore, err := common.NewObjectStore(mc, "redistest--test", TestStoreConfig)
	if err != nil {
		return nil, fmt.Errorf("creating object store: %w", err)
	}

	encryptedStore, err := common.NewEncryptedStore(objectStore, serviceKey)
	if err != nil {
		return nil, fmt.Errorf("creating encrypted store: %w", err)
	}

	db := &TestStore{
		backend: encryptedStore,
		cache:   cache,
		ready:   0,
	}

	if err := db.cache.Init(ctx, db.backend); err != nil {
		return nil, fmt.Errorf("initializing cache: %w", err)
	}
	db.ready.SetTrue()

	common.RegisterRedisOnConnectHook(func(ctx context.Context) {
		if err := cache.Init(ctx, db.backend); err != nil {
			zl.Error().Err(err).
				Str("component", serviceKey).
				Msg("cache re-initialization failed")
		}
	})
	return db, nil
}

// TestFilename returns the object store filename used for the object identified by the provided id
// TestFilename("id") --> "redistest-id.json"
func TestFilename(id string) string {
	return fmt.Sprintf("redistest-%s.json", id)
}

// StoreName returns the store name of the backing lingio store.
func (s *TestStore) StoreName() string {
	return s.backend.StoreName()
}

//=============================================================================
// Type-safe methods.
//=============================================================================

// Create attempts to store the provided object in store.
func (s *TestStore) Create(ctx context.Context, obj models.Test) (*models.Test, error) {
	if obj.ID != "" {
		// check that the object doesn't exist
		o, _, err := s.Get(ctx, obj.ID)
		if err != nil && !errors.Is(err, common.ErrObjectNotFound) {
			return nil, common.NewErrorE(http.StatusInternalServerError, err).
				Str("ID", obj.ID).Msg("failed query for object")
		}
		if o != nil { // object exists!
			return nil, common.NewError(http.StatusBadRequest).
				Str("ID", obj.ID).Msg("an object with this ID is already stored in the database")
		}
	} else {
		obj.ID = uuid.NewV4().String()
	}
	if err := s.put(ctx, obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// Get attempts to load an object with the specified ID from the store.
func (s *TestStore) Get(ctx context.Context, id string) (*models.Test, string, error) {
	obj, etag, err := s.cache.Get(ctx, id)
	if err == nil {
		return obj, etag, nil
	}

	if errors.Is(err, common.ErrObjectNotFound) {
		data, _, err := s.backend.GetObject(ctx, TestFilename(id))
		if err != nil {
			return nil, "", common.Errorf(err)
		}

		var obj models.Test
		if err := json.Unmarshal([]byte(data), &obj); err != nil {
			return nil, "", common.Errorf(err)
		}

		zl.Warn().Str("id", id).Msg("cache returned nil but backend found data")
		return &obj, "", nil
	}

	return nil, "", err
}

// Put updates or creates the object in both cache and backing store.
func (s *TestStore) Put(ctx context.Context, obj models.Test) error {
	return s.put(ctx, obj)
}

// put does the heavy lifting for both Put and Create methods.
func (s *TestStore) put(ctx context.Context, obj models.Test) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return common.NewErrorE(http.StatusInternalServerError, err).
			Str("ID", obj.ID).Msg("failed to marshal json")
	}
	info, err := s.backend.PutObject(ctx, TestFilename(obj.ID), data)
	if err != nil {
		return common.Errorf(err).Str("ID", obj.ID).Msg("failed to write to minio")
	}

	var expiration time.Duration
	if !info.Expiration.IsZero() {
		expiration = info.Expiration.Sub(time.Now())
	}
	return s.cache.Put(ctx, obj, expiration, info.ETag)
}

// Delete
func (s *TestStore) Delete(ctx context.Context, id string) error {
	if err := s.backend.DeleteObject(ctx, TestFilename(id)); err != nil {
		return common.Errorf(err).Str("ID", id).Msg("failed to delete object in minio")
	}
	return s.cache.Delete(ctx, id)
}

//=============================================================================
// Extra functions from secondary indexes, passes to cache layer
//=============================================================================

// GetAllByTopic fetches all Tests by their Topic
func (s *TestStore) GetAllByTopic(ctx context.Context, topic string) ([]models.Test, string, error) {
	return s.cache.GetAllByTopic(ctx, topic)
}

// GetAllByTopicAndSubtopic fetches all Tests by their
func (s *TestStore) GetAllByTopicAndSubtopic(ctx context.Context, topic, subtopic string) ([]models.Test, string, error) {
	return s.cache.GetAllByTopicAndSubtopic(ctx, topic, subtopic)
}

//=============================================================================
// Cache implementation
//=============================================================================

// TestRedisCache is a redis-backed implementation of TestCache
type TestRedisCache struct {
	*common.RedisCache
}

// NewTestRedisCache writes to leader and reads from follower.
func NewTestRedisCache(client *redis.Client) *TestRedisCache {
	return &TestRedisCache{
		RedisCache: common.NewRedisCache(client, "redistest--test", "1"),
	}
}

// Init checks if the store is initialized before attempting to acquire
// a temporary lock on the cache. To avoid deadlocks, the lock is designed to
// automatically expire after a few seconds. Since we don't know how long time
// the bucket --> cache initialization takes, we need to periodically extend
// the lock while we fill the cache with data from the object store backend.
func (c *TestRedisCache) Init(ctx context.Context, backend common.LingioStore) (resulterr error) {
	var objectsLoaded int
	defer func() {
		if resulterr == nil {
			c.WarmedUp.SetTrue()
			zl.Info().Str("component", "TestStore").Int("objectsLoaded", objectsLoaded).Msg("cache initialized.")
		}
	}()

	for {
		// Perform early bail check since lock acquire can take some time.
		if ok, err := c.Initialized(); err != nil {
			return fmt.Errorf("checking cache: %w", err)
		} else if ok {
			zl.Info().Str("component", "TestStore").Msg("cache found, assuming up-to-date.")
			return nil
		}

		// All concurrent processes will exit before or when this context completes.
		ctx, cancel := context.WithCancel(context.Background())

		// Try to acquire the init lock. It will be valid for a few seconds and we might need to extend it.
		// If something stops the world (GC pause / ??) we might lose the lock (and not know about it).
		// ^ This case is not currently handled.
		if err := c.AcquireInitLock(ctx); err != nil {
			cancel()
			zl.Warn().Str("component", "TestStore").Msg("could not acquire lock to initialize cache. retrying in 5s...")
			time.Sleep(5 * time.Second)
			continue
		}
		defer cancel()

		defer func(ctx context.Context) {
			// Not incredibly important, since the lock will automatically expire anyway.
			if err := c.ReleaseInitLock(ctx); err != nil {
				zl.Warn().Str("component", "TestStore").Err(err).Msg("could not release cache lock")
			}
		}(ctx)

		// Now that we have the lock, ensure that our view of the cache init status is up-to-date.
		if ok, err := c.Initialized(); err != nil {
			return fmt.Errorf("checking cache: %w", err)
		} else if ok {
			zl.Info().Str("component", "TestStore").Msg("cache found, assuming up-to-date.")
			return nil
		}

		zl.Info().Str("component", "TestStore").Msg("cache not initialized, lock acquired, now fetching all data...")

		const NUM_WORKERS = 15

		listing := backend.ListObjects(ctx)
		// workerout is closed by worker pool process
		workerout := make(chan TestCacheIngest, NUM_WORKERS*2)
		// initobj is closed by main control loop
		cacheinit := make(chan TestCacheIngest, NUM_WORKERS*2)

		// Launch worker process.
		var wg sync.WaitGroup
		wg.Add(NUM_WORKERS)
		go func() {
			defer close(workerout)

			worker := func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case req, more := <-listing:
						if !more {
							return
						}

						data, info, err := backend.GetObject(ctx, req.Key)
						if err != nil {
							workerout <- TestCacheIngest{
								ObjectInfo: common.ObjectInfo{},
								Entity:     models.Test{},
								Err:        fmt.Errorf("backend: %w", err),
							}
							return
						}

						var entity models.Test
						if err := json.Unmarshal(data, &entity); err != nil {
							workerout <- TestCacheIngest{
								ObjectInfo: common.ObjectInfo{},
								Entity:     models.Test{},
								Err:        fmt.Errorf("unmarshalling: %w", err),
							}
							return
						}

						if info.Key != TestFilename(entity.ID) {
							zl.Warn().Str("key", info.Key).Msg("skipping object with mismatched filename")
							continue
						}

						workerout <- TestCacheIngest{
							ObjectInfo: info,
							Entity:     entity,
							Err:        nil,
						}
					}
				}
			}

			for i := 0; i < NUM_WORKERS; i++ {
				go worker()
			}
			wg.Wait()
		}()

		// Main control loop: will periodically refresh the cache lock, check
		// for errors in bucket listing and put objects into the cache.
		ticker := time.NewTicker(3 * time.Second)
		objectsLoaded = 0
	main_control_loop:
		for {
			select {
			case <-ticker.C:
				if err := c.AcquireInitLock(ctx); err != nil {
					resulterr = err
					break main_control_loop
				}
			case obj, more := <-workerout:
				if !more {
					break main_control_loop
				}
				if obj.Err != nil {
					resulterr = obj.Err
					break main_control_loop
				}
				var expiration time.Duration
				if !obj.Expiration.IsZero() {
					expiration = obj.Expiration.Sub(time.Now())
				}
				if err := c.Put(ctx, obj.Entity, expiration, obj.ETag); err != nil {
					resulterr = err
					break main_control_loop
				}
				objectsLoaded++
				if objectsLoaded%10_000 == 0 {
					zl.Info().Str("component", "TestStore").
						Int("objectsLoaded", objectsLoaded).
						Msg("initializing cache")

				}
			}
		}
		ticker.Stop()
		close(cacheinit)

		// Only mark cache as initialized if we didn't encounter any error.
		if resulterr == nil {
			if err := c.Client.Set(ctx, c.InitKey(), []byte("1"), 0).Err(); err != nil {
				zl.Warn().Str("component", "TestStore").Msg("could not mark cache as initialized")
				resulterr = err
			}
		}

		// returning will cancel the ctx, which will also cancel listing and workers
		return resulterr
	}

	//unreachable!
}

func (c TestRedisCache) Put(ctx context.Context, obj models.Test, expiration time.Duration, etag string) error {

	co := testCacheObject{
		ETag:   etag,
		Entity: obj,
	}

	// Fetch the previous version of this object (if there is any)
	orig, _, err := c.Get(ctx, obj.ID)
	if err != nil && !errors.Is(err, common.ErrObjectNotFound) {
		return err
	}

	// Primary index: ID
	if err = c.put(ctx, c.Key(TestCacheKeyID, obj.ID), co, expiration); err != nil {
		return err
	}

	var idx string
	// Set index: Topic
	idx = CompoundIndex(obj.Topic)
	c.Client.SAdd(ctx, c.Key(TestCacheKeyTopic, idx), obj.ID)
	err = c.Client.Incr(ctx, c.ETagKey(TestCacheKeyTopic, idx)).Err()
	if err != nil {
		return common.NewErrorE(http.StatusInternalServerError, err)
	}

	// Set index: TopicAndSubtopic
	idx = CompoundIndex(obj.Topic, obj.Subtopic)
	c.Client.SAdd(ctx, c.Key(TestCacheKeyTopicAndSubtopic, idx), obj.ID)
	err = c.Client.Incr(ctx, c.ETagKey(TestCacheKeyTopicAndSubtopic, idx)).Err()
	if err != nil {
		return common.NewErrorE(http.StatusInternalServerError, err)
	}

	// Delete old secondary indexes if they changed
	if orig != nil {

		// Topic depends on (.Topic)
		if obj.Topic != orig.Topic {
			idx = CompoundIndex(orig.Topic)
			c.Client.SRem(ctx, c.Key(TestCacheKeyTopic, idx), orig.ID)
			err := c.Client.Incr(ctx, c.ETagKey(TestCacheKeyTopic, idx)).Err()
			if err != nil {
				return common.NewErrorE(http.StatusInternalServerError, err)
			}
		}

		// TopicAndSubtopic depends on (.Topic, .Subtopic)
		if obj.Topic != orig.Topic || obj.Subtopic != orig.Subtopic {
			idx = CompoundIndex(orig.Topic, orig.Subtopic)
			c.Client.SRem(ctx, c.Key(TestCacheKeyTopicAndSubtopic, idx), orig.ID)
			err := c.Client.Incr(ctx, c.ETagKey(TestCacheKeyTopicAndSubtopic, idx)).Err()
			if err != nil {
				return common.NewErrorE(http.StatusInternalServerError, err)
			}
		}

	}
	return nil
}

func (c TestRedisCache) put(ctx context.Context, fullKey string, co testCacheObject, expiration time.Duration) error {
	data, err := json.Marshal(co)
	if err != nil {
		return common.NewErrorE(http.StatusInternalServerError, err)
	}
	cmd := c.Client.Set(ctx, fullKey, data, expiration)
	if _, err := cmd.Result(); err != nil {
		return common.NewErrorE(http.StatusInternalServerError, err)
	}
	return nil
}

func (c TestRedisCache) get(ctx context.Context, keyName string, key string) (*models.Test, string, error) {
	cmd := c.Client.Get(ctx, c.Key(keyName, key))
	data, err := cmd.Result()
	if errors.Is(err, redis.Nil) {
		return nil, "", common.Errorf(common.ErrObjectNotFound, http.StatusNotFound)
	} else if err != nil {
		return nil, "", common.NewErrorE(http.StatusInternalServerError, err)
	}
	var co testCacheObject
	if err := json.Unmarshal([]byte(data), &co); err != nil {
		return nil, "", common.NewErrorE(http.StatusInternalServerError, err)
	}
	return &co.Entity, co.ETag, nil
}

// Get a cached Test by it's ID
func (c TestRedisCache) Get(ctx context.Context, id string) (*models.Test, string, error) {
	return c.get(ctx, TestCacheKeyID, id)
}

// GetAllByTopic fetches all cached Tests by their Topic
func (c *TestRedisCache) GetAllByTopic(ctx context.Context, topic string) ([]models.Test, string, error) {
	idx := CompoundIndex(topic)
	keys, err := c.Client.SMembers(ctx, c.Key(TestCacheKeyTopic, idx)).Result()
	if err != nil {
		return nil, "", common.NewErrorE(http.StatusInternalServerError, err)
	}

	objs := make([]models.Test, 0, 5)
	for _, key := range keys {
		obj, _, err := c.Get(ctx, key)
		if err != nil {
			return nil, "", err
		}
		objs = append(objs, *obj)
	}

	etag, err := c.Client.Get(ctx, c.ETagKey(TestCacheKeyTopic, idx)).Result()
	if err == redis.Nil && len(keys) == 0 {
		return objs, "", nil
	} else if err != nil {
		return nil, "", common.NewErrorE(http.StatusInternalServerError, err)
	}
	return objs, etag, nil
}

// GetAllByTopicAndSubtopic fetches all cached Tests by their
func (c *TestRedisCache) GetAllByTopicAndSubtopic(ctx context.Context, topic, subtopic string) ([]models.Test, string, error) {
	idx := CompoundIndex(topic, subtopic)
	keys, err := c.Client.SMembers(ctx, c.Key(TestCacheKeyTopicAndSubtopic, idx)).Result()
	if err != nil {
		return nil, "", common.NewErrorE(http.StatusInternalServerError, err)
	}

	objs := make([]models.Test, 0, 5)
	for _, key := range keys {
		obj, _, err := c.Get(ctx, key)
		if err != nil {
			return nil, "", err
		}
		objs = append(objs, *obj)
	}

	etag, err := c.Client.Get(ctx, c.ETagKey(TestCacheKeyTopicAndSubtopic, idx)).Result()
	if err == redis.Nil && len(keys) == 0 {
		return objs, "", nil
	} else if err != nil {
		return nil, "", common.NewErrorE(http.StatusInternalServerError, err)
	}
	return objs, etag, nil
}

func (c *TestRedisCache) Delete(ctx context.Context, id string) error {
	var idx string
	o, _, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	// Delete ID-cache
	keys := []string{c.Key(TestCacheKeyID, id)}

	// Delete all keys at the same time
	err = c.Client.Del(ctx, keys...).Err()
	if err != nil {
		return common.NewErrorE(http.StatusInternalServerError, err)
	}

	// Remove from 'set' secondary index: Topic
	idx = CompoundIndex(o.Topic)
	c.Client.SRem(ctx, c.Key(TestCacheKeyTopic, idx), o.ID)
	err = c.Client.Incr(ctx, c.ETagKey(TestCacheKeyTopic, idx)).Err()
	if err != nil {
		return common.NewErrorE(http.StatusInternalServerError, err)
	}

	// Remove from 'set' secondary index: TopicAndSubtopic
	idx = CompoundIndex(o.Topic, o.Subtopic)
	c.Client.SRem(ctx, c.Key(TestCacheKeyTopicAndSubtopic, idx), o.ID)
	err = c.Client.Incr(ctx, c.ETagKey(TestCacheKeyTopicAndSubtopic, idx)).Err()
	if err != nil {
		return common.NewErrorE(http.StatusInternalServerError, err)
	}

	return nil
}
