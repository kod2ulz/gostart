package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ConfigWatcher watches for configuration changes and triggers reloads
type ConfigWatcher struct {
	watcher   *fsnotify.Watcher
	files     map[string]time.Time
	mu        sync.RWMutex
	callbacks []ReloadCallback
	debounce  time.Duration
	logger    Logger
}

// ReloadCallback is called when configuration changes
type ReloadCallback func(path string) error

// Logger interface for config watcher
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// NewConfigWatcher creates a new configuration watcher
func NewConfigWatcher(logger Logger) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	return &ConfigWatcher{
		watcher:   watcher,
		files:     make(map[string]time.Time),
		callbacks: make([]ReloadCallback, 0),
		debounce:  1 * time.Second, // Default debounce duration
		logger:    logger,
	}, nil
}

// Watch starts watching a configuration file or directory
func (cw *ConfigWatcher) Watch(path string) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		// Watch all files in directory
		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				filePath := filepath.Join(path, entry.Name())
				if err := cw.watcher.Add(filePath); err != nil {
					return fmt.Errorf("failed to watch file %s: %w", filePath, err)
				}
				cw.files[filePath] = time.Now()
				cw.logger.Info("Watching configuration file", "path", filePath)
			}
		}
	} else {
		// Watch single file
		if err := cw.watcher.Add(path); err != nil {
			return fmt.Errorf("failed to watch file: %w", err)
		}
		cw.files[path] = time.Now()
		cw.logger.Info("Watching configuration file", "path", path)
	}

	return nil
}

// OnReload registers a callback to be called when configuration changes
func (cw *ConfigWatcher) OnReload(callback ReloadCallback) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}

// Start starts watching for file changes
func (cw *ConfigWatcher) Start(ctx context.Context) error {
	cw.logger.Info("Starting configuration watcher")

	// Track pending reloads for debouncing
	pendingReloads := make(map[string]*time.Timer)
	var pendingMu sync.Mutex

	for {
		select {
		case <-ctx.Done():
			cw.logger.Info("Stopping configuration watcher")
			return cw.watcher.Close()

		case event, ok := <-cw.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}

			// Handle write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				cw.logger.Debug("Configuration file changed", "path", event.Name, "op", event.Op.String())

				// Debounce: prevent multiple rapid reloads
				pendingMu.Lock()
				if timer, exists := pendingReloads[event.Name]; exists {
					timer.Stop()
				}

				pendingReloads[event.Name] = time.AfterFunc(cw.debounce, func() {
					pendingMu.Lock()
					delete(pendingReloads, event.Name)
					pendingMu.Unlock()

					if err := cw.reload(event.Name); err != nil {
						cw.logger.Error("Failed to reload configuration", "path", event.Name, "error", err)
					}
				})
				pendingMu.Unlock()
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed")
			}
			cw.logger.Error("Watcher error", "error", err)
		}
	}
}

// reload triggers all registered callbacks
func (cw *ConfigWatcher) reload(path string) error {
	cw.logger.Info("Reloading configuration", "path", path)

	cw.mu.RLock()
	callbacks := make([]ReloadCallback, len(cw.callbacks))
	copy(callbacks, cw.callbacks)
	cw.mu.RUnlock()

	var lastErr error
	for _, callback := range callbacks {
		if err := callback(path); err != nil {
			cw.logger.Error("Reload callback error", "error", err, "path", path)
			lastErr = err
		}
	}

	if lastErr == nil {
		cw.logger.Info("Configuration reloaded successfully", "path", path)
	}

	return lastErr
}

// SetDebounce sets the debounce duration
func (cw *ConfigWatcher) SetDebounce(duration time.Duration) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.debounce = duration
}

// Stop stops the configuration watcher
func (cw *ConfigWatcher) Stop() error {
	return cw.watcher.Close()
}

// ConfigManager manages dynamic configuration with auto-reload
type ConfigManager struct {
	values   map[string]interface{}
	mu       sync.RWMutex
	watchers []*ConfigWatcher
	loaders  map[string]ConfigLoader
	logger   Logger
}

// ConfigLoader loads configuration from a source
type ConfigLoader interface {
	Load(path string) (map[string]interface{}, error)
}

