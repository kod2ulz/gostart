package config

import (
	"fmt"
	"sync"
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

// consulCacheEntry holds a cached value and its expiration time.
type consulCacheEntry struct {
	value      string
	expiration time.Time
}

// consulSource manages the Consul configuration source.
type consulSource struct {
	mu       sync.RWMutex
	client   *consulapi.Client
	cache    map[string]consulCacheEntry
	cacheTTL time.Duration
}

// Consul provides access to the Consul configuration source.
var Consul consulSource

// Client returns the shared Consul client.
func (s *consulSource) Client() *consulapi.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

// Endpoint configures the shared Consul client.
// It reads the address from configuration (CONSUL_HTTP_ADDR).
func (s *consulSource) Endpoint() (*consulSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return s, nil // Already configured
	}

	addr := Get("CONSUL_HTTP_ADDR").String()
	if addr == "" {
		return nil, fmt.Errorf("consul address not configured (set CONSUL_HTTP_ADDR)")
	}

	conf := consulapi.DefaultConfig()
	conf.Address = addr

	client, err := consulapi.NewClient(conf)
	if err != nil {
		return nil, err
	}

	s.client = client
	return s, nil
}

// WithCache sets the cache TTL for discovered service URLs.
func (s *consulSource) WithCache(ttl time.Duration) *consulSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheTTL = ttl
	if s.cache == nil {
		s.cache = make(map[string]consulCacheEntry)
	}
	return s
}

// GetServiceUrl retrieves the URL for a given service from Consul.
func (s *consulSource) GetServiceUrl(serviceName string) (string, error) {
	s.mu.RLock()
	// 1. Check cache
	useCache := s.cache != nil && s.cacheTTL > 0
	if useCache {
		if entry, found := s.cache[serviceName]; found && time.Now().Before(entry.expiration) {
			s.mu.RUnlock()
			return entry.value, nil
		}
	}
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return "", fmt.Errorf("consul client not initialized, call Endpoint() first")
	}

	// 2. Get from Consul
	services, _, err := client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return "", err
	}
	if len(services) == 0 {
		return "", fmt.Errorf("service '%s' not found in consul", serviceName)
	}

	service := services[0].Service
	addr := service.Address
	if addr == "" {
		addr = "localhost"
	}
	url := fmt.Sprintf("http://%s:%d", addr, service.Port)

	// 3. Update cache
	s.mu.Lock()
	if useCache {
		s.cache[serviceName] = consulCacheEntry{
			value:      url,
			expiration: time.Now().Add(s.cacheTTL),
		}
	}
	s.mu.Unlock()

	return url, nil
}

