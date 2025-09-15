package config

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/kod2ulz/gostart/collections"
)

// VaultSource manages the Vault configuration source.
type VaultSource struct {
	mu       sync.RWMutex
	client   *api.Client
	cache    collections.Cache[string, configValue, error]
	cacheTTL time.Duration
}

// Vault provides access to the Vault configuration source.
var Vault VaultSource

// Client returns the underlying Vault API client.
func (s *VaultSource) Client() *api.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

// Endpoint configures the Vault client with an address and token, and initializes the cache.
func (s *VaultSource) Endpoint(addr, token string) (*VaultSource, error) {
	conf := api.DefaultConfig()
	conf.Address = addr

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = client

	fetcher := func(ctx context.Context, keys []string) ([]configValue, error) {
		return s.getManyFromVault(ctx, keys)
	}

	ttl := s.cacheTTL
	if ttl == 0 {
		ttl = collections.DefaultCacheKeyTTL
	}

	s.cache = collections.NewMemoryCache[string, configValue, error](
		collections.WithFetcherFunc[string, configValue, error](fetcher),
		collections.WithDefaultTTL[string, configValue, error](ttl),
	)

	return s, nil
}

// WithCacheTTL sets the cache TTL for the Vault source. Must be called before Endpoint().
func (s *VaultSource) WithCacheTTL(ttl time.Duration) *VaultSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheTTL = ttl
	return s
}

// Get retrieves a secret from Vault, using the cache.
// The key is expected to be in the format "path/to/secret.key-in-secret".
func (s *VaultSource) Get(key string, defaultValue ...interface{}) Value {
	if s.cache == nil {
		return "" // Not configured
	}

	val, err := s.cache.Get(context.Background(), key)
	if err == nil && val != nil {
		return Value(val.value)
	}

	// Fallback to default if provided, as Vault doesn't have a seeding mechanism.
	if len(defaultValue) > 0 {
		return Value(fmt.Sprint(defaultValue[0]))
	}

	return ""
}

func (s *VaultSource) getManyFromVault(ctx context.Context, keys []string) ([]configValue, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("vault client not configured")
	}

	// Group keys by path to read multiple secrets from the same path at once
	keysByPath := make(map[string][]string)
	for _, key := range keys {
		path, _ := parseVaultKey(key)
		if path != "" {
			keysByPath[path] = append(keysByPath[path], key)
		}
	}

	values := make([]configValue, 0, len(keys))
	var firstErr error

	for path, pathKeys := range keysByPath {
		secret, err := client.Logical().Read(path)
		if err != nil {
		if firstErr == nil {
				firstErr = err // Capture the first error we encounter
			}
			continue // Can't read this path
		}
		if secret == nil || secret.Data == nil {
			continue
		}
		if data, ok := secret.Data["data"].(map[string]interface{}); ok {
			for _, key := range pathKeys {
				_, secretKey := parseVaultKey(key)
				if val, found := data[secretKey]; found {
					values = append(values, configValue{key: key, value: fmt.Sprint(val)})
				}
			}
		}
	}

	return values, firstErr
}

// parseVaultKey splits a key like "path/to/secret.key" into "path/to/secret" and "key".
func parseVaultKey(key string) (path, secretKey string) {
	if i := strings.LastIndex(key, "."); i != -1 {
		return key[:i], key[i+1:]
	}
	return "", ""
}