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

// ConsulSource manages the Consul configuration source.
type ConsulSource struct {
	mu       sync.RWMutex
	client   *consulapi.Client
	cache    map[string]consulCacheEntry
	cacheTTL time.Duration
}

// Consul provides access to the Consul configuration source.
var Consul ConsulSource

// Client returns the shared Consul client.
func (s *ConsulSource) Client() *consulapi.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

// Endpoint configures the shared Consul client.
// It reads the address from configuration (CONSUL_HTTP_ADDR).
func (s *ConsulSource) Endpoint() (*ConsulSource, error) {
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
func (s *ConsulSource) WithCache(ttl time.Duration) *ConsulSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cacheTTL = ttl
	if s.cache == nil {
		s.cache = make(map[string]consulCacheEntry)
	}
	return s
}

// GetServiceUrl retrieves the URL for a given service from Consul.
func (s *ConsulSource) GetServiceUrl(serviceName string) (string, error) {
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

// ServiceRegistration holds the details for registering a service with Consul.
type ServiceRegistration struct {
	ID      string
	Name    string
	Port    int
	Address string
	Tags    []string
	Check   *consulapi.AgentServiceCheck
}

// RegisterFromEnv builds a service registration from environment variables and registers it.
// It allows overriding the service name programmatically.
func (s *ConsulSource) RegisterFromEnv(name ...string) (string, error) {
	// Ensure the shared consul client is configured
	if _, err := s.Endpoint(); err != nil {
		return "", fmt.Errorf("consul client not configured: %w", err)
	}

	var consulEnv = Env.Helper("CONSUL")
	var serviceName = consulEnv.Get("SERVICE_NAME", Get("APP_NAME", "gostart-service").String()).String()
	if len(name) > 0 && name[0] != "" {
		serviceName = name[0]
	}

	var serviceHost = consulEnv.Get("SERVICE_HOST", Get("APP_HOST", "localhost").String()).String()
	var servicePort = consulEnv.Get("SERVICE_PORT", Get("HTTP_PORT", "8080").String()).Int()

	var serviceID string
	if id := consulEnv.Get("SERVICE_ID"); id.Valid() {
		serviceID = id.String()
	} else if autoId := consulEnv.Get("SERVICE_ID_AUTO"); autoId.Bool() {
		serviceID = fmt.Sprintf("%s-%s-%d", serviceName, serviceHost, servicePort)
	} else {
		serviceID = serviceName
	}

	reg := ServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Port:    servicePort,
		Address: serviceHost,
		Tags:    []string{Get("APP_VERSION", "latest").String(), serviceName, serviceHost},
		Check: &consulapi.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://%s:%d/ok", serviceHost, servicePort),
			Interval: consulEnv.Get("CHECK_INTERVAL", "10s").String(),
			Timeout:  consulEnv.Get("CHECK_TIMEOUT", "5s").String(),
		},
	}

	return s.Register(reg)
}

// Register registers a service with Consul using the provided details.
// It returns the service ID used for registration.
func (s *ConsulSource) Register(reg ServiceRegistration) (string, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return "", fmt.Errorf("consul client not initialized, call Endpoint() first")
	}

	serviceID := reg.ID
	if serviceID == "" {
		serviceID = reg.Name
	}

	if err := client.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
		ID:      serviceID,
		Name:    reg.Name,
		Port:    reg.Port,
		Address: reg.Address,
		Tags:    reg.Tags,
		Check:   reg.Check,
	}); err != nil {
		return "", fmt.Errorf("failed to register service with consul: %w", err)
	}

	return serviceID, nil
}

// Deregister deregisters a service from Consul.
func (s *ConsulSource) Deregister(serviceID string) error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		// If the client is nil, we can't deregister, but we shouldn't block shutdown.
		// This might happen if Consul was never available in the first place.
		return nil
	}

	if serviceID == "" {
		return fmt.Errorf("cannot deregister service: service ID is empty")
	}

	if err := client.Agent().ServiceDeregister(serviceID); err != nil {
		return fmt.Errorf("failed to deregister service '%s' from consul: %w", serviceID, err)
	}

	return nil
}
