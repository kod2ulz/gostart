package config

import (
	
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
)

// vaultCacheEntry holds a cached value and its expiration time.
type vaultCacheEntry struct {
	value      string
	expiration time.Time
}

// vaultSource manages the Vault configuration source.
type vaultSource struct {
	mu       sync.RWMutex
	client   *api.Client
	cache    map[string]vaultCacheEntry
	cacheTTL time.Duration
}

// Vault provides access to the Vault configuration source.
var Vault vaultSource

// Endpoint configures the Vault client with an address and token.
func (s *vaultSource) Endpoint(addr, token string) (*vaultSource, error) {
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
	return s, nil
}

// WithCache sets the cache TTL for Vault-retrieved values.
func (s *vaultSource) WithCache(ttl time.Duration) *vaultSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheTTL = ttl
	if s.cache == nil {
		s.cache = make(map[string]vaultCacheEntry)
	}
	return s
}

// Get retrieves a secret from Vault.
// The key is expected to be in the format "path/to/secret.key-in-secret".
func (s *vaultSource) Get(key string, defaultValue ...interface{}) Value {
	s.mu.RLock()
	useCache := s.cache != nil && s.cacheTTL > 0
	if useCache {
		if entry, found := s.cache[key]; found && time.Now().Before(entry.expiration) {
			s.mu.RUnlock()
			return Value(entry.value)
		}
	}
	s.mu.RUnlock()

	// Value not in cache or expired, query Vault
	if s.client != nil {
		path, secretKey := parseVaultKey(key)
		if path != "" && secretKey != "" {
			secret, err := s.client.Logical().Read(path)
			if err == nil && secret != nil && secret.Data != nil {
				if data, ok := secret.Data["data"].(map[string]interface{}); ok {
					if val, found := data[secretKey]; found {
							vStr := fmt.Sprint(val)
							// Found in Vault, update cache if enabled
							s.mu.Lock()
							if useCache {
								s.cache[key] = vaultCacheEntry{
									value:      vStr,
									expiration: time.Now().Add(s.cacheTTL),
								}
							}
							s.mu.Unlock()
							return Value(vStr)
					}
				}
			}
		}
	}

	// Not found or client not configured, use default
	if len(defaultValue) > 0 {
		return Value(fmt.Sprint(defaultValue[0]))
	}

	return ""
}

// parseVaultKey splits a key like "path/to/secret.key" into "path/to/secret" and "key".
func parseVaultKey(key string) (path, secretKey string) {
	if i := strings.LastIndex(key, "."); i != -1 {
		return key[:i], key[i+1:]
	}
	return "", ""
}
