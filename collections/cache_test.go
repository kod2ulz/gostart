package collections_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kod2ulz/gostart/collections"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		cache            collections.Cache[string, TestUser, error]
		fetcher          func(ctx context.Context, keys []string) ([]TestUser, error)
		fetcherCallCount int
		fetcherMutex     sync.Mutex
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
	})

	JustBeforeEach(func() {
		// This runs after BeforeEach, allowing fetcher to be modified in specific contexts
		cache = collections.NewMemoryCache[string, TestUser, error](
			collections.WithFetcherFunc[string, TestUser, error](fetcher),
			collections.WithEvictionInterval[string, TestUser, error](20*time.Millisecond), // Use a short interval for testing
		)
		DeferCleanup(cache.Stop)
	})

	Context("basic Get operations", func() {
		It("should fetch an item on cache miss", func() {
			user, err := cache.Get(context.Background(), "user1")
			Expect(err).ToNot(HaveOccurred())
			Expect(user).ToNot(BeNil())
			Expect(user.Name).To(Equal("User-user1"))
			Expect(fetcherCallCount).To(Equal(1))
		})

		It("should return a cached item on cache hit", func() {
			// First call to cache the item
			_, err := cache.Get(context.Background(), "user1")
			Expect(err).ToNot(HaveOccurred())
			Expect(fetcherCallCount).To(Equal(1))

			// Second call should be a cache hit
			user, err := cache.Get(context.Background(), "user1")
			Expect(err).ToNot(HaveOccurred())
			Expect(user).ToNot(BeNil())
			Expect(user.Name).To(Equal("User-user1"))
			Expect(fetcherCallCount).To(Equal(1)) // Should not have increased
		})

		It("should return an error if the fetcher fails", func() {
			fetcher = func(ctx context.Context, keys []string) ([]TestUser, error) {
				return nil, fmt.Errorf("fetcher error")
			}
			// Re-initialize cache with the failing fetcher
			cache = collections.NewMemoryCache[string, TestUser, error](
				collections.WithFetcherFunc[string, TestUser, error](fetcher),
			)
			defer cache.Stop()

			user, err := cache.Get(context.Background(), "user1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("fetcher error"))
			Expect(user).To(BeNil())
		})
	})

	Context("TTL and expiration", func() {
		It("should refetch an item after its TTL expires", func() {
			cache.SetTTL("user1", 10*time.Millisecond)

			_, err := cache.Get(context.Background(), "user1")
			Expect(err).ToNot(HaveOccurred())
			Expect(fetcherCallCount).To(Equal(1))

			// Should be a cache hit
			_, err = cache.Get(context.Background(), "user1")
			Expect(err).ToNot(HaveOccurred())
			Expect(fetcherCallCount).To(Equal(1))

			time.Sleep(15 * time.Millisecond)

			// Should be a cache miss, triggering a refetch
			_, err = cache.Get(context.Background(), "user1")
			Expect(err).ToNot(HaveOccurred())
			Expect(fetcherCallCount).To(Equal(2))
		})

		It("should be cleared by the background eviction process", func() {
			cache.SetTTL("user1", 10*time.Millisecond)
			_, err := cache.Get(context.Background(), "user1")
			Expect(err).ToNot(HaveOccurred())
			Expect(fetcherCallCount).To(Equal(1))

			// Wait for eviction to run
			time.Sleep(30 * time.Millisecond)

			// Get again, should trigger a refetch
			_, err = cache.Get(context.Background(), "user1")
			Expect(err).ToNot(HaveOccurred())
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
			_, err := cache.Get(context.Background(), "user1")
			Expect(err).ToNot(HaveOccurred())
			Expect(fetcherCallCount).To(Equal(1))

			// Fetch user1 (hit) and user2 (miss)
			users, err := cache.Fetch(context.Background(), "user1", "user2")
			Expect(err).ToNot(HaveOccurred())
			Expect(users).To(HaveLen(2))
			Expect(fetcherCallCount).To(Equal(2)) // Fetcher called again for user2
		})
	})

	Context("Clear methods", func() {
		It("should clear specific items", func() {
			_, _ = cache.Get(context.Background(), "user1")
			_, _ = cache.Get(context.Background(), "user2")
			Expect(fetcherCallCount).To(Equal(2))

			cache.Clear("user1")

			// Get user2, should be a hit
			_, _ = cache.Get(context.Background(), "user2")
			Expect(fetcherCallCount).To(Equal(2))

			// Get user1, should be a miss
			_, _ = cache.Get(context.Background(), "user1")
			Expect(fetcherCallCount).To(Equal(3))
		})

		It("should clear the entire cache with ClearAll", func() {
			_, _ = cache.Get(context.Background(), "user1")
			_, _ = cache.Get(context.Background(), "user2")
			Expect(fetcherCallCount).To(Equal(2))

			cache.ClearAll()

			_, _ = cache.Get(context.Background(), "user1")
			Expect(fetcherCallCount).To(Equal(3)) // Should be a miss
		})
	})
})
