package common

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

// RedisCache is a versioned,
type RedisCache struct {
	Version  string
	Name     string
	Follower *redis.Client
	Leader   *redis.Client
	WarmedUp AtomicBool

	rs       *redsync.Redsync
	initLock *redsync.Mutex
}

func NewRedisCache(leaderHost, followerHost string, name, version string) *RedisCache {
	leader := redis.NewClient(&redis.Options{
		Addr:     leaderHost,
		Password: "",
		DB:       0,
	})

	follower := redis.NewClient(&redis.Options{
		Addr:     followerHost,
		Password: "",
		DB:       0,
	})

	rs := redsync.New(goredis.NewPool(leader))

	return &RedisCache{
		Version:  version,
		Name:     name,
		Follower: follower,
		Leader:   leader,
		WarmedUp: 0,
		rs:       rs,
		initLock: rs.NewMutex(name+"."+version+".initializing", redsync.WithExpiry(10*time.Second)),
	}
}

//=============================================================================
// statusProbe interface:

// Started indicates if the cache is warmed up.
func (c RedisCache) Started() bool {
	return c.WarmedUp.IsSet()
}

// Ready indicates that the cache is warmed up and ready to serve requests.
// No difference compared to Started() at this time.
func (c RedisCache) Ready() bool {
	return c.WarmedUp.IsSet()
}

// Live indicates if the service is healthy.
func (c RedisCache) Live() bool {
	if val, err := c.Leader.Ping(context.TODO()).Result(); err != nil {
		return false
	} else if val != "PONG" {
		return false
	}

	if val, err := c.Follower.Ping(context.TODO()).Result(); err != nil {
		return false
	} else if val != "PONG" {
		return false
	}

	return true
}

// Key returns the primary index key
func (c RedisCache) Key(keyName string, key string) string {
	// Example: people.v1.id=p123
	return c.Name + "." + c.Version + "." + keyName + "=" + key
}

// ETagKey returns the secondary index etag key
func (c RedisCache) ETagKey(keyName, key string) string {
	// Example: GetAllByPartner --> people.v1.etag.partnerID=nobina
	return c.Name + "." + c.Version + ".etag" + "." + keyName + "=" + key
}

// InitKey returns the key for checking and storing initialization status
func (c RedisCache) InitKey() string {
	// Example:  people.v1.initialized
	return c.Name + "." + c.Version + ".initialized"
}

// Initialized performs a greedy check if the cache is initialized.
func (c RedisCache) Initialized() (bool, error) {
	v, err := c.Follower.Exists(context.TODO(), c.InitKey()).Result()
	if err != nil {
		return false, err
	}
	return v != 0, nil
}

// AcquireInitLock attempts to either lock or extend the currently existing init lock.
func (c RedisCache) AcquireInitLock(ctx context.Context) error {
	if c.initLock.Until().IsZero() || c.initLock.Until().Before(time.Now()) {
		return c.initLock.LockContext(ctx)
	}
	if extended, err := c.initLock.ExtendContext(ctx); err != nil {
		return nil
	} else if !extended && c.initLock.Until().Before(time.Now()) {
		return errors.New("redis cache: could not extend expired lock")
	}
	return nil
}

// ReleaseInitLock attempts to release the init lock.
func (c RedisCache) ReleaseInitLock(ctx context.Context) error {
	unlocked, err := c.initLock.UnlockContext(ctx)
	if err != nil {
		return err
	}
	if !unlocked {
		return errors.New("redis cache: could not unlock lock")
	}
	return nil
}
