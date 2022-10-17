// redistest pkg contains a table-stakes test suite for our cached storage layer.
//
// To run the test suite:
//
// 1. Ensure redis-server is running & listening port 6379.
//	$ redis-cli ping
//	PONG
// 2. (Optional) Regenerate storage code.
//	$ go generate ./redistest
// 3. Run it.
// 	$ go test ./redistest
//
// Be aware that running the test suite will flush all keys in the default redis database.
//
package redistest

//go:generate bash -c "cd ../storagegen && storagegen ../redistest/storage/spec.json"

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lingio/go-common"
	"github.com/lingio/go-common/redistest/models"
	"github.com/lingio/go-common/redistest/storage"
)

func TestPutAndGet(t *testing.T) {
	tc := storage.NewTestRedisCache(client)

	a := models.Test{
		ID:      "put_and_get",
		Content: "123",
	}

	// ==

	if err := tc.Put(context.TODO(), a, time.Duration(0), ""); err != nil {
		t.Fatal(err)
	}

	if A, _, err := tc.Get(context.TODO(), a.ID); err != nil {
		t.Fatal(err)
	} else if A.Content != a.Content {
		t.Errorf("content: expected %q but got %q", a.Content, A.Content)
	}
}

func TestAllSet(t *testing.T) {
	tc := storage.NewTestRedisCache(client)
	n := 10

	// ==

	for i := 0; i < n; i++ {
		obj := models.Test{
			ID:       fmt.Sprintf("all_set_%v", i),
			Topic:    "house",
			Subtopic: "cottage",
		}

		if err := tc.Put(context.TODO(), obj, time.Duration(0), ""); err != nil {
			t.Fatal(err)
		}
	}

	// ==

	t.Run("GetAllByTopic", func(t *testing.T) {
		t.Run("should return all objects", func(t *testing.T) {
			if all, _, err := tc.GetAllByTopic(context.TODO(), "house"); err != nil {
				t.Fatal(err)
			} else if len(all) != n {
				t.Fatalf("expected %v objects but got %v", n, len(all))
			}
		})

		t.Run("should not return an empty array", func(t *testing.T) {
			if none, _, err := tc.GetAllByTopic(context.TODO(), "nonexistant"); err != nil {
				t.Fatal(err)
			} else if len(none) != 0 {
				t.Fatalf("expected %v objects but got %v", 0, len(none))
			}
		})
	})

	// ==

	t.Run("GetAllByTopicAndSubtopic", func(t *testing.T) {
		t.Run("should return all objects", func(t *testing.T) {
			if all, _, err := tc.GetAllByTopicAndSubtopic(context.TODO(), "house", "cottage"); err != nil {
				t.Fatal(err)
			} else if len(all) != n {
				t.Fatalf("expected %v objects but got %v", n, len(all))
			}
		})

		t.Run("should not return an empty array", func(t *testing.T) {
			if none, _, err := tc.GetAllByTopicAndSubtopic(context.TODO(), "house", "nonexistant"); err != nil {
				t.Fatal(err)
			} else if len(none) != 0 {
				t.Fatalf("expected %v objects but got %v", 0, len(none))
			}
		})
	})

	// ==

	t.Run("Put", func(t *testing.T) {
		obj, _, err := tc.Get(context.TODO(), "all_set_0")
		if err != nil {
			t.Fatal(err)
		}

		obj.Subtopic = "apartment"
		if err := tc.Put(context.TODO(), *obj, time.Duration(0), ""); err != nil {
			t.Fatal(err)
		}

		t.Run("should keep in unchanged indexes", func(t *testing.T) {
			if all, _, err := tc.GetAllByTopic(context.TODO(), "house"); err != nil {
				t.Fatal(err)
			} else if len(all) != n {
				t.Fatalf("expected %v objects but got %v", n, len(all))
			}
		})

		t.Run("should remove object from previous index sets", func(t *testing.T) {
			if other, _, err := tc.GetAllByTopicAndSubtopic(context.TODO(), "house", "cottage"); err != nil {
				t.Fatal(err)
			} else if len(other) != n-1 {
				t.Fatalf("expected %v objects but got %v", n-1, len(other))
			}
		})

		t.Run("should add to new index sets", func(t *testing.T) {
			if single, _, err := tc.GetAllByTopicAndSubtopic(context.TODO(), "house", "apartment"); err != nil {
				t.Fatal(err)
			} else if len(single) != 1 {
				t.Fatalf("expected %v objects but got %v", 1, len(single))
			}
		})
	})
}

// setup connects to redis and flushes the cache
func setup() {
	var err error

	client, err = common.SetupRedisClient(common.RedisConfig{
		Addr: "localhost:6379",
	})

	if err != nil {
		fmt.Println("test startup: redis setup:", err)
		os.Exit(1)
	}

	if err := client.FlushAll(context.TODO()).Err(); err != nil {
		fmt.Println("test startup: flushall:", err)
		os.Exit(1)
	}
}

// shutdown flushes the cache and closes the connection to redis
func shutdown() {
	if err := client.FlushAll(context.TODO()).Err(); err != nil {
		fmt.Println("test shutdown: flushall:", err)
		os.Exit(1)
	}

	if err := client.Close(); err != nil {
		fmt.Println("test shutdown:", err)
		os.Exit(1)
	}
}

// client is a shared redis client by all tests
var client *redis.Client

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}