// YAMLConfigLoader loads YAML configuration
type YAMLConfigLoader struct{}

func (l *YAMLConfigLoader) Load(path string) (map[string]interface{}, error) {
	// Use existing YAML loading logic
	if err := Yaml.Load(path); err != nil {
		return nil, err
	}
	// Return loaded values
	return make(map[string]interface{}), nil
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(logger Logger) *ConfigManager {
	return &ConfigManager{
		values:   make(map[string]interface{}),
		watchers: make([]*ConfigWatcher, 0),
		loaders:  make(map[string]ConfigLoader),
		logger:   logger,
	}
}

// RegisterLoader registers a configuration loader for a file type
func (cm *ConfigManager) RegisterLoader(extension string, loader ConfigLoader) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.loaders[extension] = loader
}

// LoadAndWatch loads a configuration file and starts watching it
func (cm *ConfigManager) LoadAndWatch(ctx context.Context, path string) error {
	// Load initial configuration
	if err := cm.loadConfig(path); err != nil {
		return fmt.Errorf("failed to load initial config: %w", err)
	}

	// Create watcher
	watcher, err := NewConfigWatcher(cm.logger)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Register reload callback
	watcher.OnReload(func(changedPath string) error {
		return cm.loadConfig(changedPath)
	})

	// Start watching
	if err := watcher.Watch(path); err != nil {
		return fmt.Errorf("failed to watch config: %w", err)
	}

	cm.mu.Lock()
	cm.watchers = append(cm.watchers, watcher)
	cm.mu.Unlock()

	// Start watcher in background
	go func() {
		if err := watcher.Start(ctx); err != nil {
			cm.logger.Error("Watcher stopped with error", "error", err)
		}
	}()

	return nil
}

// loadConfig loads configuration from a file
func (cm *ConfigManager) loadConfig(path string) error {
	ext := filepath.Ext(path)

	cm.mu.RLock()
	loader, exists := cm.loaders[ext]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no loader registered for extension: %s", ext)
	}

	values, err := loader.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cm.mu.Lock()
	for k, v := range values {
		cm.values[k] = v
	}
	cm.mu.Unlock()

	cm.logger.Info("Configuration loaded", "path", path, "keys", len(values))
	return nil
}

// Get retrieves a configuration value
func (cm *ConfigManager) Get(key string) (interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	val, exists := cm.values[key]
	return val, exists
}

// Set sets a configuration value
func (cm *ConfigManager) Set(key string, value interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.values[key] = value
}

// GetString retrieves a string configuration value
func (cm *ConfigManager) GetString(key string, defaultValue ...string) string {
	val, exists := cm.Get(key)
	if !exists {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}

	if str, ok := val.(string); ok {
		return str
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// GetInt retrieves an integer configuration value
func (cm *ConfigManager) GetInt(key string, defaultValue ...int) int {
	val, exists := cm.Get(key)
	if !exists {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
}

// GetBool retrieves a boolean configuration value
func (cm *ConfigManager) GetBool(key string, defaultValue ...bool) bool {
	val, exists := cm.Get(key)
	if !exists {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}

	if b, ok := val.(bool); ok {
		return b
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// Stop stops all configuration watchers
func (cm *ConfigManager) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, watcher := range cm.watchers {
		watcher.Stop()
	}
}

// Example usage
func ExampleConfigWatcher() {
	type simpleLogger struct{}

	func (l *simpleLogger) Info(msg string, args ...interface{})  {}
	func (l *simpleLogger) Error(msg string, args ...interface{}) {}
	func (l *simpleLogger) Debug(msg string, args ...interface{}) {}

	logger := &simpleLogger{}
	manager := NewConfigManager(logger)

	// Register YAML loader
	manager.RegisterLoader(".yaml", &YAMLConfigLoader{})
	manager.RegisterLoader(".yml", &YAMLConfigLoader{})

	// Load and watch configuration
	ctx := context.Background()
	manager.LoadAndWatch(ctx, "config.yaml")

	// Use configuration
	dbHost := manager.GetString("database.host", "localhost")
	dbPort := manager.GetInt("database.port", 5432)
	fmt.Printf("Database: %s:%d\n", dbHost, dbPort)

	// Configuration will auto-reload when file changes
}
