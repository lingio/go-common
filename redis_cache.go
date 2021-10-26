package common

import (
	"context"

	"github.com/go-redis/redis/v8"
)

// RedisCache is a versioned,
type RedisCache struct {
	Version  string
	Name     string
	Follower *redis.Client
	Leader   *redis.Client

	WarmedUp AtomicBool
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

	return &RedisCache{
		Version:  version,
		Name:     name,
		Follower: follower,
		Leader:   leader,
		WarmedUp: 0,
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
	// Example: people.v1.o.id=p123
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
