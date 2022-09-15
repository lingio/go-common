package common

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

var ErrInvalidRedisConfig = errors.New("redis cache config is not valid")

const redisCacheKeyInitialized = "initialized"
const redisCacheKeyInitializing = "initializing"

// RedisSetupErr wraps an underlying error that occured during cache setup.
type RedisSetupErr struct {
	Err        error
	MasterName string
	ServiceDNS string
}

func (e *RedisSetupErr) Error() string {
	// redis cache: %err
	// failover redis cache with master on a.s.sd.d: %err
	if e == nil {
		return "<nil>"
	}
	s := "redis cache"
	if e.MasterName != "" && e.ServiceDNS != "" {
		s = "failover " + s + " with " + e.MasterName + " on " + e.ServiceDNS
	}
	s = s + ": " + e.Err.Error()
	return s
}
func (e *RedisSetupErr) Unwrap() error { return e.Err }

// RedisCache is a named and versioned cache for a specific collections
type RedisCache struct {
	*redis.Client

	Version string
	Name    string

	WarmedUp AtomicBool

	redsync  *redsync.Redsync
	initLock *redsync.Mutex
}

// NewRedisCache returns an initialized redis cache using name and version.
func NewRedisCache(client *redis.Client, name, version string) *RedisCache {
	rc := &RedisCache{
		Version:  version,
		Name:     name,
		Client:   client,
		WarmedUp: 0,
		redsync:  redsync.New(goredis.NewPool(client)),
	}
	rc.initLock = rc.redsync.NewMutex(rc.baseKey(redisCacheKeyInitializing), redsync.WithExpiry(10*time.Second))
	return rc
}

// SetupRedisClient will attempt to 1) create a failover redis client by looking
// up sentinel addrs using the provided service DNS, or 2) attempt to create a
// simple redis client using the provided simpleAddr. If all three arguments are
// empty, the function will return ErrInvalidRedisConfig.
func SetupRedisClient(cfg RedisConfig) (*redis.Client, error) {
	if cfg.MasterName != "" && cfg.ServiceDNS != "" {
		_, srvs, err := net.LookupSRV("redis", "tcp", cfg.ServiceDNS)
		if err != nil {
			return nil, &RedisSetupErr{Err: err, MasterName: cfg.MasterName, ServiceDNS: cfg.ServiceDNS}
		}
		sentinelAddrs := make([]string, 0)
		for _, srv := range srvs {
			sentinelAddrs = append(sentinelAddrs, fmt.Sprintf("%s:%d", srv.Target, srv.Port))
		}

		failOverOptions := &redis.FailoverOptions{
			MasterName:    cfg.MasterName,
			SentinelAddrs: sentinelAddrs,
			DialTimeout:   time.Second * 5,
			MaxRetries:    3,
			ReadTimeout:   time.Second,
			WriteTimeout:  time.Second,
		}

		if cfg.SentinelPassword != nil || cfg.MasterPassword != nil {
			failOverOptions.SentinelPassword = *cfg.SentinelPassword
			failOverOptions.Password = *cfg.MasterPassword
		}

		return redis.NewFailoverClient(failOverOptions), nil
	}

	if cfg.Addr != "" {
		return redis.NewClient(&redis.Options{
			Addr:         cfg.Addr,
			DialTimeout:  time.Second * 5,
			MaxRetries:   3,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
		}), nil
	}

	return nil, &RedisSetupErr{Err: ErrInvalidRedisConfig}
}

//=============================================================================
// statusProbe interface:

// Started indicates if the cache is reachable.
func (c RedisCache) Started() bool {
	return c.Live()
}

// Ready indicates that the cache is warmed up and ready to serve requests.
// No difference compared to Started() at this time.
func (c RedisCache) Ready() bool {
	return c.WarmedUp.IsSet()
}

// Live indicates if the service is healthy.
func (c RedisCache) Live() bool {
	if val, err := c.Ping(context.TODO()).Result(); err != nil {
		return false
	} else if val != "PONG" {
		return false
	}
	return true
}

func formatRedisCacheKey(elems ...string) string {
	// e.g. people.v1.initialized
	// e.g. people.v1.id
	// e.g. people.v1.etag.id
	return strings.Join(elems, ".")
}

// BaseKey returns a scoped cache key
func (c RedisCache) baseKey(elems ...string) string {
	var s []string
	s = append(s, c.Name, c.Version)
	s = append(s, elems...)
	return formatRedisCacheKey(s...)
}

// Key returns the an index key
func (c RedisCache) Key(keyName, key string) string {
	// Example: $scope.id=p123
	return c.baseKey(keyName) + "=" + key
}

// ETagKey returns the etag key for an index key
func (c RedisCache) ETagKey(keyName, key string) string {
	// Example: GetAllByPartner --> $scope.etag.partnerID=nobina
	return c.baseKey("etag", keyName) + "=" + key
}

// InitKey returns the key for checking and storing initialization status
func (c RedisCache) InitKey() string {
	// Example: $scope.initialized
	return c.baseKey(redisCacheKeyInitialized)
}

// Initialized performs a greedy check if the cache is initialized.
func (c RedisCache) Initialized() (bool, error) {
	v, err := c.Exists(context.TODO(), c.InitKey()).Result()
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
