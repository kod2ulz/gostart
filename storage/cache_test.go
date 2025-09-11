
package storage_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kod2ulz/gostart/logr"
	"github.com/kod2ulz/gostart/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

// TestUser is a mock model for testing the cache
type TestUser struct {
	ID   string
	Name string
}

func (u TestUser) Key() string {
	return u.ID
}

var _ = Describe("MemoryCache", func() {

	var (
		cache         storage.Cache[string, TestUser, error]
		fetcher       func(ctx context.Context, keys []string) ([]TestUser, error)
		fetcherCallCount int
		fetcherMutex  sync.Mutex
	)

	BeforeEach(func() {
		// Reset the fetcher and its call count for each test
		fetcherMutex.Lock()
		fetcherCallCount = 0
		fetcher = func(ctx context.Context, keys []string) ([]TestUser, error) {
			fetcherCallCount++
			var users []TestUser
			for _, key := range keys {
				users = append(users, TestUser{ID: key, Name: fmt.Sprintf("User-%s", key)})
			}
			return users, nil
		}
		fetcherMutex.Unlock()

		// Initialize logger
		logr.SetUpLogger(logrus.NewEntry(logrus.New()))
		logger := logr.Log()
		logger.Logger.SetOutput(GinkgoWriter)
		logger.Logger.SetLevel(logrus.DebugLevel)
	})

	JustBeforeEach(func() {
		// This runs after BeforeEach, allowing fetcher to be modified in specific contexts
		cache = storage.NewMemoryCache[string, TestUser, error](
			logr.Log(),
			storage.WithFetcherFunc[string, TestUser, error](fetcher),
		)
	})

	Context("basic Get operations", func() {
		It("should fetch an item on cache miss", func() {
			user := cache.Get("user1")
			Expect(user).ToNot(BeNil())
			Expect(user.Name).To(Equal("User-user1"))
			Expect(fetcherCallCount).To(Equal(1))
		})

		It("should return a cached item on cache hit", func() {
			// First call to cache the item
			_ = cache.Get("user1")
			Expect(fetcherCallCount).To(Equal(1))

			// Second call should be a cache hit
			user := cache.Get("user1")
			Expect(user).ToNot(BeNil())
			Expect(user.Name).To(Equal("User-user1"))
			Expect(fetcherCallCount).To(Equal(1)) // Should not have increased
		})
	})

	Context("TTL and expiration", func() {
		It("should refetch an item after its TTL expires", func() {
			cache.SetTTL("user1", 10*time.Millisecond)

			_ = cache.Get("user1")
			Expect(fetcherCallCount).To(Equal(1))

			// Should be a cache hit
			_ = cache.Get("user1")
			Expect(fetcherCallCount).To(Equal(1))

			time.Sleep(15 * time.Millisecond)

			// Should be a cache miss, triggering a refetch
			_ = cache.Get("user1")
			Expect(fetcherCallCount).To(Equal(2))
		})
	})

	Context("Fetch method", func() {
		It("should fetch multiple items at once", func() {
			users, err := cache.Fetch(context.Background(), "user1", "user2")
			Expect(err).ToNot(HaveOccurred())
			Expect(users).To(HaveLen(2))
			Expect(fetcherCallCount).To(Equal(1))
		})

		It("should only fetch items that are not in the cache", func() {
			// Pre-cache user1
			_ = cache.Get("user1")
			Expect(fetcherCallCount).To(Equal(1))

			// Fetch user1 (hit) and user2 (miss)
			_, err := cache.Fetch(context.Background(), "user1", "user2")
			Expect(err).ToNot(HaveOccurred())
			Expect(fetcherCallCount).To(Equal(2)) // Fetcher called again for user2
		})
	})

	Context("Clear method", func() {
		It("should clear the entire cache", func() {
			_ = cache.Get("user1")
			_ = cache.Get("user2")
			Expect(fetcherCallCount).To(Equal(2))

			cache.Clear()

			_ = cache.Get("user1")
			Expect(fetcherCallCount).To(Equal(3)) // Should be a miss
		})
	})
})
